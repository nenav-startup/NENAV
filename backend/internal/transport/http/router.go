// Package router defines the HTTP route tree for the NENAV API.
//
// Middleware is applied deliberately per route group. Public health endpoints
// have minimal middleware; API routes have the full security stack.
//
// Reference: Master Prompt §11 — API Design, §14 — Security Middleware
package router

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"

	"github.com/nenav/nenav/backend/internal/config"
	"github.com/nenav/nenav/backend/internal/domain/dispatch"
	"github.com/nenav/nenav/backend/internal/events"
	"github.com/nenav/nenav/backend/internal/platform/cache"
	redisdispatch "github.com/nenav/nenav/backend/internal/platform/dispatch"
	mockgateway "github.com/nenav/nenav/backend/internal/platform/payment"
	"github.com/nenav/nenav/backend/internal/repository"
	"github.com/nenav/nenav/backend/internal/service"
	"github.com/nenav/nenav/backend/internal/transport/http/handler"
	"github.com/nenav/nenav/backend/internal/transport/middleware"
)

// Deps holds the dependencies needed to build the router.
type Deps struct {
	Config *config.Config
	Logger *slog.Logger
	DB     handler.Pinger
	Cache  handler.Pinger
	Store  repository.Store
}

// New creates the HTTP handler with all routes and middleware configured.
func New(deps Deps) http.Handler {
	r := chi.NewRouter()

	// --- Global middleware (order matters: outermost first) ---
	r.Use(middleware.RequestID)
	r.Use(middleware.Recovery(deps.Logger))
	r.Use(middleware.Logger(deps.Logger))
	r.Use(middleware.SecurityHeaders)
	r.Use(chimw.RealIP)
	r.Use(chimw.Compress(5))
	r.Use(middleware.BodySizeLimit(1 << 20)) // 1 MB
	r.Use(middleware.DPDPMiddleware)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   deps.Config.CORS.AllowedOrigins,
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-Request-ID", "X-Trace-ID", "X-Idempotency-Key"},
		ExposedHeaders:   []string{"X-Request-ID", "X-Trace-ID"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	// --- Health / Infrastructure endpoints (no auth required) ---
	healthHandler := handler.NewHealth(deps.DB, deps.Cache)
	r.Get("/health", healthHandler.HandleHealth)
	r.Get("/ready", healthHandler.HandleReady)
	r.Get("/live", healthHandler.HandleLive)

	// --- Initialize Services ---
	store := deps.Store
	if store == nil {
		store = repository.NewMemoryStore()
	}

	eventBus := events.NewBus(deps.Logger)
	matchingSvc := service.NewMatchingService(store, eventBus)
	
	// Payment & Dispatch
	var dispatcher dispatch.Dispatcher
	if deps.Cache != nil {
		if r, ok := deps.Cache.(*cache.Redis); ok {
			dispatcher = redisdispatch.NewRedisDispatcher(r, deps.Logger)
			// (A real deployment would start the subscriber loop here or in main)
		}
	}
	if dispatcher == nil {
		dispatcher = redisdispatch.NewMockDispatcher(matchingSvc)
	}

	gateway := mockgateway.NewMockRazorpayGateway()

	authSvc := service.NewAuthService(store, eventBus, "")
	paymentSvc := service.NewPaymentService(store, dispatcher, gateway, eventBus)
	bookingSvc := service.NewBookingService(store, dispatcher, paymentSvc, eventBus)
	supportSvc := service.NewSupportService(store, paymentSvc, eventBus)
	privacySvc := service.NewPrivacyService(store, eventBus)

	authHandler := handler.NewAuthHandler(authSvc)
	bookingHandler := handler.NewBookingHandler(bookingSvc)
	lifecycleHandler := handler.NewJobLifecycleHandler(bookingSvc)
	paymentHandler := handler.NewPaymentHandler(paymentSvc)
	sreHandler := handler.NewSREHandler(store, matchingSvc)
	profHandler := handler.NewProfHandler(store)
	supportHandler := handler.NewSupportHandler(supportSvc)
	privacyHandler := handler.NewPrivacyHandler(privacySvc)
	realtimeHandler := handler.NewRealtimeHandler(eventBus, deps.Logger)

	// --- API v1 ---
	r.Route("/api/v1", func(api chi.Router) {
		// API root
		api.Get("/", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"service":"nenav-api","version":"0.1.0","status":"operational"}`))
		})

		// --- Real-time SSE channel stream (Master Prompt §05, §07, §08, §09, §44) ---
		// Streaming endpoints are exempt from the standard HTTP request timeout.
		api.Get("/realtime/stream", realtimeHandler.HandleStream)

		// Timed routes group for request-response APIs
		api.Group(func(timed chi.Router) {
			timed.Use(chimw.Timeout(deps.Config.App.ShutdownTimeout))

			// --- Auth routes ---
			timed.Route("/auth", func(r chi.Router) {
				r.Post("/otp/send", authHandler.HandleSendOTP)
				r.Post("/otp/verify", authHandler.HandleVerifyOTP)
			})

			// --- Professional routes ---
			timed.Route("/profs", func(r chi.Router) {
				r.Get("/", profHandler.HandleListPROFs)
				r.Get("/{id}", profHandler.HandleGetPROF)
			})

			// --- Protected Routes ---
			timed.Group(func(protected chi.Router) {
				protected.Use(middleware.RequireAuth(store))

			// --- Booking & Job routes ---
			protected.Route("/bookings", func(r chi.Router) {
				r.Post("/quote", bookingHandler.HandleQuote)
				r.Post("/", bookingHandler.HandleCreateBooking)
				r.Get("/", bookingHandler.HandleListBookings)
				r.Get("/{id}", bookingHandler.HandleGetBooking)
				r.Post("/{id}/cancel", bookingHandler.HandleCancelBooking)

				// Lifecycle transitions & real-time actions
				r.Post("/{id}/claim", lifecycleHandler.HandleClaim)
				r.Post("/{id}/en-route", lifecycleHandler.HandleEnRoute)
				r.Post("/{id}/arrived", lifecycleHandler.HandleArrived)
				r.Post("/{id}/checkin", lifecycleHandler.HandleCheckin)
				r.Post("/{id}/additional-work", lifecycleHandler.HandleAdditionalWork)
				r.Post("/{id}/additional-work/approve", lifecycleHandler.HandleApproveAdditionalWork)
				r.Post("/{id}/additional-work/decline", lifecycleHandler.HandleDeclineAdditionalWork)
				r.Post("/{id}/location", lifecycleHandler.HandleLocation)
				r.Post("/{id}/complete", lifecycleHandler.HandleComplete)
			})

			// --- Payment routes ---
			protected.Route("/payments", func(r chi.Router) {
				r.Post("/online-confirm", paymentHandler.HandleOnlineConfirm)
				r.Post("/cash-confirm", paymentHandler.HandleCashConfirm)
			})

			// --- Support & Dispute routes ---
			protected.Route("/disputes", func(r chi.Router) {
				r.Post("/", supportHandler.HandleCreateDispute)
				r.Get("/", supportHandler.HandleListDisputes)
				r.Post("/{id}/resolve", supportHandler.HandleResolveDispute)
			})

			// --- DPDP Privacy & Statutory Rights routes ---
			protected.Route("/privacy", func(r chi.Router) {
				r.Post("/consents", privacyHandler.HandleRecordConsent)
				r.Get("/consents", privacyHandler.HandleGetConsents)
				r.Post("/dsr/export", privacyHandler.HandleExportData)
				r.Post("/dsr/erasure", privacyHandler.HandleErasure)
				r.Get("/dsr", privacyHandler.HandleListDSRs)
				r.Post("/grievances", privacyHandler.HandleFileGrievance)
				r.Get("/grievances", privacyHandler.HandleListGrievances)
				r.Post("/nominee", privacyHandler.HandleSetNominee)
				r.Get("/nominee", privacyHandler.HandleGetNominee)
				r.Delete("/nominee", privacyHandler.HandleRevokeNominee)
			})

			// --- SRE / Admin Ops routes ---
			protected.Route("/ops", func(r chi.Router) {
				r.Use(middleware.RequireAdmin(store))
				r.Get("/bookings/stuck", sreHandler.HandleGetStuckBookings)
				r.Post("/bookings/{id}/force-assign", sreHandler.HandleForceAssign)
				r.Post("/privacy/grievances/{id}/resolve", privacyHandler.HandleResolveGrievance)
				r.Post("/privacy/breaches", privacyHandler.HandleReportBreach)
				r.Get("/privacy/breaches", privacyHandler.HandleListBreaches)
				r.Post("/privacy/breaches/{id}/report-dpbi", privacyHandler.HandleReportBreachToDPBI)
			})
		})
	})
})

	return r
}
