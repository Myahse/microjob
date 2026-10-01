package api

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/myahse/microjob/backend/internal/domain"
)

// ---- statistics (§33, §41) ----

func (s *Server) adminStats(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	stats := map[string]any{}
	var posted, accepted, completed, cancelled, activeWorkers, availableWorkers, activeClients, users, openReports, pendingWithdrawals int
	var gmv, commission int
	var avgAssignMinutes *float64
	err := s.db.QueryRow(ctx, `SELECT
			(SELECT count(*) FROM jobs),
			(SELECT count(*) FROM jobs WHERE worker_id IS NOT NULL OR status IN ('COMPLETED', 'PAID', 'RATED')),
			(SELECT count(*) FROM jobs WHERE status IN ('COMPLETED', 'PAID', 'RATED')),
			(SELECT count(*) FROM jobs WHERE status = 'CANCELLED'),
			(SELECT count(DISTINCT worker_id) FROM job_assignments WHERE assigned_at > now() - interval '30 days'),
			(SELECT count(*) FROM worker_profiles WHERE available),
			(SELECT count(DISTINCT client_id) FROM jobs WHERE created_at > now() - interval '30 days'),
			(SELECT count(*) FROM users),
			(SELECT count(*) FROM reports WHERE status = 'open'),
			(SELECT count(*) FROM withdrawals WHERE status = 'pending'),
			(SELECT COALESCE(sum(amount - refunded_amount), 0) FROM payments WHERE status IN ('released', 'partially_refunded')),
			(SELECT COALESCE(sum(platform_fee), 0) FROM payments WHERE status IN ('released', 'partially_refunded')),
			(SELECT avg(EXTRACT(EPOCH FROM (a.assigned_at - j.created_at)) / 60) FROM job_assignments a JOIN jobs j ON j.id = a.job_id)`).
		Scan(&posted, &accepted, &completed, &cancelled, &activeWorkers, &availableWorkers, &activeClients, &users,
			&openReports, &pendingWithdrawals, &gmv, &commission, &avgAssignMinutes)
	if err != nil {
		return err
	}
	rate := func(a, b int) float64 {
		if b == 0 {
			return 0
		}
		return float64(int(float64(a)/float64(b)*1000)) / 10
	}
	stats["jobs_posted"] = posted
	stats["jobs_accepted"] = accepted
	stats["jobs_completed"] = completed
	stats["jobs_cancelled"] = cancelled
	stats["acceptance_rate"] = rate(accepted, posted)
	stats["completion_rate"] = rate(completed, accepted)
	stats["active_workers"] = activeWorkers
	stats["available_workers"] = availableWorkers
	stats["active_clients"] = activeClients
	stats["users"] = users
	stats["open_reports"] = openReports
	stats["pending_withdrawals"] = pendingWithdrawals
	stats["gmv"] = gmv
	stats["commission"] = commission
	stats["avg_minutes_to_assignment"] = avgAssignMinutes

	type day struct {
		Day       string `json:"day"`
		Posted    int    `json:"posted"`
		Completed int    `json:"completed"`
		GMV       int    `json:"gmv"`
	}
	rows, err := s.db.Query(ctx, `SELECT to_char(d, 'YYYY-MM-DD'),
			(SELECT count(*) FROM jobs WHERE created_at::date = d),
			(SELECT count(*) FROM job_status_history WHERE status = 'COMPLETED' AND created_at::date = d),
			(SELECT COALESCE(sum(amount - refunded_amount), 0) FROM payments
			 WHERE status IN ('released', 'partially_refunded') AND updated_at::date = d)
		FROM generate_series(current_date - 13, current_date, interval '1 day') AS d`)
	if err != nil {
		return err
	}
	stats["daily"], err = pgx.CollectRows(rows, func(row pgx.CollectableRow) (day, error) {
		var d day
		err := row.Scan(&d.Day, &d.Posted, &d.Completed, &d.GMV)
		return d, err
	})
	if err != nil {
		return err
	}

	type catStat struct {
		Name string `json:"name"`
		Icon string `json:"icon"`
		Jobs int    `json:"jobs"`
		Done int    `json:"done"`
	}
	rows, err = s.db.Query(ctx, `SELECT c.name, c.icon, count(j.id), count(j.id) FILTER (WHERE j.status IN ('COMPLETED', 'PAID', 'RATED'))
		FROM categories c LEFT JOIN jobs j ON j.category_id = c.id GROUP BY c.id ORDER BY c.sort_order`)
	if err != nil {
		return err
	}
	stats["by_category"], err = pgx.CollectRows(rows, func(row pgx.CollectableRow) (catStat, error) {
		var c catStat
		err := row.Scan(&c.Name, &c.Icon, &c.Jobs, &c.Done)
		return c, err
	})
	if err != nil {
		return err
	}
	return writeJSON(w, http.StatusOK, stats)
}

// ---- users ----

func (s *Server) adminListUsers(w http.ResponseWriter, r *http.Request) error {
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	role := r.URL.Query().Get("role")
	rows, err := s.db.Query(r.Context(), `SELECT id FROM users
		WHERE ($1 = '' OR first_name ILIKE '%'||$1||'%' OR last_name ILIKE '%'||$1||'%' OR username ILIKE '%'||$1||'%'
		       OR email ILIKE '%'||$1||'%' OR phone ILIKE '%'||$1||'%')
		  AND ($2 = '' OR ($2 = 'worker' AND is_worker) OR ($2 = 'client' AND is_client) OR ($2 = 'admin' AND is_admin)
		       OR ($2 = 'suspended' AND status <> 'active'))
		ORDER BY created_at DESC LIMIT 200`, q, role)
	if err != nil {
		return err
	}
	ids, err := pgx.CollectRows(rows, pgx.RowTo[string])
	if err != nil {
		return err
	}
	out := make([]*userView, 0, len(ids))
	for _, id := range ids {
		u, err := s.loadUser(r.Context(), id, true)
		if err != nil {
			return err
		}
		out = append(out, u)
	}
	return writeJSON(w, http.StatusOK, out)
}

// adminUpdateUser suspends, bans, reactivates or verifies a user.
func (s *Server) adminUpdateUser(w http.ResponseWriter, r *http.Request) error {
	var req struct {
		Status           *string `json:"status"`
		IdentityVerified *bool   `json:"identity_verified"`
		PhoneVerified    *bool   `json:"phone_verified"`
		EmailVerified    *bool   `json:"email_verified"`
		IsAdmin          *bool   `json:"is_admin"`
	}
	if err := decode(r, &req); err != nil {
		return err
	}
	id := r.PathValue("id")
	if req.Status != nil && *req.Status != "active" && *req.Status != "suspended" && *req.Status != "banned" {
		return badRequest("status must be active, suspended or banned")
	}
	if id == me(r).ID && ((req.Status != nil && *req.Status != "active") || (req.IsAdmin != nil && !*req.IsAdmin)) {
		return badRequest("you cannot suspend or demote yourself")
	}
	err := s.inTx(r.Context(), func(tx pgx.Tx) error {
		tag, err := tx.Exec(r.Context(), `UPDATE users SET status = COALESCE($2, status),
				identity_verified = COALESCE($3, identity_verified), phone_verified = COALESCE($4, phone_verified),
				email_verified = COALESCE($5, email_verified), is_admin = COALESCE($6, is_admin), updated_at = now()
			WHERE id = $1`, id, req.Status, req.IdentityVerified, req.PhoneVerified, req.EmailVerified, req.IsAdmin)
		if err != nil {
			return err
		}
		if tag.RowsAffected() == 0 {
			return notFound("user not found")
		}
		if req.Status != nil && *req.Status != "active" {
			// A suspended worker stops receiving jobs.
			_, err = tx.Exec(r.Context(), `UPDATE worker_profiles SET available = FALSE WHERE user_id = $1`, id)
		}
		return err
	})
	if err != nil {
		return err
	}
	u, err := s.loadUser(r.Context(), id, true)
	if err != nil {
		return err
	}
	return writeJSON(w, http.StatusOK, u)
}

func (s *Server) adminVerifySkill(w http.ResponseWriter, r *http.Request) error {
	var req struct {
		Verified bool `json:"verified"`
	}
	if err := decode(r, &req); err != nil {
		return err
	}
	tag, err := s.db.Exec(r.Context(), `UPDATE user_skills SET verified = $3 WHERE user_id = $1 AND skill_id = $2`,
		r.PathValue("id"), r.PathValue("skillId"), req.Verified)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return notFound("user does not have this skill")
	}
	u, err := s.loadUser(r.Context(), r.PathValue("id"), true)
	if err != nil {
		return err
	}
	return writeJSON(w, http.StatusOK, u)
}

// ---- jobs ----

func (s *Server) adminListJobs(w http.ResponseWriter, r *http.Request) error {
	status := r.URL.Query().Get("status")
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	jobs, err := s.queryJobs(r.Context(), `WHERE ($1 = '' OR j.status = $1) AND ($2 = '' OR j.title ILIKE '%'||$2||'%' OR j.zone ILIKE '%'||$2||'%')
		ORDER BY j.created_at DESC LIMIT 200`, status, q)
	if err != nil {
		return err
	}
	out, err := s.presentAll(r, jobs)
	if err != nil {
		return err
	}
	return writeJSON(w, http.StatusOK, out)
}

// adminCancelJob moderates a job: it is cancelled and any escrow refunded.
func (s *Server) adminCancelJob(w http.ResponseWriter, r *http.Request) error {
	var req struct {
		Reason string `json:"reason"`
	}
	if err := decode(r, &req); err != nil {
		return err
	}
	if strings.TrimSpace(req.Reason) == "" {
		return badRequest("a reason is required")
	}
	p := me(r)
	var notes []note
	var notify []string
	err := s.inTx(r.Context(), func(tx pgx.Tx) error {
		j, err := lockJob(r.Context(), tx, r.PathValue("id"))
		if err != nil {
			return err
		}
		if domain.IsFinished(j.Status) || j.Status == domain.StatusCancelled {
			return conflict("job is already closed")
		}
		if err := s.refundHeld(r.Context(), tx, j.ID); err != nil {
			return err
		}
		if _, err := tx.Exec(r.Context(), `UPDATE job_assignments SET cancelled_at = now() WHERE job_id = $1 AND cancelled_at IS NULL`, j.ID); err != nil {
			return err
		}
		if _, err := tx.Exec(r.Context(), `INSERT INTO cancellations (job_id, user_id, role, reason, status_at_cancel) VALUES ($1, $2, 'admin', $3, $4)`,
			j.ID, p.ID, req.Reason, string(j.Status)); err != nil {
			return err
		}
		if _, err := tx.Exec(r.Context(), `UPDATE reports SET status = 'resolved', resolution = 'cancelled', resolved_by = $2, resolved_at = now()
			WHERE job_id = $1 AND status = 'open'`, j.ID, p.ID); err != nil {
			return err
		}
		if err := setStatus(r.Context(), tx, j.ID, domain.StatusCancelled, p.ID, "Annulé par la modération : "+req.Reason); err != nil {
			return err
		}
		for _, u := range []*string{&j.ClientID, j.WorkerID} {
			if u != nil {
				notify = append(notify, *u)
				notes = append(notes, note{UserID: *u, Kind: "job_cancelled", JobID: j.ID, Title: "❌ Job annulé par la modération", Body: j.Title + " · " + req.Reason})
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	s.send(r.Context(), notes...)
	s.jobChanged(r.PathValue("id"), string(domain.StatusCancelled), notify...)
	return s.getJob(w, r)
}

// ---- categories & skills ----

func (s *Server) adminListCategories(w http.ResponseWriter, r *http.Request) error {
	cats, err := s.categories(r, true)
	if err != nil {
		return err
	}
	return writeJSON(w, http.StatusOK, cats)
}

type categoryInput struct {
	Slug           *string  `json:"slug"`
	Name           *string  `json:"name"`
	Icon           *string  `json:"icon"`
	CommissionRate *float64 `json:"commission_rate"`
	ClearRate      bool     `json:"clear_commission_rate"`
	Active         *bool    `json:"active"`
	SortOrder      *int     `json:"sort_order"`
}

func (c categoryInput) validate() error {
	if c.CommissionRate != nil && (*c.CommissionRate < 0 || *c.CommissionRate > 0.5) {
		return badRequest("commission rate must be between 0 and 0.5")
	}
	return nil
}

func (s *Server) adminCreateCategory(w http.ResponseWriter, r *http.Request) error {
	var req categoryInput
	if err := decode(r, &req); err != nil {
		return err
	}
	if err := req.validate(); err != nil {
		return err
	}
	if req.Name == nil || strings.TrimSpace(*req.Name) == "" {
		return badRequest("name is required")
	}
	slug := strings.ToLower(strings.ReplaceAll(strings.TrimSpace(*req.Name), " ", "-"))
	if req.Slug != nil && *req.Slug != "" {
		slug = *req.Slug
	}
	_, err := s.db.Exec(r.Context(), `INSERT INTO categories (slug, name, icon, commission_rate, sort_order)
		VALUES ($1, $2, COALESCE($3, ''), $4, COALESCE($5, (SELECT COALESCE(max(sort_order), 0) + 1 FROM categories)))`,
		slug, strings.TrimSpace(*req.Name), req.Icon, req.CommissionRate, req.SortOrder)
	if err != nil {
		return err
	}
	return s.adminListCategories(w, r)
}

func (s *Server) adminUpdateCategory(w http.ResponseWriter, r *http.Request) error {
	var req categoryInput
	if err := decode(r, &req); err != nil {
		return err
	}
	if err := req.validate(); err != nil {
		return err
	}
	tag, err := s.db.Exec(r.Context(), `UPDATE categories SET name = COALESCE($2, name), icon = COALESCE($3, icon),
			commission_rate = CASE WHEN $7 THEN NULL ELSE COALESCE($4, commission_rate) END,
			active = COALESCE($5, active), sort_order = COALESCE($6, sort_order)
		WHERE id = $1`, r.PathValue("id"), req.Name, req.Icon, req.CommissionRate, req.Active, req.SortOrder, req.ClearRate)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return notFound("category not found")
	}
	return s.adminListCategories(w, r)
}

func (s *Server) adminCreateSkill(w http.ResponseWriter, r *http.Request) error {
	var req struct {
		Name string `json:"name"`
	}
	if err := decode(r, &req); err != nil {
		return err
	}
	if strings.TrimSpace(req.Name) == "" {
		return badRequest("name is required")
	}
	if _, err := s.db.Exec(r.Context(), `INSERT INTO skills (category_id, name) VALUES ($1, $2)`, r.PathValue("id"), strings.TrimSpace(req.Name)); err != nil {
		return err
	}
	return s.adminListCategories(w, r)
}

func (s *Server) adminUpdateSkill(w http.ResponseWriter, r *http.Request) error {
	var req struct {
		Name   *string `json:"name"`
		Active *bool   `json:"active"`
	}
	if err := decode(r, &req); err != nil {
		return err
	}
	tag, err := s.db.Exec(r.Context(), `UPDATE skills SET name = COALESCE($2, name), active = COALESCE($3, active) WHERE id = $1`,
		r.PathValue("id"), req.Name, req.Active)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return notFound("skill not found")
	}
	return s.adminListCategories(w, r)
}

// ---- reports & disputes (§34) ----

type reportView struct {
	ID             string     `json:"id"`
	JobID          string     `json:"job_id"`
	JobTitle       string     `json:"job_title"`
	JobStatus      string     `json:"job_status"`
	ReporterID     string     `json:"reporter_id"`
	ReporterName   string     `json:"reporter_name"`
	ReporterRole   string     `json:"reporter_role"`
	Type           string     `json:"type"`
	Description    string     `json:"description"`
	Status         string     `json:"status"`
	Resolution     *string    `json:"resolution"`
	RefundAmount   *int       `json:"refund_amount"`
	ResolutionNote string     `json:"resolution_note"`
	HeldAmount     *int       `json:"held_amount"`
	CreatedAt      time.Time  `json:"created_at"`
	ResolvedAt     *time.Time `json:"resolved_at"`
}

func (s *Server) adminListReports(w http.ResponseWriter, r *http.Request) error {
	status := r.URL.Query().Get("status")
	rows, err := s.db.Query(r.Context(), `SELECT rp.id, j.id, j.title, j.status, u.id, u.first_name || ' ' || u.last_name,
			CASE WHEN u.id = j.client_id THEN 'client' ELSE 'worker' END,
			rp.type, rp.description, rp.status, rp.resolution, rp.refund_amount, rp.resolution_note,
			(SELECT amount FROM payments p WHERE p.job_id = j.id AND p.status = 'held'),
			rp.created_at, rp.resolved_at
		FROM reports rp JOIN jobs j ON j.id = rp.job_id JOIN users u ON u.id = rp.reporter_id
		WHERE $1 = '' OR rp.status = $1
		ORDER BY (rp.status = 'open') DESC, rp.created_at DESC LIMIT 200`, status)
	if err != nil {
		return err
	}
	out, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (reportView, error) {
		var v reportView
		err := row.Scan(&v.ID, &v.JobID, &v.JobTitle, &v.JobStatus, &v.ReporterID, &v.ReporterName, &v.ReporterRole,
			&v.Type, &v.Description, &v.Status, &v.Resolution, &v.RefundAmount, &v.ResolutionNote, &v.HeldAmount, &v.CreatedAt, &v.ResolvedAt)
		return v, err
	})
	if err != nil {
		return err
	}
	return writeJSON(w, http.StatusOK, out)
}

// adminResolveReport settles a dispute: refund the client, pay the worker,
// split, or dismiss the report and resume the job.
func (s *Server) adminResolveReport(w http.ResponseWriter, r *http.Request) error {
	var req struct {
		Resolution   string `json:"resolution"` // refund_client | pay_worker | partial | dismiss
		RefundAmount int    `json:"refund_amount"`
		Note         string `json:"note"`
	}
	if err := decode(r, &req); err != nil {
		return err
	}
	p := me(r)
	var notes []note
	var notify []string
	var jobID string
	err := s.inTx(r.Context(), func(tx pgx.Tx) error {
		var status, statusBefore string
		err := tx.QueryRow(r.Context(), `SELECT job_id, status, status_before FROM reports WHERE id = $1 FOR UPDATE`, r.PathValue("id")).
			Scan(&jobID, &status, &statusBefore)
		if err != nil {
			return err
		}
		if status != "open" {
			return conflict("report already handled")
		}
		j, err := lockJob(r.Context(), tx, jobID)
		if err != nil {
			return err
		}
		var held, amount int
		if err := tx.QueryRow(r.Context(), `SELECT count(*), COALESCE(max(amount), 0) FROM payments WHERE job_id = $1 AND status = 'held'`, jobID).Scan(&held, &amount); err != nil {
			return err
		}
		needsFunds := req.Resolution == "refund_client" || req.Resolution == "pay_worker" || req.Resolution == "partial"
		if needsFunds && held == 0 {
			return conflict("no funds are held for this job; dismiss the report or handle it manually")
		}
		reportStatus := "resolved"
		var refund *int
		summary := ""
		switch req.Resolution {
		case "refund_client":
			if err := s.refundHeld(r.Context(), tx, jobID); err != nil {
				return err
			}
			if _, err := tx.Exec(r.Context(), `UPDATE job_assignments SET cancelled_at = now() WHERE job_id = $1 AND cancelled_at IS NULL`, jobID); err != nil {
				return err
			}
			if err := setStatus(r.Context(), tx, jobID, domain.StatusCancelled, p.ID, "Litige : client remboursé"); err != nil {
				return err
			}
			refund = &amount
			summary = "Le client a été remboursé de " + formatFCFA(amount)
		case "pay_worker", "partial":
			value := 0
			if req.Resolution == "partial" {
				if req.RefundAmount <= 0 || req.RefundAmount >= amount {
					return badRequest(fmt.Sprintf("refund_amount must be between 1 and %d", amount-1))
				}
				value = req.RefundAmount
				refund = &value
			}
			if err := addHistory(r.Context(), tx, jobID, domain.StatusCompleted, p.ID, "Litige résolu par l'administration"); err != nil {
				return err
			}
			n, err := s.settle(r.Context(), tx, jobID, p.ID, value)
			if err != nil {
				return err
			}
			notes = append(notes, n)
			summary = "Le worker a été payé"
			if value > 0 {
				summary = fmt.Sprintf("Remboursement partiel de %s au client, le reste payé au worker", formatFCFA(value))
			}
		case "dismiss":
			reportStatus = "rejected"
			if j.Status == domain.StatusDisputed {
				if err := setStatus(r.Context(), tx, jobID, domain.JobStatus(statusBefore), p.ID, "Signalement classé, reprise du job"); err != nil {
					return err
				}
			}
			summary = "Le signalement a été classé"
		default:
			return badRequest("resolution must be refund_client, pay_worker, partial or dismiss")
		}
		if _, err := tx.Exec(r.Context(), `UPDATE reports SET status = $2, resolution = $3, refund_amount = $4, resolution_note = $5,
				resolved_by = $6, resolved_at = now() WHERE id = $1`,
			r.PathValue("id"), reportStatus, req.Resolution, refund, req.Note, p.ID); err != nil {
			return err
		}
		for _, u := range []*string{&j.ClientID, j.WorkerID} {
			if u != nil {
				notify = append(notify, *u)
				notes = append(notes, note{UserID: *u, Kind: "report_resolved", JobID: jobID, Title: "⚖️ Litige traité", Body: summary})
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	s.send(r.Context(), notes...)
	s.jobChanged(jobID, "", notify...)
	return s.adminListReports(w, r)
}

// ---- payments, withdrawals, settings ----

func (s *Server) adminListPayments(w http.ResponseWriter, r *http.Request) error {
	type row struct {
		paymentView
		JobID      string `json:"job_id"`
		JobTitle   string `json:"job_title"`
		ClientName string `json:"client_name"`
		WorkerName string `json:"worker_name"`
	}
	rows, err := s.db.Query(r.Context(), `SELECT p.id, p.amount, p.platform_fee, p.worker_amount, p.refunded_amount, p.commission_rate,
			p.provider, p.status, p.created_at, j.id, j.title, c.first_name || ' ' || c.last_name, wk.first_name || ' ' || wk.last_name
		FROM payments p JOIN jobs j ON j.id = p.job_id JOIN users c ON c.id = p.payer_id JOIN users wk ON wk.id = p.worker_id
		WHERE $1 = '' OR p.status = $1
		ORDER BY p.created_at DESC LIMIT 200`, r.URL.Query().Get("status"))
	if err != nil {
		return err
	}
	out, err := pgx.CollectRows(rows, func(rr pgx.CollectableRow) (row, error) {
		var v row
		err := rr.Scan(&v.ID, &v.Amount, &v.PlatformFee, &v.WorkerAmount, &v.RefundedAmount, &v.CommissionRate, &v.Provider, &v.Status,
			&v.CreatedAt, &v.JobID, &v.JobTitle, &v.ClientName, &v.WorkerName)
		return v, err
	})
	if err != nil {
		return err
	}
	return writeJSON(w, http.StatusOK, out)
}

func (s *Server) adminListWithdrawals(w http.ResponseWriter, r *http.Request) error {
	rows, err := s.db.Query(r.Context(), `SELECT wd.id, u.id, u.first_name || ' ' || u.last_name, wd.amount, wd.method, wd.destination,
			wd.status, wd.created_at, wd.processed_at
		FROM withdrawals wd JOIN users u ON u.id = wd.user_id
		WHERE $1 = '' OR wd.status = $1
		ORDER BY (wd.status = 'pending') DESC, wd.created_at DESC LIMIT 200`, r.URL.Query().Get("status"))
	if err != nil {
		return err
	}
	out, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (withdrawalView, error) {
		var v withdrawalView
		err := row.Scan(&v.ID, &v.UserID, &v.UserName, &v.Amount, &v.Method, &v.Destination, &v.Status, &v.CreatedAt, &v.ProcessedAt)
		return v, err
	})
	if err != nil {
		return err
	}
	return writeJSON(w, http.StatusOK, out)
}

// adminProcessWithdrawal marks a payout as paid, or rejects it and credits
// the amount back to the wallet.
func (s *Server) adminProcessWithdrawal(w http.ResponseWriter, r *http.Request) error {
	var req struct {
		Status string `json:"status"` // paid | rejected
	}
	if err := decode(r, &req); err != nil {
		return err
	}
	if req.Status != "paid" && req.Status != "rejected" {
		return badRequest("status must be paid or rejected")
	}
	var notes []note
	err := s.inTx(r.Context(), func(tx pgx.Tx) error {
		var userID, status string
		var amount int
		err := tx.QueryRow(r.Context(), `SELECT user_id, amount, status FROM withdrawals WHERE id = $1 FOR UPDATE`, r.PathValue("id")).Scan(&userID, &amount, &status)
		if err != nil {
			return err
		}
		if status != "pending" {
			return conflict("withdrawal already processed")
		}
		if _, err := tx.Exec(r.Context(), `UPDATE withdrawals SET status = $2, processed_at = now() WHERE id = $1`, r.PathValue("id"), req.Status); err != nil {
			return err
		}
		if req.Status == "rejected" {
			if _, err := tx.Exec(r.Context(), `INSERT INTO wallet_transactions (user_id, amount, kind, description) VALUES ($1, $2, 'withdrawal_reversal', 'Retrait refusé')`,
				userID, amount); err != nil {
				return err
			}
			notes = append(notes, note{UserID: userID, Kind: "payment", Title: "Retrait refusé", Body: formatFCFA(amount) + " recrédités sur votre solde"})
		} else {
			notes = append(notes, note{UserID: userID, Kind: "payment", Title: "💸 Retrait envoyé", Body: formatFCFA(amount)})
		}
		return nil
	})
	if err != nil {
		return err
	}
	s.send(r.Context(), notes...)
	return s.adminListWithdrawals(w, r)
}

var editableSettings = map[string]func(string) error{
	"default_commission_rate": func(v string) error {
		f, err := strconv.ParseFloat(v, 64)
		if err != nil || f < 0 || f > 0.5 {
			return badRequest("default_commission_rate must be between 0 and 0.5")
		}
		return nil
	},
	"matching_max_notified": func(v string) error {
		n, err := strconv.Atoi(v)
		if err != nil || n < 1 || n > 500 {
			return badRequest("matching_max_notified must be between 1 and 500")
		}
		return nil
	},
	"default_worker_radius_km": func(v string) error {
		f, err := strconv.ParseFloat(v, 64)
		if err != nil || f < 1 || f > 50 {
			return badRequest("default_worker_radius_km must be between 1 and 50")
		}
		return nil
	},
}

func (s *Server) adminGetSettings(w http.ResponseWriter, r *http.Request) error {
	rows, err := s.db.Query(r.Context(), `SELECT key, value FROM settings`)
	if err != nil {
		return err
	}
	out := map[string]string{}
	var k, v string
	_, err = pgx.ForEachRow(rows, []any{&k, &v}, func() error { out[k] = v; return nil })
	if err != nil {
		return err
	}
	return writeJSON(w, http.StatusOK, out)
}

func (s *Server) adminUpdateSettings(w http.ResponseWriter, r *http.Request) error {
	var req map[string]string
	if err := decode(r, &req); err != nil {
		return err
	}
	for k, v := range req {
		check, ok := editableSettings[k]
		if !ok {
			return badRequest("unknown setting " + k)
		}
		if err := check(v); err != nil {
			return err
		}
	}
	err := s.inTx(r.Context(), func(tx pgx.Tx) error {
		for k, v := range req {
			if _, err := tx.Exec(r.Context(), `UPDATE settings SET value = $2, updated_at = now() WHERE key = $1`, k, v); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	return s.adminGetSettings(w, r)
}
