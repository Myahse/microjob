package api

import (
	"context"
	"errors"
	"math"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/myahse/microjob/backend/internal/auth"
)

var usernameRe = regexp.MustCompile(`^[a-z0-9_.]{3,30}$`)

type workerView struct {
	Headline       string      `json:"headline"`
	Available      bool        `json:"available"`
	HourlyRate     int         `json:"hourly_rate"`
	RadiusKm       float64     `json:"radius_km"`
	Rating         float64     `json:"rating"`
	RatingCount    int         `json:"rating_count"`
	CompletedJobs  int         `json:"completed_jobs"`
	CompletionRate float64     `json:"completion_rate"`
	Skills         []skillView `json:"skills"`
	Badges         []string    `json:"badges"`
}

type userView struct {
	ID                string      `json:"id"`
	FirstName         string      `json:"first_name"`
	LastName          string      `json:"last_name"`
	Username          string      `json:"username"`
	AvatarURL         string      `json:"avatar_url"`
	Bio               string      `json:"bio"`
	Zone              string      `json:"zone"`
	IsClient          bool        `json:"is_client"`
	IsWorker          bool        `json:"is_worker"`
	IdentityVerified  bool        `json:"identity_verified"`
	PhoneVerified     bool        `json:"phone_verified"`
	EmailVerified     bool        `json:"email_verified"`
	ClientRating      float64     `json:"client_rating"`
	ClientRatingCount int         `json:"client_rating_count"`
	Worker            *workerView `json:"worker,omitempty"`
	CreatedAt         time.Time   `json:"created_at"`

	// Private fields, only returned to the user themself and admins.
	Email     string   `json:"email,omitempty"`
	Phone     string   `json:"phone,omitempty"`
	IsAdmin   bool     `json:"is_admin,omitempty"`
	Status    string   `json:"status,omitempty"`
	Latitude  *float64 `json:"latitude,omitempty"`
	Longitude *float64 `json:"longitude,omitempty"`
}

type reviewView struct {
	ID           string    `json:"id"`
	JobTitle     string    `json:"job_title"`
	ReviewerName string    `json:"reviewer_name"`
	ReviewerRole string    `json:"reviewer_role"`
	Rating       int       `json:"rating"`
	Comment      string    `json:"comment"`
	CreatedAt    time.Time `json:"created_at"`
}

func completionRate(completed, cancelled int) float64 {
	if completed+cancelled == 0 {
		return 0
	}
	return math.Round(float64(completed)/float64(completed+cancelled)*1000) / 10
}

// badges derives profile badges from real activity only (§26).
func badges(identityVerified bool, w *workerView) []string {
	out := []string{}
	if identityVerified {
		out = append(out, "Identité vérifiée")
	}
	if w == nil {
		return out
	}
	if w.Rating >= 4.8 && w.RatingCount >= 20 && w.CompletionRate >= 95 {
		out = append(out, "Top Worker")
	}
	seen := map[string]bool{}
	for _, sk := range w.Skills {
		if sk.Verified && !seen[sk.CategoryName] {
			seen[sk.CategoryName] = true
			out = append(out, sk.CategoryName+" vérifié")
		}
	}
	return out
}

func (s *Server) loadUser(ctx context.Context, id string, private bool) (*userView, error) {
	u := &userView{}
	var status, email, phone string
	var isAdmin bool
	var lat, lng *float64
	err := s.db.QueryRow(ctx, `SELECT id, first_name, last_name, username, avatar_url, bio, zone,
			is_client, is_worker, identity_verified, phone_verified, email_verified,
			client_rating_avg, client_rating_count, created_at, email, phone, is_admin, status, latitude, longitude
		FROM users WHERE id = $1`, id).Scan(&u.ID, &u.FirstName, &u.LastName, &u.Username, &u.AvatarURL, &u.Bio, &u.Zone,
		&u.IsClient, &u.IsWorker, &u.IdentityVerified, &u.PhoneVerified, &u.EmailVerified,
		&u.ClientRating, &u.ClientRatingCount, &u.CreatedAt, &email, &phone, &isAdmin, &status, &lat, &lng)
	if err != nil {
		return nil, err
	}
	if private {
		u.Email, u.Phone, u.IsAdmin, u.Status, u.Latitude, u.Longitude = email, phone, isAdmin, status, lat, lng
	}
	if u.IsWorker {
		w := &workerView{}
		var cancelled int
		err := s.db.QueryRow(ctx, `SELECT headline, available, hourly_rate, radius_km, rating_avg, rating_count, completed_jobs, cancelled_jobs
			FROM worker_profiles WHERE user_id = $1`, id).Scan(&w.Headline, &w.Available, &w.HourlyRate, &w.RadiusKm,
			&w.Rating, &w.RatingCount, &w.CompletedJobs, &cancelled)
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return nil, err
		}
		w.CompletionRate = completionRate(w.CompletedJobs, cancelled)
		if w.Skills, err = s.userSkills(ctx, id); err != nil {
			return nil, err
		}
		u.Worker = w
		w.Badges = badges(u.IdentityVerified, w)
	}
	return u, nil
}

func (s *Server) userSkills(ctx context.Context, userID string) ([]skillView, error) {
	rows, err := s.db.Query(ctx, `SELECT s.id, s.name, s.category_id, c.name, us.verified
		FROM user_skills us JOIN skills s ON s.id = us.skill_id JOIN categories c ON c.id = s.category_id
		WHERE us.user_id = $1 ORDER BY c.sort_order, s.name`, userID)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (skillView, error) {
		var sk skillView
		err := row.Scan(&sk.ID, &sk.Name, &sk.CategoryID, &sk.CategoryName, &sk.Verified)
		return sk, err
	})
}

type authResponse struct {
	Token string    `json:"token"`
	User  *userView `json:"user"`
}

func (s *Server) register(w http.ResponseWriter, r *http.Request) error {
	var req struct {
		FirstName      string   `json:"first_name"`
		LastName       string   `json:"last_name"`
		Username       string   `json:"username"`
		Phone          string   `json:"phone"`
		Email          string   `json:"email"`
		Password       string   `json:"password"`
		AvatarURL      string   `json:"avatar_url"`
		Latitude       *float64 `json:"latitude"`
		Longitude      *float64 `json:"longitude"`
		Zone           string   `json:"zone"`
		WantsServices  bool     `json:"wants_services"`
		OffersServices bool     `json:"offers_services"`
	}
	if err := decode(r, &req); err != nil {
		return err
	}
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))
	req.Username = strings.ToLower(strings.TrimSpace(req.Username))
	req.Phone = strings.ReplaceAll(strings.TrimSpace(req.Phone), " ", "")
	switch {
	case strings.TrimSpace(req.FirstName) == "" || strings.TrimSpace(req.LastName) == "":
		return badRequest("first and last name are required")
	case !usernameRe.MatchString(req.Username):
		return badRequest("username must be 3-30 characters: letters, digits, _ or .")
	case !strings.Contains(req.Email, "@"):
		return badRequest("invalid email")
	case len(req.Phone) < 8:
		return badRequest("invalid phone number")
	case len(req.Password) < 8:
		return badRequest("password must be at least 8 characters")
	case (req.Latitude == nil) != (req.Longitude == nil):
		return badRequest("latitude and longitude go together")
	}
	if !req.WantsServices && !req.OffersServices {
		req.WantsServices = true
	}
	hash, err := auth.HashPassword(req.Password)
	if err != nil {
		return err
	}

	var id string
	err = s.inTx(r.Context(), func(tx pgx.Tx) error {
		var taken string
		err := tx.QueryRow(r.Context(), `SELECT CASE WHEN email = $1 THEN 'email' WHEN username = $2 THEN 'username' ELSE 'phone' END
			FROM users WHERE email = $1 OR username = $2 OR phone = $3 LIMIT 1`, req.Email, req.Username, req.Phone).Scan(&taken)
		if err == nil {
			return conflict(taken + " already in use")
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return err
		}
		err = tx.QueryRow(r.Context(), `INSERT INTO users (first_name, last_name, username, phone, email, password_hash,
				avatar_url, latitude, longitude, zone, is_client, is_worker)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12) RETURNING id`,
			strings.TrimSpace(req.FirstName), strings.TrimSpace(req.LastName), req.Username, req.Phone, req.Email, hash,
			req.AvatarURL, req.Latitude, req.Longitude, req.Zone, req.WantsServices, req.OffersServices).Scan(&id)
		if err != nil {
			return err
		}
		if req.OffersServices {
			_, err = tx.Exec(r.Context(), `INSERT INTO worker_profiles (user_id, radius_km)
				VALUES ($1, (SELECT value::float FROM settings WHERE key = 'default_worker_radius_km'))`, id)
		}
		return err
	})
	if err != nil {
		return err
	}
	return s.respondWithToken(w, r, id, http.StatusCreated)
}

func (s *Server) respondWithToken(w http.ResponseWriter, r *http.Request, id string, status int) error {
	token, err := s.tokens.Issue(id)
	if err != nil {
		return err
	}
	u, err := s.loadUser(r.Context(), id, true)
	if err != nil {
		return err
	}
	return writeJSON(w, status, authResponse{Token: token, User: u})
}

func (s *Server) login(w http.ResponseWriter, r *http.Request) error {
	var req struct {
		Identifier string `json:"identifier"` // email, username or phone
		Password   string `json:"password"`
	}
	if err := decode(r, &req); err != nil {
		return err
	}
	ident := strings.ToLower(strings.TrimSpace(req.Identifier))
	var id, hash, status string
	err := s.db.QueryRow(r.Context(), `SELECT id, password_hash, status FROM users
		WHERE email = $1 OR username = $1 OR phone = $2`, ident, strings.ReplaceAll(ident, " ", "")).Scan(&id, &hash, &status)
	if errors.Is(err, pgx.ErrNoRows) || (err == nil && !auth.CheckPassword(hash, req.Password)) {
		return errStatus(http.StatusUnauthorized, "invalid credentials")
	}
	if err != nil {
		return err
	}
	if status != "active" {
		return forbidden("account " + status)
	}
	return s.respondWithToken(w, r, id, http.StatusOK)
}

func (s *Server) getMe(w http.ResponseWriter, r *http.Request) error {
	u, err := s.loadUser(r.Context(), me(r).ID, true)
	if err != nil {
		return err
	}
	return writeJSON(w, http.StatusOK, u)
}

func (s *Server) updateMe(w http.ResponseWriter, r *http.Request) error {
	var req struct {
		FirstName *string  `json:"first_name"`
		LastName  *string  `json:"last_name"`
		Bio       *string  `json:"bio"`
		AvatarURL *string  `json:"avatar_url"`
		Zone      *string  `json:"zone"`
		Latitude  *float64 `json:"latitude"`
		Longitude *float64 `json:"longitude"`
		IsClient  *bool    `json:"is_client"`
	}
	if err := decode(r, &req); err != nil {
		return err
	}
	if (req.Latitude == nil) != (req.Longitude == nil) {
		return badRequest("latitude and longitude go together")
	}
	_, err := s.db.Exec(r.Context(), `UPDATE users SET
			first_name = COALESCE($2, first_name), last_name = COALESCE($3, last_name), bio = COALESCE($4, bio),
			avatar_url = COALESCE($5, avatar_url), zone = COALESCE($6, zone),
			latitude = COALESCE($7, latitude), longitude = COALESCE($8, longitude),
			is_client = COALESCE($9, is_client), updated_at = now()
		WHERE id = $1`, me(r).ID, req.FirstName, req.LastName, req.Bio, req.AvatarURL, req.Zone, req.Latitude, req.Longitude, req.IsClient)
	if err != nil {
		return err
	}
	return s.getMe(w, r)
}

// updateWorkerProfile turns the user into a worker (if needed) and sets the
// professional profile and skills.
func (s *Server) updateWorkerProfile(w http.ResponseWriter, r *http.Request) error {
	var req struct {
		Headline   *string  `json:"headline"`
		HourlyRate *int     `json:"hourly_rate"`
		RadiusKm   *float64 `json:"radius_km"`
		Available  *bool    `json:"available"`
		SkillIDs   []int    `json:"skill_ids"`
	}
	if err := decode(r, &req); err != nil {
		return err
	}
	if req.RadiusKm != nil && (*req.RadiusKm < 1 || *req.RadiusKm > 50) {
		return badRequest("radius must be between 1 and 50 km")
	}
	if req.HourlyRate != nil && *req.HourlyRate < 0 {
		return badRequest("hourly rate must be positive")
	}
	id := me(r).ID
	err := s.inTx(r.Context(), func(tx pgx.Tx) error {
		if _, err := tx.Exec(r.Context(), `UPDATE users SET is_worker = TRUE, updated_at = now() WHERE id = $1`, id); err != nil {
			return err
		}
		_, err := tx.Exec(r.Context(), `INSERT INTO worker_profiles (user_id, radius_km)
			VALUES ($1, (SELECT value::float FROM settings WHERE key = 'default_worker_radius_km'))
			ON CONFLICT (user_id) DO NOTHING`, id)
		if err != nil {
			return err
		}
		_, err = tx.Exec(r.Context(), `UPDATE worker_profiles SET headline = COALESCE($2, headline),
				hourly_rate = COALESCE($3, hourly_rate), radius_km = COALESCE($4, radius_km),
				available = COALESCE($5, available), updated_at = now()
			WHERE user_id = $1`, id, req.Headline, req.HourlyRate, req.RadiusKm, req.Available)
		if err != nil {
			return err
		}
		if req.SkillIDs == nil {
			return nil
		}
		// Keep verification on skills the worker still has.
		if _, err := tx.Exec(r.Context(), `DELETE FROM user_skills WHERE user_id = $1 AND NOT (skill_id = ANY($2))`, id, req.SkillIDs); err != nil {
			return err
		}
		_, err = tx.Exec(r.Context(), `INSERT INTO user_skills (user_id, skill_id)
			SELECT $1, s.id FROM skills s WHERE s.id = ANY($2) AND s.active ON CONFLICT DO NOTHING`, id, req.SkillIDs)
		return err
	})
	if err != nil {
		return err
	}
	return s.getMe(w, r)
}

// updateAvailability is the worker's online/offline switch; it also
// refreshes their position so nearby jobs are accurate.
func (s *Server) updateAvailability(w http.ResponseWriter, r *http.Request) error {
	var req struct {
		Available bool     `json:"available"`
		Latitude  *float64 `json:"latitude"`
		Longitude *float64 `json:"longitude"`
		Zone      *string  `json:"zone"`
	}
	if err := decode(r, &req); err != nil {
		return err
	}
	p := me(r)
	if !p.IsWorker {
		return forbidden("create a worker profile first")
	}
	if (req.Latitude == nil) != (req.Longitude == nil) {
		return badRequest("latitude and longitude go together")
	}
	err := s.inTx(r.Context(), func(tx pgx.Tx) error {
		if req.Latitude != nil {
			if _, err := tx.Exec(r.Context(), `UPDATE users SET latitude = $2, longitude = $3, zone = COALESCE($4, zone) WHERE id = $1`,
				p.ID, req.Latitude, req.Longitude, req.Zone); err != nil {
				return err
			}
		}
		if req.Available {
			var hasLocation bool
			if err := tx.QueryRow(r.Context(), `SELECT latitude IS NOT NULL FROM users WHERE id = $1`, p.ID).Scan(&hasLocation); err != nil {
				return err
			}
			if !hasLocation {
				return badRequest("set your location before going available")
			}
		}
		_, err := tx.Exec(r.Context(), `UPDATE worker_profiles SET available = $2, updated_at = now() WHERE user_id = $1`, p.ID, req.Available)
		return err
	})
	if err != nil {
		return err
	}
	return s.getMe(w, r)
}

func (s *Server) getUserProfile(w http.ResponseWriter, r *http.Request) error {
	u, err := s.loadUser(r.Context(), r.PathValue("id"), me(r).IsAdmin)
	if err != nil {
		return err
	}
	rows, err := s.db.Query(r.Context(), `SELECT rv.id, j.title, u.first_name, rv.reviewer_role, rv.rating, rv.comment, rv.created_at
		FROM reviews rv JOIN jobs j ON j.id = rv.job_id JOIN users u ON u.id = rv.reviewer_id
		WHERE rv.reviewee_id = $1 ORDER BY rv.created_at DESC LIMIT 10`, u.ID)
	if err != nil {
		return err
	}
	reviews, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (reviewView, error) {
		var v reviewView
		err := row.Scan(&v.ID, &v.JobTitle, &v.ReviewerName, &v.ReviewerRole, &v.Rating, &v.Comment, &v.CreatedAt)
		return v, err
	})
	if err != nil {
		return err
	}
	return writeJSON(w, http.StatusOK, map[string]any{"user": u, "reviews": reviews})
}
