package tests

import (
	"bufio"
	"bytes"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/google/uuid"
)

func TestRealtimeLifecycleAndRaceConditions(t *testing.T) {
	ts := setupTestServer()
	defer ts.Close()

	client := ts.Client()

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

	// 1. Authenticate Customer
	otpReq, _ := json.Marshal(map[string]string{
		"phone":        "9849099887",
		"country_code": "+91",
		"purpose":      "login",
	})
	client.Post(ts.URL+"/api/v1/auth/otp/send", "application/json", bytes.NewReader(otpReq))
	verifyReq, _ := json.Marshal(map[string]string{
		"phone":        "9849099887",
		"country_code": "+91",
		"code":         "123456",
	})
	resp, err := client.Post(ts.URL+"/api/v1/auth/otp/verify", "application/json", bytes.NewReader(verifyReq))
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("auth failed: %v", err)
	}
	var authData struct {
		Data struct {
			Token string `json:"token"`
		} `json:"data"`
	}
	json.NewDecoder(resp.Body).Decode(&authData)
	customerToken := authData.Data.Token

	// 2. Connect to SSE Stream in background
	sseReq, err := http.NewRequest("GET", ts.URL+"/api/v1/realtime/stream?channel=admin:marketplace", nil)
	if err != nil {
		t.Fatalf("failed to create SSE request: %v", err)
	}
	sseResp, err := client.Do(sseReq)
	if err != nil || sseResp.StatusCode != http.StatusOK {
		t.Fatalf("failed to connect to SSE stream: %v", err)
	}
	scanner := bufio.NewScanner(sseResp.Body)
	if !scanner.Scan() {
		sseResp.Body.Close()
		t.Fatalf("failed to read initial SSE stream preamble")
	}
	preamble := scanner.Text()
	sseResp.Body.Close()
	if !strings.Contains(preamble, "connected") {
		t.Errorf("expected connected message, got: %s", preamble)
	}

	// 3. Create a Cash Booking (automatically enters 'searching' status)
	createReq, _ := json.Marshal(map[string]any{
		"trade_id":        uuid.New().String(),
		"trade_name":      "AC Service & Repair",
		"duration_type":   "hourly",
		"scheduled_date":  "2026-09-15",
		"time_slot":       "14:00 - 16:00",
		"hours":           2,
		"worker_count":    1,
		"address_line":    "Hitech City, Madhapur, Hyderabad",
		"job_description": "Cooling issues in master bedroom split AC",
		"payment_method":  "cash",
	})
	bResp, err := doAuthReq("POST", "/api/v1/bookings", createReq, customerToken)
	if err != nil || bResp.StatusCode != http.StatusCreated {
		var errResp map[string]any
		if bResp != nil {
			json.NewDecoder(bResp.Body).Decode(&errResp)
			t.Fatalf("booking creation failed (status %d): %v", bResp.StatusCode, errResp)
		}
		t.Fatalf("booking creation failed: %v", err)
	}
	var newBooking struct {
		Data struct {
			ID     string `json:"id"`
			Status string `json:"status"`
		} `json:"data"`
	}
	json.NewDecoder(bResp.Body).Decode(&newBooking)
	bookingID := newBooking.Data.ID

	// 4. Test Race Condition Protection (Master Prompt §12):
	// Professional A claims job -> succeeds.
	// Professional B claims same job -> receives 409 Conflict.
	profA := uuid.New()
	profB := uuid.New()

	claimABody, _ := json.Marshal(map[string]string{"prof_id": profA.String()})
	claimRespA, err := doAuthReq("POST", "/api/v1/bookings/"+bookingID+"/claim", claimABody, customerToken)
	if err != nil || claimRespA.StatusCode != http.StatusOK {
		t.Fatalf("prof A claim should succeed: %v, status: %d", err, claimRespA.StatusCode)
	}

	claimBBody, _ := json.Marshal(map[string]string{"prof_id": profB.String()})
	claimRespB, err := doAuthReq("POST", "/api/v1/bookings/"+bookingID+"/claim", claimBBody, customerToken)
	if err != nil {
		t.Fatalf("prof B claim request error: %v", err)
	}
	if claimRespB.StatusCode != http.StatusConflict {
		t.Fatalf("expected 409 Conflict for duplicate claim by Prof B, got: %d", claimRespB.StatusCode)
	}

	// 5. Professional transitions to En Route
	enRouteResp, err := doAuthReq("POST", "/api/v1/bookings/"+bookingID+"/en-route", nil, customerToken)
	if err != nil || enRouteResp.StatusCode != http.StatusOK {
		t.Fatalf("en-route failed: %v", err)
	}

	// 6. Professional streams GPS Location (Master Prompt §14)
	locBody, _ := json.Marshal(map[string]float64{
		"lat": 17.4435,
		"lng": 78.3842,
	})
	locResp, err := doAuthReq("POST", "/api/v1/bookings/"+bookingID+"/location", locBody, customerToken)
	if err != nil || locResp.StatusCode != http.StatusOK {
		t.Fatalf("location update failed: %v", err)
	}

	// 7. Professional Arrives
	arrivedResp, err := doAuthReq("POST", "/api/v1/bookings/"+bookingID+"/arrived", nil, customerToken)
	if err != nil || arrivedResp.StatusCode != http.StatusOK {
		t.Fatalf("arrived transition failed: %v", err)
	}

	// 8. Mutual Check-in using 4-digit code (4829)
	checkinBody, _ := json.Marshal(map[string]string{"otp": "4829"})
	checkinResp, err := doAuthReq("POST", "/api/v1/bookings/"+bookingID+"/checkin", checkinBody, customerToken)
	if err != nil || checkinResp.StatusCode != http.StatusOK {
		t.Fatalf("checkin failed: %v", err)
	}

	// 9. Additional Work Handshake (Master Prompt §18):
	// Professional requests additional ₹350 (₹300 labor + ₹50 materials)
	addWorkBody, _ := json.Marshal(map[string]any{
		"labor_cost":    300,
		"material_cost": 50,
		"description":   "Capacitor replacement and copper brazing",
	})
	awResp, err := doAuthReq("POST", "/api/v1/bookings/"+bookingID+"/additional-work", addWorkBody, customerToken)
	if err != nil || awResp.StatusCode != http.StatusOK {
		t.Fatalf("additional work request failed: %v", err)
	}
	var awBooking struct {
		Data struct {
			Status         string `json:"status"`
			AdditionalWork struct {
				TotalAmount int `json:"total_amount"`
			} `json:"additional_work"`
		} `json:"data"`
	}
	json.NewDecoder(awResp.Body).Decode(&awBooking)
	if awBooking.Data.Status != "additional_work_pending" || awBooking.Data.AdditionalWork.TotalAmount != 350 {
		t.Fatalf("expected additional_work_pending with 350, got status: %s, total: %d",
			awBooking.Data.Status, awBooking.Data.AdditionalWork.TotalAmount)
	}

	// 10. Customer Approves Additional Work
	approveResp, err := doAuthReq("POST", "/api/v1/bookings/"+bookingID+"/additional-work/approve", nil, customerToken)
	if err != nil || approveResp.StatusCode != http.StatusOK {
		t.Fatalf("customer approve additional work failed: %v", err)
	}
	var approvedBooking struct {
		Data struct {
			Status string `json:"status"`
			Quote  struct {
				TotalAmount int `json:"total_amount"`
			} `json:"quote"`
		} `json:"data"`
	}
	json.NewDecoder(approveResp.Body).Decode(&approvedBooking)
	if approvedBooking.Data.Status != "in_progress" {
		t.Fatalf("expected in_progress after approval, got: %s", approvedBooking.Data.Status)
	}

	// 11. Complete Service
	compResp, err := doAuthReq("POST", "/api/v1/bookings/"+bookingID+"/complete", nil, customerToken)
	if err != nil || compResp.StatusCode != http.StatusOK {
		t.Fatalf("job completion failed: %v", err)
	}
}
