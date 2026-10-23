package identity

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"time"

	"github.com/google/uuid"
)

var (
	ErrInvalidPhone       = errors.New("invalid phone number: must be 10 digits for India")
	ErrOTPExpired         = errors.New("otp expired")
	ErrOTPInvalid         = errors.New("invalid otp code")
	ErrMaxAttemptsExceeded = errors.New("maximum otp verification attempts exceeded")
	ErrSessionExpired     = errors.New("session expired")
)

type UserRole string

const (
	RoleUser     UserRole = "user"
	RoleProf     UserRole = "prof"
	RoleBusiness UserRole = "business"
	RoleAdmin    UserRole = "admin"
	RoleSupport  UserRole = "support"
)

type UserStatus string

const (
	StatusPending     UserStatus = "pending"
	StatusActive      UserStatus = "active"
	StatusSuspended   UserStatus = "suspended"
	StatusDeactivated UserStatus = "deactivated"
)

// User represents an authenticated platform entity.
type User struct {
	ID               uuid.UUID  `json:"id"`
	Phone            string     `json:"phone"`
	PhoneCountryCode string     `json:"phone_country_code"`
	Email            *string    `json:"email,omitempty"`
	DisplayName      string     `json:"display_name"`
	Role             UserRole   `json:"role"`
	Status           UserStatus `json:"status"`
	Locale           string     `json:"locale"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
}

// OTPRequest represents a challenge to verify phone ownership.
type OTPRequest struct {
	ID               uuid.UUID `json:"id"`
	Phone            string    `json:"phone"`
	PhoneCountryCode string    `json:"phone_country_code"`
	OTPHash          string    `json:"-"`
	Purpose          string    `json:"purpose"`
	Attempts         int       `json:"attempts"`
	MaxAttempts      int       `json:"max_attempts"`
	Verified         bool      `json:"verified"`
	ExpiresAt        time.Time `json:"expires_at"`
	CreatedAt        time.Time `json:"created_at"`
}

// Session represents an active authentication token.
type Session struct {
	ID        uuid.UUID  `json:"id"`
	UserID    uuid.UUID  `json:"user_id"`
	TokenHash string     `json:"-"`
	Token     string     `json:"token,omitempty"`
	ExpiresAt time.Time  `json:"expires_at"`
	RevokedAt *time.Time `json:"revoked_at,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
}

// GenerateOTP creates a cryptographically secure 6-digit numeric OTP.
func GenerateOTP() (string, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(900000))
	if err != nil {
		return "", fmt.Errorf("generate otp: %w", err)
	}
	return fmt.Sprintf("%06d", n.Int64()+100000), nil
}

// HashOTP produces a SHA-256 hash with phone salt.
func HashOTP(phone, code, salt string) string {
	h := sha256.Sum256([]byte(phone + ":" + code + ":" + salt))
	return hex.EncodeToString(h[:])
}

// GenerateSessionToken creates a secure random bearer token string.
func GenerateSessionToken() (string, string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", "", fmt.Errorf("generate token: %w", err)
	}
	rawToken := hex.EncodeToString(bytes)
	h := sha256.Sum256([]byte(rawToken))
	tokenHash := hex.EncodeToString(h[:])
	return rawToken, tokenHash, nil
}
