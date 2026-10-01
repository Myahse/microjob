package api

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/jackc/pgx/v5"

	"github.com/myahse/microjob/backend/internal/domain"
)

// createReview lets each side rate the other once per finished job (§25).
// Aggregates are recomputed from stored reviews, so they always reflect
// real, completed jobs (§26).
func (s *Server) createReview(w http.ResponseWriter, r *http.Request) error {
	var req struct {
		Rating        int    `json:"rating"`
		Punctuality   *int   `json:"punctuality"`
		Communication *int   `json:"communication"`
		Quality       *int   `json:"quality"`
		Respect       *int   `json:"respect"`
		Payment       *int   `json:"payment"`
		Comment       string `json:"comment"`
	}
	if err := decode(r, &req); err != nil {
		return err
	}
	for _, v := range []*int{&req.Rating, req.Punctuality, req.Communication, req.Quality, req.Respect, req.Payment} {
		if v != nil && (*v < 1 || *v > 5) {
			return badRequest("ratings must be between 1 and 5")
		}
	}
	p := me(r)
	var notes []note
	var other string
	err := s.inTx(r.Context(), func(tx pgx.Tx) error {
		j, err := lockJob(r.Context(), tx, r.PathValue("id"))
		if err != nil {
			return err
		}
		if !domain.IsFinished(j.Status) || j.WorkerID == nil {
			return conflict("you can review once the job is completed")
		}
		var role string
		switch {
		case j.ClientID == p.ID:
			role, other = "client", *j.WorkerID
		case j.isWorker(p.ID):
			role, other = "worker", j.ClientID
		default:
			return forbidden("not your job")
		}
		_, err = tx.Exec(r.Context(), `INSERT INTO reviews (job_id, reviewer_id, reviewee_id, reviewer_role, rating,
				punctuality, communication, quality, respect, payment, comment)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`,
			j.ID, p.ID, other, role, req.Rating, req.Punctuality, req.Communication, req.Quality, req.Respect, req.Payment,
			strings.TrimSpace(req.Comment))
		if err != nil {
			return err
		}
		if role == "client" {
			_, err = tx.Exec(r.Context(), `UPDATE worker_profiles wp SET rating_avg = agg.avg, rating_count = agg.n
				FROM (SELECT COALESCE(round(avg(rating), 2), 0) AS avg, count(*) AS n FROM reviews
				      WHERE reviewee_id = $1 AND reviewer_role = 'client') agg
				WHERE wp.user_id = $1`, other)
		} else {
			_, err = tx.Exec(r.Context(), `UPDATE users u SET client_rating_avg = agg.avg, client_rating_count = agg.n
				FROM (SELECT COALESCE(round(avg(rating), 2), 0) AS avg, count(*) AS n FROM reviews
				      WHERE reviewee_id = $1 AND reviewer_role = 'worker') agg
				WHERE u.id = $1`, other)
		}
		if err != nil {
			return err
		}
		var count int
		if err := tx.QueryRow(r.Context(), `SELECT count(*) FROM reviews WHERE job_id = $1`, j.ID).Scan(&count); err != nil {
			return err
		}
		if count == 2 && j.Status == domain.StatusPaid {
			if err := setStatus(r.Context(), tx, j.ID, domain.StatusRated, p.ID, ""); err != nil {
				return err
			}
		}
		notes = append(notes, note{UserID: other, Kind: "review", JobID: j.ID,
			Title: fmt.Sprintf("⭐ Nouvelle évaluation : %d/5", req.Rating), Body: j.Title})
		return nil
	})
	if err != nil {
		return err
	}
	s.send(r.Context(), notes...)
	s.jobChanged(r.PathValue("id"), "", other)
	return s.getJob(w, r)
}

var reportTypes = map[string]bool{"not_done": true, "bad_quality": true, "payment": true, "behavior": true, "fraud": true, "other": true}

// createReport opens a dispute (§34). If funds are still in escrow the job
// moves to DISPUTED and the money stays blocked until an admin decides.
func (s *Server) createReport(w http.ResponseWriter, r *http.Request) error {
	var req struct {
		Type        string `json:"type"`
		Description string `json:"description"`
	}
	if err := decode(r, &req); err != nil {
		return err
	}
	if !reportTypes[req.Type] {
		return badRequest("unknown report type")
	}
	p := me(r)
	var notes []note
	var notify []string
	err := s.inTx(r.Context(), func(tx pgx.Tx) error {
		j, err := lockJob(r.Context(), tx, r.PathValue("id"))
		if err != nil {
			return err
		}
		if j.ClientID != p.ID && !j.isWorker(p.ID) {
			return forbidden("not your job")
		}
		if !domain.CanReport(j.Status) && j.Status != domain.StatusDisputed {
			return conflict("nothing to report on this job yet")
		}
		var open bool
		if err := tx.QueryRow(r.Context(), `SELECT EXISTS (SELECT 1 FROM reports WHERE job_id = $1 AND status = 'open')`, j.ID).Scan(&open); err != nil {
			return err
		}
		if open {
			return conflict("a report is already open for this job")
		}
		if _, err := tx.Exec(r.Context(), `INSERT INTO reports (job_id, reporter_id, type, description, status_before) VALUES ($1, $2, $3, $4, $5)`,
			j.ID, p.ID, req.Type, req.Description, string(j.Status)); err != nil {
			return err
		}
		var held bool
		if err := tx.QueryRow(r.Context(), `SELECT EXISTS (SELECT 1 FROM payments WHERE job_id = $1 AND status = 'held')`, j.ID).Scan(&held); err != nil {
			return err
		}
		if held && domain.IsActive(j.Status) {
			if err := setStatus(r.Context(), tx, j.ID, domain.StatusDisputed, p.ID, "Litige ouvert : "+req.Type); err != nil {
				return err
			}
		}
		other := j.ClientID
		if p.ID == j.ClientID && j.WorkerID != nil {
			other = *j.WorkerID
		}
		notify = append(notify, other)
		notes = append(notes, note{UserID: other, Kind: "report", JobID: j.ID, Title: "⚠️ Un problème a été signalé", Body: j.Title + " · un administrateur va examiner la situation."})
		rows, err := tx.Query(r.Context(), `SELECT id FROM users WHERE is_admin AND status = 'active'`)
		if err != nil {
			return err
		}
		admins, err := pgx.CollectRows(rows, pgx.RowTo[string])
		if err != nil {
			return err
		}
		for _, a := range admins {
			notes = append(notes, note{UserID: a, Kind: "admin_report", JobID: j.ID, Title: "🚩 Nouveau signalement : " + req.Type, Body: j.Title})
		}
		return nil
	})
	if err != nil {
		return err
	}
	s.send(r.Context(), notes...)
	s.jobChanged(r.PathValue("id"), "", notify...)
	return s.getJob(w, r)
}
