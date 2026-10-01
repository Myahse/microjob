package api

import (
	"net/http"
	"sort"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"

	"github.com/myahse/microjob/backend/internal/domain"
)

type skillView struct {
	ID           int    `json:"id"`
	Name         string `json:"name"`
	CategoryID   int    `json:"category_id"`
	CategoryName string `json:"category_name,omitempty"`
	Verified     bool   `json:"verified,omitempty"`
	Active       bool   `json:"active"`
}

type categoryView struct {
	ID             int         `json:"id"`
	Slug           string      `json:"slug"`
	Name           string      `json:"name"`
	Icon           string      `json:"icon"`
	CommissionRate *float64    `json:"commission_rate"`
	Active         bool        `json:"active"`
	SortOrder      int         `json:"sort_order"`
	Skills         []skillView `json:"skills"`
}

func (s *Server) categories(r *http.Request, includeInactive bool) ([]categoryView, error) {
	rows, err := s.db.Query(r.Context(), `SELECT id, slug, name, icon, commission_rate, active, sort_order
		FROM categories WHERE active OR $1 ORDER BY sort_order, name`, includeInactive)
	if err != nil {
		return nil, err
	}
	cats, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (categoryView, error) {
		var c categoryView
		err := row.Scan(&c.ID, &c.Slug, &c.Name, &c.Icon, &c.CommissionRate, &c.Active, &c.SortOrder)
		c.Skills = []skillView{}
		return c, err
	})
	if err != nil {
		return nil, err
	}
	index := map[int]int{}
	for i, c := range cats {
		index[c.ID] = i
	}
	rows, err = s.db.Query(r.Context(), `SELECT id, name, category_id, active FROM skills WHERE active OR $1 ORDER BY name`, includeInactive)
	if err != nil {
		return nil, err
	}
	skills, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (skillView, error) {
		var sk skillView
		err := row.Scan(&sk.ID, &sk.Name, &sk.CategoryID, &sk.Active)
		return sk, err
	})
	if err != nil {
		return nil, err
	}
	for _, sk := range skills {
		if i, ok := index[sk.CategoryID]; ok {
			cats[i].Skills = append(cats[i].Skills, sk)
		}
	}
	return cats, nil
}

func (s *Server) listCategories(w http.ResponseWriter, r *http.Request) error {
	cats, err := s.categories(r, false)
	if err != nil {
		return err
	}
	return writeJSON(w, http.StatusOK, cats)
}

func queryFloat(r *http.Request, key string) (float64, bool) {
	v, err := strconv.ParseFloat(r.URL.Query().Get(key), 64)
	return v, err == nil
}

type workerSearchResult struct {
	ID            string   `json:"id"`
	FirstName     string   `json:"first_name"`
	LastName      string   `json:"last_name"`
	AvatarURL     string   `json:"avatar_url"`
	Headline      string   `json:"headline"`
	Zone          string   `json:"zone"`
	Rating        float64  `json:"rating"`
	RatingCount   int      `json:"rating_count"`
	CompletedJobs int      `json:"completed_jobs"`
	HourlyRate    int      `json:"hourly_rate"`
	Available     bool     `json:"available"`
	DistanceKm    *float64 `json:"distance_km"`
	Skills        []string `json:"skills"`
}

// searchWorkers answers "Je cherche un plombier" (§31). Only rounded
// distances are returned, never worker coordinates.
func (s *Server) searchWorkers(w http.ResponseWriter, r *http.Request) error {
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	categoryID, _ := strconv.Atoi(r.URL.Query().Get("category_id"))
	lat, hasLat := queryFloat(r, "lat")
	lng, hasLng := queryFloat(r, "lng")

	rows, err := s.db.Query(r.Context(), `SELECT u.id, u.first_name, u.last_name, u.avatar_url, wp.headline, u.zone,
			wp.rating_avg, wp.rating_count, wp.completed_jobs, wp.hourly_rate, wp.available, u.latitude, u.longitude,
			COALESCE(array_agg(DISTINCT sk.name) FILTER (WHERE sk.name IS NOT NULL), '{}')
		FROM users u
		JOIN worker_profiles wp ON wp.user_id = u.id
		LEFT JOIN user_skills us ON us.user_id = u.id
		LEFT JOIN skills sk ON sk.id = us.skill_id
		LEFT JOIN categories c ON c.id = sk.category_id
		WHERE u.is_worker AND u.status = 'active' AND u.id <> $1
		  AND ($2 = 0 OR sk.category_id = $2)
		  AND ($3 = '' OR sk.name ILIKE '%' || $3 || '%' OR c.name ILIKE '%' || $3 || '%'
		       OR wp.headline ILIKE '%' || $3 || '%' OR u.first_name ILIKE '%' || $3 || '%')
		GROUP BY u.id, wp.user_id
		LIMIT 200`, me(r).ID, categoryID, q)
	if err != nil {
		return err
	}
	results, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (workerSearchResult, error) {
		var v workerSearchResult
		var wlat, wlng *float64
		err := row.Scan(&v.ID, &v.FirstName, &v.LastName, &v.AvatarURL, &v.Headline, &v.Zone, &v.Rating, &v.RatingCount,
			&v.CompletedJobs, &v.HourlyRate, &v.Available, &wlat, &wlng, &v.Skills)
		if hasLat && hasLng && wlat != nil && wlng != nil {
			d := domain.RoundDistance(domain.DistanceKm(lat, lng, *wlat, *wlng))
			v.DistanceKm = &d
		}
		return v, err
	})
	if err != nil {
		return err
	}
	sort.SliceStable(results, func(i, j int) bool {
		a, b := results[i], results[j]
		if a.Available != b.Available {
			return a.Available
		}
		if a.DistanceKm != nil && b.DistanceKm != nil && *a.DistanceKm != *b.DistanceKm {
			return *a.DistanceKm < *b.DistanceKm
		}
		return a.Rating > b.Rating
	})
	if len(results) > 50 {
		results = results[:50]
	}
	return writeJSON(w, http.StatusOK, results)
}

// nearbyWorkerCount powers the map's "12 workers disponibles dans cette
// zone" (§32) without exposing positions.
func (s *Server) nearbyWorkerCount(w http.ResponseWriter, r *http.Request) error {
	lat, ok1 := queryFloat(r, "lat")
	lng, ok2 := queryFloat(r, "lng")
	if !ok1 || !ok2 {
		return badRequest("lat and lng are required")
	}
	radius, ok := queryFloat(r, "radius_km")
	if !ok || radius <= 0 || radius > 50 {
		radius = 5
	}
	categoryID, _ := strconv.Atoi(r.URL.Query().Get("category_id"))
	rows, err := s.db.Query(r.Context(), `SELECT u.latitude, u.longitude FROM users u
		JOIN worker_profiles wp ON wp.user_id = u.id
		WHERE u.is_worker AND u.status = 'active' AND wp.available AND u.latitude IS NOT NULL
		  AND ($1 = 0 OR EXISTS (SELECT 1 FROM user_skills us JOIN skills sk ON sk.id = us.skill_id
		                         WHERE us.user_id = u.id AND sk.category_id = $1))`, categoryID)
	if err != nil {
		return err
	}
	count := 0
	var wlat, wlng float64
	_, err = pgx.ForEachRow(rows, []any{&wlat, &wlng}, func() error {
		if domain.DistanceKm(lat, lng, wlat, wlng) <= radius {
			count++
		}
		return nil
	})
	if err != nil {
		return err
	}
	return writeJSON(w, http.StatusOK, map[string]any{"count": count, "radius_km": radius})
}
