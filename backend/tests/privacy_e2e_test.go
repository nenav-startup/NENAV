package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"math"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/nenav/nenav/backend/internal/domain/identity"
	"github.com/nenav/nenav/backend/internal/domain/privacy"
	"github.com/nenav/nenav/backend/internal/jobs"
	"github.com/nenav/nenav/backend/internal/repository"
	"github.com/nenav/nenav/backend/internal/transport/middleware"
)

func TestConsentLedgerImmutabilityAndResolution(t *testing.T) {
	ts := setupTestServer()
	defer ts.Close()

	client := ts.Client()

	// 1. Authenticate demo user
	otpReqBody, _ := json.Marshal(map[string]string{
		"phone":        "9111223344",
		"country_code": "+91",
		"purpose":      "login",
	})
	resp, err := client.Post(ts.URL+"/api/v1/auth/otp/send", "application/json", bytes.NewReader(otpReqBody))
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("send otp failed: %v", err)
	}

	verifyReqBody, _ := json.Marshal(map[string]string{
		"phone":        "9111223344",
		"country_code": "+91",
		"code":         "123456",
	})
	resp, err = client.Post(ts.URL+"/api/v1/auth/otp/verify", "application/json", bytes.NewReader(verifyReqBody))
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("verify otp failed: %v", err)
	}

	var authResp struct {
		Data struct {
			Token string `json:"token"`
		} `json:"data"`
	}
	json.NewDecoder(resp.Body).Decode(&authResp)
	token := authResp.Data.Token

	doAuthReq := func(method, path string, body []byte) (*http.Response, error) {
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

	// 2. Grant Analytics Consent
	grantAnalytics, _ := json.Marshal(map[string]any{
		"purpose":        "analytics",
		"status":         "granted",
		"notice_version": "v1.0.0",
		"metadata":       map[string]any{"source": "prof-mobile-onboarding"},
	})
	resp, err = doAuthReq("POST", "/api/v1/privacy/consents", grantAnalytics)
	if err != nil || resp.StatusCode != http.StatusCreated {
		t.Fatalf("grant analytics consent failed: %v, status: %d", err, resp.StatusCode)
	}

	// 3. Grant Precise Location Consent
	grantLocation, _ := json.Marshal(map[string]any{
		"purpose":        "precise_location",
		"status":         "granted",
		"notice_version": "v1.0.0",
	})
	resp, err = doAuthReq("POST", "/api/v1/privacy/consents", grantLocation)
	if err != nil || resp.StatusCode != http.StatusCreated {
		t.Fatalf("grant location consent failed: %v, status: %d", err, resp.StatusCode)
	}

	// 4. Verify Active Consents
	resp, err = doAuthReq("GET", "/api/v1/privacy/consents", nil)
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("get consents failed: %v", err)
	}

	var consentsResp struct {
		Data struct {
			ActiveConsents map[string]bool         `json:"active_consents"`
			History        []privacy.ConsentRecord `json:"history"`
		} `json:"data"`
	}
	json.NewDecoder(resp.Body).Decode(&consentsResp)

	if !consentsResp.Data.ActiveConsents["analytics"] {
		t.Errorf("expected analytics consent to be true")
	}
	if !consentsResp.Data.ActiveConsents["precise_location"] {
		t.Errorf("expected precise_location consent to be true")
	}
	if len(consentsResp.Data.History) != 2 {
		t.Errorf("expected 2 history entries, got %d", len(consentsResp.Data.History))
	}

	// 5. Revoke Precise Location Consent
	revokeLocation, _ := json.Marshal(map[string]any{
		"purpose":        "precise_location",
		"status":         "revoked",
		"notice_version": "v1.0.0",
	})
	resp, err = doAuthReq("POST", "/api/v1/privacy/consents", revokeLocation)
	if err != nil || resp.StatusCode != http.StatusCreated {
		t.Fatalf("revoke location consent failed: %v", err)
	}

	// 6. Verify Active Consents again: analytics=true, precise_location=false, history=3
	resp, err = doAuthReq("GET", "/api/v1/privacy/consents", nil)
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("get consents after revoke failed: %v", err)
	}
	json.NewDecoder(resp.Body).Decode(&consentsResp)

	if !consentsResp.Data.ActiveConsents["analytics"] {
		t.Errorf("expected analytics consent to remain true")
	}
	if consentsResp.Data.ActiveConsents["precise_location"] {
		t.Errorf("expected precise_location consent to be revoked (false)")
	}
	if len(consentsResp.Data.History) != 3 {
		t.Errorf("expected 3 history entries in immutable audit trail, got %d", len(consentsResp.Data.History))
	}
}

func TestDataSubjectRightExportAndErasure(t *testing.T) {
	ts := setupTestServer()
	defer ts.Close()

	client := ts.Client()

	// 1. Authenticate user
	otpReqBody, _ := json.Marshal(map[string]string{
		"phone":        "9777888999",
		"country_code": "+91",
		"purpose":      "login",
	})
	client.Post(ts.URL+"/api/v1/auth/otp/send", "application/json", bytes.NewReader(otpReqBody))

	verifyReqBody, _ := json.Marshal(map[string]string{
		"phone":        "9777888999",
		"country_code": "+91",
		"code":         "123456",
	})
	resp, _ := client.Post(ts.URL+"/api/v1/auth/otp/verify", "application/json", bytes.NewReader(verifyReqBody))

	var authResp struct {
		Data struct {
			Token string `json:"token"`
		} `json:"data"`
	}
	json.NewDecoder(resp.Body).Decode(&authResp)
	token := authResp.Data.Token

	doAuthReq := func(method, path string, body []byte) (*http.Response, error) {
		var req *http.Request
		if body != nil {
			req, _ = http.NewRequest(method, ts.URL+path, bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
		} else {
			req, _ = http.NewRequest(method, ts.URL+path, nil)
		}
		req.Header.Set("Authorization", "Bearer "+token)
		return client.Do(req)
	}

	// 2. Test Section 11: Right to Data Portability / Export
	resp, err := doAuthReq("POST", "/api/v1/privacy/dsr/export", nil)
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("export data failed: %v, status: %d", err, resp.StatusCode)
	}

	var exportResp struct {
		Data struct {
			ExportID        uuid.UUID `json:"export_id"`
			ExportTimestamp time.Time `json:"export_timestamp"`
			User            struct {
				Phone string `json:"phone"`
			} `json:"user"`
		} `json:"data"`
	}
	json.NewDecoder(resp.Body).Decode(&exportResp)
	if exportResp.Data.ExportID == uuid.Nil {
		t.Errorf("expected valid export ID")
	}
	if exportResp.Data.User.Phone != "9777888999" {
		t.Errorf("expected user phone 9777888999 in export, got %s", exportResp.Data.User.Phone)
	}

	// 3. Test Section 12: Right to Erasure with 72h SLA
	erasureReq, _ := json.Marshal(map[string]string{
		"reason": "User closing account and requesting DPDP personal data erasure",
	})
	resp, err = doAuthReq("POST", "/api/v1/privacy/dsr/erasure", erasureReq)
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("erasure request failed: %v, status: %d", err, resp.StatusCode)
	}

	var erasureResp struct {
		Data struct {
			Message     string    `json:"message"`
			DSRID       uuid.UUID `json:"dsr_id"`
			SLADeadline time.Time `json:"sla_deadline"`
			Status      string    `json:"status"`
		} `json:"data"`
	}
	json.NewDecoder(resp.Body).Decode(&erasureResp)

	// Verify statutory 72-hour SLA deadline
	slaDelta := erasureResp.Data.SLADeadline.Sub(time.Now())
	if slaDelta < 71*time.Hour || slaDelta > 73*time.Hour {
		t.Errorf("expected SLA deadline around 72 hours from now, got %v", slaDelta)
	}

	// 4. Verify Immediate Session Invalidation (Security & Privacy enforcement)
	resp, err = doAuthReq("GET", "/api/v1/privacy/consents", nil)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected 401 Unauthorized after account erasure, got status %d", resp.StatusCode)
	}
}

func TestGrievanceRedressalDPOWorkflow(t *testing.T) {
	ts := setupTestServer()
	defer ts.Close()

	client := ts.Client()

	// 1. Authenticate user
	otpReqBody, _ := json.Marshal(map[string]string{
		"phone":        "9555666777",
		"country_code": "+91",
		"purpose":      "login",
	})
	client.Post(ts.URL+"/api/v1/auth/otp/send", "application/json", bytes.NewReader(otpReqBody))

	verifyReqBody, _ := json.Marshal(map[string]string{
		"phone":        "9555666777",
		"country_code": "+91",
		"code":         "123456",
	})
	resp, _ := client.Post(ts.URL+"/api/v1/auth/otp/verify", "application/json", bytes.NewReader(verifyReqBody))

	var authResp struct {
		Data struct {
			Token string `json:"token"`
		} `json:"data"`
	}
	json.NewDecoder(resp.Body).Decode(&authResp)
	token := authResp.Data.Token

	doAuthReq := func(method, path string, body []byte) (*http.Response, error) {
		var req *http.Request
		if body != nil {
			req, _ = http.NewRequest(method, ts.URL+path, bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
		} else {
			req, _ = http.NewRequest(method, ts.URL+path, nil)
		}
		req.Header.Set("Authorization", "Bearer "+token)
		return client.Do(req)
	}

	// 2. File grievance to DPO under DPDP Section 13
	grievanceReq, _ := json.Marshal(map[string]string{
		"subject":     "Unsolicited marketing SMS received",
		"description": "I revoked marketing consent yesterday but received a promotional SMS today.",
	})
	resp, err := doAuthReq("POST", "/api/v1/privacy/grievances", grievanceReq)
	if err != nil || resp.StatusCode != http.StatusCreated {
		t.Fatalf("file grievance failed: %v, status: %d", err, resp.StatusCode)
	}

	var grievanceResp struct {
		Data privacy.Grievance `json:"data"`
	}
	json.NewDecoder(resp.Body).Decode(&grievanceResp)

	g := grievanceResp.Data
	if g.ID == uuid.Nil {
		t.Errorf("expected valid grievance ID")
	}
	if len(g.TicketNumber) < 10 {
		t.Errorf("expected formatted ticket number, got %s", g.TicketNumber)
	}
	if g.Status != privacy.GrievanceStatusOpen {
		t.Errorf("expected status open, got %s", g.Status)
	}

	// 3. User lists their grievances
	resp, err = doAuthReq("GET", "/api/v1/privacy/grievances", nil)
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("list grievances failed: %v", err)
	}

	var listResp struct {
		Data []privacy.Grievance `json:"data"`
	}
	json.NewDecoder(resp.Body).Decode(&listResp)
	if len(listResp.Data) != 1 {
		t.Errorf("expected 1 grievance in list, got %d", len(listResp.Data))
	}

	// 4. DPO resolves the grievance via Ops route
	resolveReq, _ := json.Marshal(map[string]string{
		"resolution_notes": "SMS service provider queue purged. Marketing opt-out strictly enforced.",
		"dpo_name":         "NENAV Chief Privacy Officer",
	})
	resp, err = doAuthReq("POST", "/api/v1/ops/privacy/grievances/"+g.ID.String()+"/resolve", resolveReq)
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("resolve grievance failed: %v, status: %d", err, resp.StatusCode)
	}

	var resolvedResp struct {
		Data privacy.Grievance `json:"data"`
	}
	json.NewDecoder(resp.Body).Decode(&resolvedResp)
	if resolvedResp.Data.Status != privacy.GrievanceStatusResolved {
		t.Errorf("expected resolved status, got %s", resolvedResp.Data.Status)
	}
	if resolvedResp.Data.ResolutionNotes == nil || *resolvedResp.Data.ResolutionNotes == "" {
		t.Errorf("expected resolution notes to be populated")
	}
}

func TestDPDPMiddlewareHeaders(t *testing.T) {
	var capturedConsents middleware.DPDPConsents

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedConsents = middleware.GetDPDPConsents(r.Context())
		w.WriteHeader(http.StatusOK)
	})

	handlerWithMW := middleware.DPDPMiddleware(testHandler)

	// Case 1: Headers with explicit opt-ins/opt-outs
	req := httptest.NewRequest("GET", "/api/v1/test", nil)
	req.Header.Set("X-Nenav-Analytics-Consent", "true")
	req.Header.Set("X-Nenav-Location-Consent", "false")
	req.Header.Set("X-Nenav-Marketing-Consent", "true")
	rec := httptest.NewRecorder()

	handlerWithMW.ServeHTTP(rec, req)

	if !capturedConsents.AnalyticsConsent {
		t.Errorf("expected AnalyticsConsent to be true")
	}
	if capturedConsents.LocationConsent {
		t.Errorf("expected LocationConsent to be false")
	}
	if !capturedConsents.MarketingConsent {
		t.Errorf("expected MarketingConsent to be true")
	}

	// Case 2: Headers absent (default to false under privacy by default)
	req2 := httptest.NewRequest("GET", "/api/v1/test", nil)
	rec2 := httptest.NewRecorder()

	handlerWithMW.ServeHTTP(rec2, req2)

	if capturedConsents.AnalyticsConsent || capturedConsents.LocationConsent || capturedConsents.MarketingConsent {
		t.Errorf("expected all consents to be false when headers are absent")
	}
}

func TestRetentionWorkerSweep(t *testing.T) {
	ctx := context.Background()
	store := repository.NewMemoryStore()

	// Seed expired OTP
	expiredOTP := &identity.OTPRequest{
		ID:               uuid.New(),
		Phone:            "9999000011",
		PhoneCountryCode: "+91",
		OTPHash:          "hash-expired",
		ExpiresAt:        time.Now().Add(-1 * time.Hour),
	}
	_ = store.CreateOTP(ctx, expiredOTP)

	// Seed valid active OTP
	activeOTP := &identity.OTPRequest{
		ID:               uuid.New(),
		Phone:            "9999000022",
		PhoneCountryCode: "+91",
		OTPHash:          "hash-active",
		ExpiresAt:        time.Now().Add(10 * time.Minute),
	}
	_ = store.CreateOTP(ctx, activeOTP)

	worker := jobs.NewRetentionWorker(store, nil)
	report, err := worker.RunSweep(ctx)
	if err != nil {
		t.Fatalf("retention sweep failed: %v", err)
	}

	if report.PurgedOTPs != 1 {
		t.Errorf("expected 1 purged OTP, got %d", report.PurgedOTPs)
	}

	// Verify expired OTP is gone
	_, err = store.GetLatestOTP(ctx, "+91", "9999000011")
	if err == nil {
		t.Errorf("expected expired OTP to be purged from store")
	}

	// Verify active OTP still exists
	activeFound, err := store.GetLatestOTP(ctx, "+91", "9999000022")
	if err != nil || activeFound == nil {
		t.Errorf("expected active OTP to remain in store: %v", err)
	}
}

func TestNomineeSection14Lifecycle(t *testing.T) {
	ts := setupTestServer()
	defer ts.Close()

	client := ts.Client()

	// Authenticate user
	otpReqBody, _ := json.Marshal(map[string]string{
		"phone":        "9849099887",
		"country_code": "+91",
		"purpose":      "login",
	})
	client.Post(ts.URL+"/api/v1/auth/otp/send", "application/json", bytes.NewReader(otpReqBody))

	verifyReqBody, _ := json.Marshal(map[string]string{
		"phone":        "9849099887",
		"country_code": "+91",
		"code":         "123456",
	})
	resp, _ := client.Post(ts.URL+"/api/v1/auth/otp/verify", "application/json", bytes.NewReader(verifyReqBody))

	var authResp struct {
		Data struct {
			Token string `json:"token"`
		} `json:"data"`
	}
	json.NewDecoder(resp.Body).Decode(&authResp)
	token := authResp.Data.Token

	doAuthReq := func(method, path string, body []byte) (*http.Response, error) {
		var req *http.Request
		if body != nil {
			req, _ = http.NewRequest(method, ts.URL+path, bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
		} else {
			req, _ = http.NewRequest(method, ts.URL+path, nil)
		}
		req.Header.Set("Authorization", "Bearer "+token)
		return client.Do(req)
	}

	// 1. Designate Nominee under Section 14
	nomineeReq, _ := json.Marshal(map[string]string{
		"nominee_name":  "Ananya Reddy",
		"relationship":  "Spouse",
		"phone":         "9849011223",
		"email":         "ananya@example.com",
	})
	resp, err := doAuthReq("POST", "/api/v1/privacy/nominee", nomineeReq)
	if err != nil || resp.StatusCode != http.StatusCreated {
		t.Fatalf("set nominee failed: %v, status: %d", err, resp.StatusCode)
	}

	var nomineeResp struct {
		Data privacy.Nominee `json:"data"`
	}
	json.NewDecoder(resp.Body).Decode(&nomineeResp)
	if nomineeResp.Data.NomineeName != "Ananya Reddy" {
		t.Errorf("expected nominee name Ananya Reddy, got %s", nomineeResp.Data.NomineeName)
	}
	if nomineeResp.Data.Status != privacy.NomineeStatusActive {
		t.Errorf("expected status active, got %s", nomineeResp.Data.Status)
	}

	// 2. Fetch Nominee
	resp, err = doAuthReq("GET", "/api/v1/privacy/nominee", nil)
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("get nominee failed: %v, status: %d", err, resp.StatusCode)
	}

	// 3. Revoke Nominee
	resp, err = doAuthReq("DELETE", "/api/v1/privacy/nominee", nil)
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("revoke nominee failed: %v, status: %d", err, resp.StatusCode)
	}

	// 4. Verify Nominee no longer found
	resp, err = doAuthReq("GET", "/api/v1/privacy/nominee", nil)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404 Not Found for revoked nominee, got status: %d", resp.StatusCode)
	}
}

func TestDataBreachNotificationSection8(t *testing.T) {
	ts := setupTestServer()
	defer ts.Close()

	client := ts.Client()

	// Authenticate user
	otpReqBody, _ := json.Marshal(map[string]string{
		"phone":        "9849099887",
		"country_code": "+91",
		"purpose":      "login",
	})
	client.Post(ts.URL+"/api/v1/auth/otp/send", "application/json", bytes.NewReader(otpReqBody))

	verifyReqBody, _ := json.Marshal(map[string]string{
		"phone":        "9849099887",
		"country_code": "+91",
		"code":         "123456",
	})
	resp, _ := client.Post(ts.URL+"/api/v1/auth/otp/verify", "application/json", bytes.NewReader(verifyReqBody))

	var authResp struct {
		Data struct {
			Token string `json:"token"`
		} `json:"data"`
	}
	json.NewDecoder(resp.Body).Decode(&authResp)
	token := authResp.Data.Token

	doAuthReq := func(method, path string, body []byte) (*http.Response, error) {
		var req *http.Request
		if body != nil {
			req, _ = http.NewRequest(method, ts.URL+path, bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
		} else {
			req, _ = http.NewRequest(method, ts.URL+path, nil)
		}
		req.Header.Set("Authorization", "Bearer "+token)
		return client.Do(req)
	}

	// 1. Record Data Breach Incident
	breachReq, _ := json.Marshal(map[string]any{
		"incident_reference": "INC-HYD-2026-004",
		"nature_of_breach":   "Unauthorized credential testing from suspicious IP subnet",
		"severity":           "high",
		"affected_count":     42,
		"mitigation_taken":   "IP subnet blocked and affected sessions terminated",
	})
	resp, err := doAuthReq("POST", "/api/v1/ops/privacy/breaches", breachReq)
	if err != nil || resp.StatusCode != http.StatusCreated {
		t.Fatalf("record breach failed: %v, status: %d", err, resp.StatusCode)
	}

	var breachResp struct {
		Data privacy.DataBreach `json:"data"`
	}
	json.NewDecoder(resp.Body).Decode(&breachResp)
	breachID := breachResp.Data.ID
	if breachID == uuid.Nil {
		t.Errorf("expected valid breach ID")
	}

	// 2. Report Breach to DPBI under Section 8(6)
	resp, err = doAuthReq("POST", "/api/v1/ops/privacy/breaches/"+breachID.String()+"/report-dpbi", nil)
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("report to DPBI failed: %v, status: %d", err, resp.StatusCode)
	}

	// 3. List Breaches and verify status updated
	resp, err = doAuthReq("GET", "/api/v1/ops/privacy/breaches", nil)
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("list breaches failed: %v, status: %d", err, resp.StatusCode)
	}

	var listResp struct {
		Data []privacy.DataBreach `json:"data"`
	}
	json.NewDecoder(resp.Body).Decode(&listResp)
	found := false
	for _, b := range listResp.Data {
		if b.ID == breachID {
			found = true
			if b.Status != privacy.BreachStatusReported {
				t.Errorf("expected status reported, got %s", b.Status)
			}
			if b.ReportedToDPBIAt == nil {
				t.Errorf("expected reported_to_dpbi_at to be populated")
			}
		}
	}
	if !found {
		t.Errorf("expected to find recorded breach in list")
	}
}

func TestGPSLocationFuzzingAndPhoneMasking(t *testing.T) {
	// Test Phone Number Masking (Data Minimization)
	masked1 := privacy.MaskPhoneNumber("9849099887")
	if masked1 != "98*** **887" {
		t.Errorf("expected 98*** **887, got %s", masked1)
	}

	masked2 := privacy.MaskPhoneNumber("+919849099887")
	if masked2 != "+91 98*** **887" {
		t.Errorf("expected +91 98*** **887, got %s", masked2)
	}

	// Test GPS Fuzzing (~500m precision step)
	originalLat := 17.385044
	originalLon := 78.486671
	fuzzedLat, fuzzedLon := privacy.FuzzGPSCoordinates(originalLat, originalLon)

	// Fuzzed lat/lon should be multiples of 0.005
	diffLat := math.Abs(fuzzedLat - originalLat)
	diffLon := math.Abs(fuzzedLon - originalLon)

	if diffLat > 0.005 || diffLon > 0.005 {
		t.Errorf("fuzzed coordinate difference should be within 0.005 deg: diffLat=%f, diffLon=%f", diffLat, diffLon)
	}
}
