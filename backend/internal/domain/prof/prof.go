package prof

import (
	"math"
	"sort"
	"time"

	"github.com/google/uuid"
)

type KYCStatus string

const (
	KYCPending  KYCStatus = "pending"
	KYCVerified KYCStatus = "verified"
	KYCRejected KYCStatus = "rejected"
)

// ProfProfile represents a skilled professional on NENAV.
type ProfProfile struct {
	ID                  uuid.UUID `json:"id"`
	UserID              uuid.UUID `json:"user_id"`
	Name                string    `json:"name"`
	Phone               string    `json:"phone"`
	TradeID             uuid.UUID `json:"trade_id"`
	TradeName           string    `json:"trade_name"`
	KYCStatus           KYCStatus `json:"kyc_status"`
	ExperienceYears     int       `json:"experience_years"`
	Rating              float64   `json:"rating"`
	TotalReviews        int       `json:"total_reviews"`
	TotalJobs           int       `json:"total_jobs"`
	IsAvailable         bool      `json:"is_available"`
	Latitude            float64   `json:"latitude"`
	Longitude           float64   `json:"longitude"`
	BadgePoliceVerified bool      `json:"badge_police_verified"`
	BadgeSkillCertified bool      `json:"badge_skill_certified"`
	BadgeNenavPro       bool      `json:"badge_nenav_pro"`
	DistanceKm          float64   `json:"distance_km,omitempty"`
	Score               float64   `json:"score,omitempty"`
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"`
}

// HaversineDistance calculates the great-circle distance in kilometers.
func HaversineDistance(lat1, lon1, lat2, lon2 float64) float64 {
	const earthRadiusKm = 6371.0

	dLat := (lat2 - lat1) * (math.Pi / 180.0)
	dLon := (lon2 - lon1) * (math.Pi / 180.0)

	rLat1 := lat1 * (math.Pi / 180.0)
	rLat2 := lat2 * (math.Pi / 180.0)

	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Sin(dLon/2)*math.Sin(dLon/2)*math.Cos(rLat1)*math.Cos(rLat2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	return earthRadiusKm * c
}

// RankPROFs filters candidates within radiusKm (default 15 km) and ranks them deterministically.
func RankPROFs(profs []ProfProfile, targetTradeID uuid.UUID, custLat, custLon, maxRadiusKm float64) []ProfProfile {
	if maxRadiusKm <= 0 {
		maxRadiusKm = 15.0 // Hyderabad MVP maximum dispatch radius
	}

	var candidates []ProfProfile
	for _, p := range profs {
		if p.TradeID != targetTradeID {
			continue
		}
		if !p.IsAvailable || p.KYCStatus != KYCVerified {
			continue
		}

		dist := HaversineDistance(custLat, custLon, p.Latitude, p.Longitude)
		if dist <= maxRadiusKm {
			p.DistanceKm = math.Round(dist*100) / 100
			// Deterministic ranking formula:
			// High rating rewards, closer distance rewards, more experience rewards
			p.Score = (p.Rating * 10.0) - (p.DistanceKm * 0.5) + (float64(p.ExperienceYears) * 0.2)
			candidates = append(candidates, p)
		}
	}

	// Sort descending by score
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].Score > candidates[j].Score
	})

	return candidates
}
