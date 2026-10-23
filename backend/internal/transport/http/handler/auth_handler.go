package handler

import (
	"encoding/json"
	"net/http"

	"github.com/nenav/nenav/backend/internal/service"
	"github.com/nenav/nenav/backend/internal/transport/http/response"
)

type AuthHandler struct {
	auth *service.AuthService
}

func NewAuthHandler(auth *service.AuthService) *AuthHandler {
	return &AuthHandler{auth: auth}
}

type SendOTPRequest struct {
	Phone       string `json:"phone"`
	CountryCode string `json:"country_code"`
	Purpose     string `json:"purpose"`
}

func (h *AuthHandler) HandleSendOTP(w http.ResponseWriter, r *http.Request) {
	var req SendOTPRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request payload"}`, http.StatusBadRequest)
		return
	}

	_, err := h.auth.SendOTP(r.Context(), req.CountryCode, req.Phone, req.Purpose)
	if err != nil {
		http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusBadRequest)
		return
	}

	response.JSON(w, http.StatusOK, map[string]any{
		"message": "OTP sent successfully",
		"phone":   req.CountryCode + req.Phone,
	})
}

type VerifyOTPRequest struct {
	Phone       string `json:"phone"`
	CountryCode string `json:"country_code"`
	Code        string `json:"code"`
}

func (h *AuthHandler) HandleVerifyOTP(w http.ResponseWriter, r *http.Request) {
	var req VerifyOTPRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request payload"}`, http.StatusBadRequest)
		return
	}

	user, session, err := h.auth.VerifyOTP(r.Context(), req.CountryCode, req.Phone, req.Code)
	if err != nil {
		http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusUnauthorized)
		return
	}

	response.JSON(w, http.StatusOK, map[string]any{
		"user":       user,
		"token":      session.Token,
		"expires_at": session.ExpiresAt,
	})
}
