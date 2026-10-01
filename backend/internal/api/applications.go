package api

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/myahse/microjob/backend/internal/domain"
)

type applicationView struct {
	ID         string     `json:"id"`
	Price      int        `json:"price"`
	Message    string     `json:"message"`
	Status     string     `json:"status"`
	CreatedAt  time.Time  `json:"created_at"`
	Worker     *personRef `json:"worker,omitempty"`
	Headline   string     `json:"headline,omitempty"`
	DistanceKm *float64   `json:"distance_km,omitempty"`
}

func (s *Server) applicationOf(ctx context.Context, jobID, workerID string) (*applicationView, error) {
	a := &applicationView{}
	err := s.db.QueryRow(ctx, `SELECT id, price, message, status, created_at FROM job_applications WHERE job_id = $1 AND worker_id = $2`,
		jobID, workerID).Scan(&a.ID, &a.Price, &a.Message, &a.Status, &a.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return a, err
}

// listApplications shows the client the interested workers and their offers.
func (s *Server) listApplications(w http.ResponseWriter, r *http.Request) error {
	p := me(r)
	jobID := r.PathValue("id")
	var clientID string
	var lat, lng float64
	err := s.db.QueryRow(r.Context(), `SELECT client_id, latitude, longitude FROM jobs WHERE id = $1`, jobID).Scan(&clientID, &lat, &lng)
	if err != nil {
		return err
	}
	if clientID != p.ID && !p.IsAdmin {
		return forbidden("only the client can see applications")
	}
	rows, err := s.db.Query(r.Context(), `SELECT a.id, a.price, a.message, a.status, a.created_at,
			u.id, u.first_name, u.last_name, u.avatar_url, wp.rating_avg, wp.rating_count, wp.completed_jobs, wp.headline,
			u.latitude, u.longitude
		FROM job_applications a JOIN users u ON u.id = a.worker_id JOIN worker_profiles wp ON wp.user_id = u.id
		WHERE a.job_id = $1 ORDER BY (a.status = 'pending') DESC, a.price, wp.rating_avg DESC`, jobID)
	if err != nil {
		return err
	}
	apps, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (applicationView, error) {
		a := applicationView{Worker: &personRef{}}
		var wlat, wlng *float64
		err := row.Scan(&a.ID, &a.Price, &a.Message, &a.Status, &a.CreatedAt,
			&a.Worker.ID, &a.Worker.FirstName, &a.Worker.LastName, &a.Worker.AvatarURL, &a.Worker.Rating, &a.Worker.RatingCount,
			&a.Worker.CompletedJobs, &a.Headline, &wlat, &wlng)
		if wlat != nil && wlng != nil {
			d := domain.RoundDistance(domain.DistanceKm(lat, lng, *wlat, *wlng))
			a.DistanceKm = &d
		}
		return a, err
	})
	if err != nil {
		return err
	}
	return writeJSON(w, http.StatusOK, apps)
}

// applyToJob sends a worker's offer on a "Proposez votre prix" job (§11).
func (s *Server) applyToJob(w http.ResponseWriter, r *http.Request) error {
	var req struct {
		Price   int    `json:"price"`
		Message string `json:"message"`
	}
	if err := decode(r, &req); err != nil {
		return err
	}
	if req.Price <= 0 {
		return badRequest("price must be positive")
	}
	p := me(r)
	var notes []note
	var clientID string
	err := s.inTx(r.Context(), func(tx pgx.Tx) error {
		j, err := lockJob(r.Context(), tx, r.PathValue("id"))
		if err != nil {
			return err
		}
		clientID = j.ClientID
		switch {
		case j.ClientID == p.ID:
			return forbidden("you cannot apply to your own job")
		case !domain.IsOpen(j.Status):
			return conflict("job is no longer available")
		case j.PricingMode != "offers":
			return badRequest("this job has a fixed price: accept it directly")
		}
		if err := workerEligible(r.Context(), tx, p.ID, j.CategoryID, j.ID); err != nil {
			return err
		}
		_, err = tx.Exec(r.Context(), `INSERT INTO job_applications (job_id, worker_id, price, message) VALUES ($1, $2, $3, $4)
			ON CONFLICT (job_id, worker_id) DO UPDATE SET price = EXCLUDED.price, message = EXCLUDED.message, status = 'pending', created_at = now()`,
			j.ID, p.ID, req.Price, req.Message)
		if err != nil {
			return err
		}
		notes = append(notes, note{UserID: j.ClientID, Kind: "new_offer", JobID: j.ID,
			Title: "💬 Nouvelle offre : " + formatFCFA(req.Price), Body: p.Name + " · " + j.Title})
		return nil
	})
	if err != nil {
		return err
	}
	s.send(r.Context(), notes...)
	s.jobChanged(r.PathValue("id"), "", clientID)
	return s.getJob(w, r)
}

// chooseApplication is the client picking a worker's offer.
func (s *Server) chooseApplication(w http.ResponseWriter, r *http.Request) error {
	p := me(r)
	var notes []note
	var workerID string
	err := s.inTx(r.Context(), func(tx pgx.Tx) error {
		j, err := lockJob(r.Context(), tx, r.PathValue("id"))
		if err != nil {
			return err
		}
		if j.ClientID != p.ID {
			return forbidden("only the client can choose a worker")
		}
		if !domain.IsOpen(j.Status) {
			return conflict("a worker is already assigned")
		}
		var price int
		var status string
		err = tx.QueryRow(r.Context(), `SELECT worker_id, price, status FROM job_applications WHERE id = $1 AND job_id = $2`,
			r.PathValue("appId"), j.ID).Scan(&workerID, &price, &status)
		if errors.Is(err, pgx.ErrNoRows) {
			return notFound("application not found")
		}
		if err != nil {
			return err
		}
		if status != "pending" {
			return conflict("this offer is no longer available")
		}
		if err := workerEligible(r.Context(), tx, workerID, j.CategoryID, j.ID); err != nil {
			return conflict("this worker can no longer take the job")
		}
		notes, err = s.assign(r.Context(), tx, j, workerID, price)
		return err
	})
	if err != nil {
		return err
	}
	s.send(r.Context(), notes...)
	s.jobChanged(r.PathValue("id"), string(domain.StatusAssigned), workerID)
	return s.getJob(w, r)
}
