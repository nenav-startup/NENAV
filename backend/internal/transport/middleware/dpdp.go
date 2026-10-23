package middleware

import (
	"context"
	"net/http"
	"strings"
)

type dpdpContextKey string

const (
	DPDPConsentKey dpdpContextKey = "dpdp_consents"
)

// DPDPConsents stores client-asserted or token-resolved consent flags.
type DPDPConsents struct {
	AnalyticsConsent bool
	LocationConsent  bool
	MarketingConsent bool
}

func isConsentGranted(val string) bool {
	return strings.EqualFold(val, "true") || strings.EqualFold(val, "granted") || strings.EqualFold(val, "1")
}

// DPDPMiddleware parses statutory privacy headers sent by NENAV clients (e.g. Flutter mobile apps).
// Headers:
// - X-Nenav-Analytics-Consent: "true" | "granted" | "false" | "denied"
// - X-Nenav-Location-Consent: "true" | "granted" | "false" | "denied"
// - X-Nenav-Marketing-Consent: "true" | "granted" | "false" | "denied"
func DPDPMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		consents := DPDPConsents{
			AnalyticsConsent: isConsentGranted(r.Header.Get("X-Nenav-Analytics-Consent")),
			LocationConsent:  isConsentGranted(r.Header.Get("X-Nenav-Location-Consent")),
			MarketingConsent: isConsentGranted(r.Header.Get("X-Nenav-Marketing-Consent")),
		}

		ctx := context.WithValue(r.Context(), DPDPConsentKey, consents)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// GetDPDPConsents extracts parsed consent flags from request context.
func GetDPDPConsents(ctx context.Context) DPDPConsents {
	if val, ok := ctx.Value(DPDPConsentKey).(DPDPConsents); ok {
		return val
	}
	return DPDPConsents{}
}
