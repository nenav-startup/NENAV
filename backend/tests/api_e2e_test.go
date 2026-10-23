package tests

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/nenav/nenav/backend/internal/config"
	"github.com/nenav/nenav/backend/internal/observability"
	"github.com/nenav/nenav/backend/internal/repository"
	router "github.com/nenav/nenav/backend/internal/transport/http"
)

type dummyPinger struct{}

func (d *dummyPinger) Ping(ctx interface{}) error { return nil }

func setupTestServer() *httptest.Server {
	cfg := &config.Config{}
	cfg.App.Port = "8080"
	cfg.App.ShutdownTimeout = 15 * time.Second
	cfg.CORS.AllowedOrigins = []string{"*"}

	logger := observability.NewLogger("error", nil)
	store := repository.NewMemoryStore()

	handler := router.New(router.Deps{
		Config: cfg,
		Logger: logger,
		DB:     nil,
		Cache:  nil,
		Store:  store,
	})

	return httptest.NewServer(handler)
}

func TestFullBookingAndPaymentE2E(t *testing.T) {
	ts := setupTestServer()
	defer ts.Close()

	client := ts.Client()

	// Helper for auth requests
	doAuthReq := func(method, path string, body []byte, token string) (*http.Response, error) {
		var req *http.Request
		var err error
		if body != nil {
			req, err = http.NewRequest(method, ts.URL+path, bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
		} else {
			req, err = http.NewRequest(method, ts.URL+path, nil)
		}
		if err != nil {
			return nil, err
		}
		req.Header.Set("Authorization", "Bearer "+token)
		return client.Do(req)
	}

	// 1. Health Probe
	resp, err := client.Get(ts.URL + "/health")
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("health check failed: %v", err)
	}

	// 2. Request OTP
	otpReqBody, _ := json.Marshal(map[string]string{
		"phone":        "9849099887",
		"country_code": "+91",
		"purpose":      "login",
	})
	resp, err = client.Post(ts.URL+"/api/v1/auth/otp/send", "application/json", bytes.NewReader(otpReqBody))
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("send otp failed: %v", err)
	}

	code := "123456"

	// 3. Verify OTP
	verifyReqBody, _ := json.Marshal(map[string]string{
		"phone":        "9849099887",
		"country_code": "+91",
		"code":         code,
	})
	resp, err = client.Post(ts.URL+"/api/v1/auth/otp/verify", "application/json", bytes.NewReader(verifyReqBody))
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("verify otp failed: %v", err)
	}

	var authResp struct {
		Data struct {
			Token string `json:"token"`
			User  struct {
				ID string `json:"id"`
			} `json:"user"`
		} `json:"data"`
	}
	json.NewDecoder(resp.Body).Decode(&authResp)
	token := authResp.Data.Token
	userID := authResp.Data.User.ID
	if token == "" || userID == "" {
		t.Fatalf("expected token and user ID, got token=%s, user=%s", token, userID)
	}

	// 4. Request Dynamic Pricing Quote
	tradeElectrician := "10000000-0000-0000-0000-000000000001"
	quoteReqBody, _ := json.Marshal(map[string]any{
		"trade_id":     tradeElectrician,
		"hours":        2,
		"worker_count": 1,
	})
	resp, err = doAuthReq("POST", "/api/v1/bookings/quote", quoteReqBody, token)
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("quote request failed: %v", err)
	}

	var quoteResp struct {
		Data struct {
			TotalAmount int `json:"total_amount"`
		} `json:"data"`
	}
	json.NewDecoder(resp.Body).Decode(&quoteResp)
	if quoteResp.Data.TotalAmount != 691 {
		t.Errorf("expected ₹691, got %d", quoteResp.Data.TotalAmount)
	}

	// 5. Create Booking
	createBookingBody, _ := json.Marshal(map[string]any{
		"customer_id":     userID,
		"trade_id":        tradeElectrician,
		"trade_name":      "Electrician",
		"scheduled_date":  "2026-09-10",
		"time_slot":       "09:00 AM - 11:00 AM",
		"duration_type":   "hourly",
		"hours":           2,
		"worker_count":    1,
		"address_line":    "Flat 402, Banjara Hills, Hyderabad",
		"job_description": "Ceiling fan installation and circuit repair",
		"payment_method":  "cash",
	})
	resp, err = doAuthReq("POST", "/api/v1/bookings", createBookingBody, token)
	if err != nil || resp.StatusCode != http.StatusCreated {
		t.Fatalf("create booking failed: %v", err)
	}

	var bookingResp struct {
		Data struct {
			ID               string `json:"id"`
			BookingReference string `json:"booking_reference"`
			Status           string `json:"status"`
			CheckinOTP       string `json:"checkin_otp"`
		} `json:"data"`
	}
	json.NewDecoder(resp.Body).Decode(&bookingResp)
	bookingID := bookingResp.Data.ID
	if bookingID == "" {
		t.Fatalf("missing booking ID in creation response")
	}

	// 6. Test Worker Lifecycle: En Route
	resp, err = doAuthReq("POST", "/api/v1/bookings/"+bookingID+"/en-route", nil, token)
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("advance to en-route failed: %v", err)
	}

	// 7. Test Worker Lifecycle: Arrived
	resp, err = doAuthReq("POST", "/api/v1/bookings/"+bookingID+"/arrived", nil, token)
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("advance to arrived failed: %v", err)
	}

	// 8. Test Mutual Check-in OTP verification
	checkinBody, _ := json.Marshal(map[string]string{"otp": "4829"})
	resp, err = doAuthReq("POST", "/api/v1/bookings/"+bookingID+"/checkin", checkinBody, token)
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("checkin OTP verification failed: %v", err)
	}

	// 9. Complete Job
	resp, err = doAuthReq("POST", "/api/v1/bookings/"+bookingID+"/complete", nil, token)
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("complete job failed: %v", err)
	}

	// 10. Bilateral Cash Settlement
	// Customer confirmation
	custCashBody, _ := json.Marshal(map[string]any{"booking_id": bookingID, "is_customer": true})
	resp, err = doAuthReq("POST", "/api/v1/payments/cash-confirm", custCashBody, token)
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("customer cash confirm failed: %v", err)
	}

	// Worker confirmation
	profCashBody, _ := json.Marshal(map[string]any{"booking_id": bookingID, "is_customer": false})
	resp, err = doAuthReq("POST", "/api/v1/payments/cash-confirm", profCashBody, token)
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("prof cash confirm failed: %v", err)
	}

	var cashResp struct {
		Data struct {
			BothConfirmed bool `json:"both_confirmed"`
		} `json:"data"`
	}
	json.NewDecoder(resp.Body).Decode(&cashResp)
	if !cashResp.Data.BothConfirmed {
		t.Errorf("expected both_confirmed true, got false")
	}

	// 11. Verify Final Booking State
	resp, err = doAuthReq("GET", "/api/v1/bookings/"+bookingID, nil, token)
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("get booking failed: %v", err)
	}

	var finalResp struct {
		Data struct {
			Status        string `json:"status"`
			PaymentStatus string `json:"payment_status"`
		} `json:"data"`
	}
	json.NewDecoder(resp.Body).Decode(&finalResp)
	if finalResp.Data.Status != "completed" {
		t.Errorf("expected final status completed, got %s", finalResp.Data.Status)
	}
	if finalResp.Data.PaymentStatus != "completed" {
		t.Errorf("expected final payment status completed, got %s", finalResp.Data.PaymentStatus)
	}
}

func init() {
	_ = uuid.Nil
}
