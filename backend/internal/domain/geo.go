package domain

import "math"

const earthRadiusKm = 6371.0

// DistanceKm is the great-circle distance between two coordinates.
func DistanceKm(lat1, lng1, lat2, lng2 float64) float64 {
	toRad := func(d float64) float64 { return d * math.Pi / 180 }
	dLat := toRad(lat2 - lat1)
	dLng := toRad(lng2 - lng1)
	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(toRad(lat1))*math.Cos(toRad(lat2))*math.Sin(dLng/2)*math.Sin(dLng/2)
	return 2 * earthRadiusKm * math.Asin(math.Sqrt(a))
}

// RoundDistance rounds to 100 m so shown distances don't leak exact positions.
func RoundDistance(km float64) float64 {
	return math.Round(km*10) / 10
}

// MatchInput describes a candidate worker for a job.
type MatchInput struct {
	DistanceKm    float64
	RadiusKm      float64
	Rating        float64 // 0..5, 0 when unrated
	RatingCount   int
	SimilarJobs   int // completed jobs in the same category
	CompletedJobs int
}

// MatchScore ranks a compatible worker (higher is better), combining the
// criteria of §12: distance first, then reputation and relevant history.
// Unrated newcomers get a neutral rating so they can still be matched.
func MatchScore(in MatchInput) float64 {
	radius := in.RadiusKm
	if radius <= 0 {
		radius = 10
	}
	proximity := math.Max(0, 1-in.DistanceKm/radius) // 1 = next door

	rating := 3.5
	if in.RatingCount > 0 {
		// Shrink toward the neutral prior when there are few ratings, so a
		// single 5★ cannot outrank an established 4.8★ worker.
		w := float64(in.RatingCount) / float64(in.RatingCount+5)
		rating = w*in.Rating + (1-w)*3.5
	}
	reputation := rating / 5

	history := math.Min(1, float64(in.SimilarJobs)/10)

	return 0.5*proximity + 0.3*reputation + 0.2*history
}
