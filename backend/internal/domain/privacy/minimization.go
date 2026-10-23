package privacy

import (
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/google/uuid"
)

// NomineeStatus indicates if a Section 14 nominee designation is active.
type NomineeStatus string

const (
	NomineeStatusActive  NomineeStatus = "active"
	NomineeStatusRevoked NomineeStatus = "revoked"
)

// Nominee represents a designated individual under DPDP Act Section 14
// empowered to exercise data principal rights in the event of death or incapacity.
type Nominee struct {
	ID           uuid.UUID     `json:"id"`
	UserID       uuid.UUID     `json:"user_id"`
	NomineeName  string        `json:"nominee_name"`
	Relationship string        `json:"relationship"`
	Phone        string        `json:"phone"`
	Email        *string       `json:"email,omitempty"`
	Status       NomineeStatus `json:"status"`
	CreatedAt    time.Time     `json:"created_at"`
	UpdatedAt    time.Time     `json:"updated_at"`
}

// BreachSeverity categorizes incident severity under DPDP Section 8(6).
type BreachSeverity string

const (
	BreachSeverityLow      BreachSeverity = "low"
	BreachSeverityMedium   BreachSeverity = "medium"
	BreachSeverityHigh     BreachSeverity = "high"
	BreachSeverityCritical BreachSeverity = "critical"
)

// BreachStatus tracks incident response stages.
type BreachStatus string

const (
	BreachStatusInvestigating BreachStatus = "investigating"
	BreachStatusContained     BreachStatus = "contained"
	BreachStatusReported      BreachStatus = "reported"
	BreachStatusClosed        BreachStatus = "closed"
)

// DataBreach models a personal data breach event requiring statutory reporting.
type DataBreach struct {
	ID                   uuid.UUID      `json:"id"`
	IncidentReference    string         `json:"incident_reference"`
	NatureOfBreach       string         `json:"nature_of_breach"`
	AffectedCount        int            `json:"affected_count"`
	Severity             BreachSeverity `json:"severity"`
	MitigationTaken      string         `json:"mitigation_taken"`
	ReportedToDPBIAt     *time.Time     `json:"reported_to_dpbi_at,omitempty"`
	NotifiedPrincipalsAt *time.Time     `json:"notified_principals_at,omitempty"`
	Status               BreachStatus   `json:"status"`
	CreatedAt            time.Time      `json:"created_at"`
	UpdatedAt            time.Time      `json:"updated_at"`
}

// MaskPhoneNumber obfuscates phone numbers for bipartite or public displays (e.g., "+91 98*** **887").
func MaskPhoneNumber(phone string) string {
	cleaned := strings.TrimSpace(phone)
	if len(cleaned) < 10 {
		return "**********"
	}
	// Take last 10 digits
	last10 := cleaned
	prefix := ""
	if len(cleaned) > 10 {
		prefix = cleaned[:len(cleaned)-10] + " "
		last10 = cleaned[len(cleaned)-10:]
	}

	return fmt.Sprintf("%s%s*** **%s", prefix, last10[:2], last10[7:])
}

// FuzzGPSCoordinates blurs high-precision GPS coordinates to approximately 500m precision (0.005 deg),
// enforcing statutory Data Minimization when precise location consent is revoked.
func FuzzGPSCoordinates(lat, lon float64) (float64, float64) {
	const step = 0.005 // ~555 meters latitude / ~480 meters longitude in India
	fuzzedLat := math.Round(lat/step) * step
	fuzzedLon := math.Round(lon/step) * step
	return fuzzedLat, fuzzedLon
}
