package api

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/myahse/microjob/backend/internal/domain"
	"github.com/myahse/microjob/backend/internal/payments"
)

type personRef struct {
	ID            string  `json:"id"`
	FirstName     string  `json:"first_name"`
	LastName      string  `json:"last_name"`
	AvatarURL     string  `json:"avatar_url"`
	Rating        float64 `json:"rating"`
	RatingCount   int     `json:"rating_count"`
	CompletedJobs int     `json:"completed_jobs,omitempty"`
}

type categoryRef struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
	Icon string `json:"icon"`
}

type assignmentView struct {
	AssignedAt  time.Time  `json:"assigned_at"`
	OnTheWayAt  *time.Time `json:"on_the_way_at"`
	ArrivedAt   *time.Time `json:"arrived_at"`
	StartedAt   *time.Time `json:"started_at"`
	FinishedAt  *time.Time `json:"finished_at"`
	CompletedAt *time.Time `json:"completed_at"`
	// Live position of the worker, shown to the client once assigned (§17).
	WorkerLatitude   *float64   `json:"worker_latitude"`
	WorkerLongitude  *float64   `json:"worker_longitude"`
	WorkerDistanceKm *float64   `json:"worker_distance_km"`
	LocationAt       *time.Time `json:"location_updated_at"`
}

type paymentView struct {
	ID             string    `json:"id"`
	Amount         int       `json:"amount"`
	PlatformFee    int       `json:"platform_fee"`
	WorkerAmount   int       `json:"worker_amount"`
	RefundedAmount int       `json:"refunded_amount"`
	CommissionRate float64   `json:"commission_rate"`
	Provider       string    `json:"provider"`
	Status         string    `json:"status"`
	CreatedAt      time.Time `json:"created_at"`
}

type historyItem struct {
	Status    string    `json:"status"`
	Note      string    `json:"note"`
	CreatedAt time.Time `json:"created_at"`
}

type jobView struct {
	ID                string      `json:"id"`
	Title             string      `json:"title"`
	Description       string      `json:"description"`
	Category          categoryRef `json:"category"`
	SkillID           *int        `json:"skill_id"`
	SkillName         *string     `json:"skill_name"`
	Zone              string      `json:"zone"`
	PricingMode       string      `json:"pricing_mode"`
	Budget            int         `json:"budget"`
	AgreedPrice       *int        `json:"agreed_price"`
	ScheduledAt       *time.Time  `json:"scheduled_at"`
	DurationMinutes   int         `json:"duration_minutes"`
	Photos            []string    `json:"photos"`
	Status            string      `json:"status"`
	CreatedAt         time.Time   `json:"created_at"`
	UpdatedAt         time.Time   `json:"updated_at"`
	Client            personRef   `json:"client"`
	Worker            *personRef  `json:"worker"`
	ApplicationsCount int         `json:"applications_count"`

	// Viewer-dependent fields.
	MyRole     string   `json:"my_role"` // client | worker | candidate | admin
	Address    *string  `json:"address,omitempty"`
	Latitude   *float64 `json:"latitude,omitempty"`
	Longitude  *float64 `json:"longitude,omitempty"`
	DistanceKm *float64 `json:"distance_km,omitempty"`

	// Detail-only fields.
	Assignment      *assignmentView  `json:"assignment,omitempty"`
	Payment         *paymentView     `json:"payment,omitempty"`
	MyApplication   *applicationView `json:"my_application,omitempty"`
	ReviewedByMe    bool             `json:"reviewed_by_me"`
	ReviewedByOther bool             `json:"reviewed_by_other"`
	OpenReport      bool             `json:"open_report"`
	HasConversation bool             `json:"has_conversation"`
	History         []historyItem    `json:"history,omitempty"`
}

// jobRow is a job with its joins, before privacy rules are applied.
type jobRow struct {
	jobView
	clientID  string
	workerID  *string
	address   string
	latitude  float64
	longitude float64
}

const jobSelect = `SELECT j.id, j.client_id, j.category_id, c.name, c.icon, j.skill_id, sk.name, j.title, j.description,
		j.latitude, j.longitude, j.address, j.zone, j.pricing_mode, j.budget, j.agreed_price, j.scheduled_at,
		j.duration_minutes, j.photos, j.status, j.worker_id, j.created_at, j.updated_at,
		cu.first_name, cu.last_name, cu.avatar_url, cu.client_rating_avg, cu.client_rating_count,
		wu.first_name, wu.last_name, wu.avatar_url, wp.rating_avg, wp.rating_count, wp.completed_jobs,
		(SELECT count(*) FROM job_applications a WHERE a.job_id = j.id AND a.status = 'pending')
	FROM jobs j
	JOIN categories c ON c.id = j.category_id
	LEFT JOIN skills sk ON sk.id = j.skill_id
	JOIN users cu ON cu.id = j.client_id
	LEFT JOIN users wu ON wu.id = j.worker_id
	LEFT JOIN worker_profiles wp ON wp.user_id = j.worker_id`

func scanJobRow(row pgx.Row) (*jobRow, error) {
	j := &jobRow{}
	var wFirst, wLast, wAvatar *string
	var wRating *float64
	var wCount, wCompleted *int
	err := row.Scan(&j.ID, &j.clientID, &j.Category.ID, &j.Category.Name, &j.Category.Icon, &j.SkillID, &j.SkillName,
		&j.Title, &j.Description, &j.latitude, &j.longitude, &j.address, &j.Zone, &j.PricingMode, &j.Budget,
		&j.AgreedPrice, &j.ScheduledAt, &j.DurationMinutes, &j.Photos, &j.Status, &j.workerID, &j.CreatedAt, &j.UpdatedAt,
		&j.Client.FirstName, &j.Client.LastName, &j.Client.AvatarURL, &j.Client.Rating, &j.Client.RatingCount,
		&wFirst, &wLast, &wAvatar, &wRating, &wCount, &wCompleted, &j.ApplicationsCount)
	if err != nil {
		return nil, err
	}
	j.Client.ID = j.clientID
	if j.workerID != nil && wFirst != nil {
		j.Worker = &personRef{ID: *j.workerID, FirstName: *wFirst, LastName: *wLast, AvatarURL: *wAvatar}
		if wRating != nil {
			j.Worker.Rating, j.Worker.RatingCount, j.Worker.CompletedJobs = *wRating, *wCount, *wCompleted
		}
	}
	return j, nil
}

func (s *Server) queryJobs(ctx context.Context, where string, args ...any) ([]*jobRow, error) {
	rows, err := s.db.Query(ctx, jobSelect+" "+where, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*jobRow
	for rows.Next() {
		j, err := scanJobRow(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, j)
	}
	return out, rows.Err()
}

// present applies the location privacy rules (§17): only the client and the
// assigned worker (and admins) see the exact address and coordinates; other
// workers get the zone and a rounded distance.
func (j *jobRow) present(viewerID string, isAdmin bool, viewerLat, viewerLng *float64) jobView {
	v := j.jobView
	isWorker := j.workerID != nil && *j.workerID == viewerID
	switch {
	case j.clientID == viewerID:
		v.MyRole = "client"
	case isWorker:
		v.MyRole = "worker"
	case isAdmin:
		v.MyRole = "admin"
	default:
		v.MyRole = "candidate"
	}
	if domain.AddressVisible(j.clientID == viewerID, isWorker) || isAdmin {
		addr, lat, lng := j.address, j.latitude, j.longitude
		v.Address, v.Latitude, v.Longitude = &addr, &lat, &lng
	}
	if viewerLat != nil && viewerLng != nil && j.clientID != viewerID {
		d := domain.RoundDistance(domain.DistanceKm(*viewerLat, *viewerLng, j.latitude, j.longitude))
		v.DistanceKm = &d
	}
	if v.Photos == nil {
		v.Photos = []string{}
	}
	return v
}

func (s *Server) userLocation(ctx context.Context, userID string) (lat, lng *float64, err error) {
	err = s.db.QueryRow(ctx, `SELECT latitude, longitude FROM users WHERE id = $1`, userID).Scan(&lat, &lng)
	return
}

func addHistory(ctx context.Context, tx pgx.Tx, jobID string, status domain.JobStatus, actorID, note string) error {
	var actor *string
	if actorID != "" {
		actor = &actorID
	}
	_, err := tx.Exec(ctx, `INSERT INTO job_status_history (job_id, status, actor_id, note) VALUES ($1, $2, $3, $4)`,
		jobID, string(status), actor, note)
	return err
}

func setStatus(ctx context.Context, tx pgx.Tx, jobID string, status domain.JobStatus, actorID, note string) error {
	if _, err := tx.Exec(ctx, `UPDATE jobs SET status = $2, updated_at = now() WHERE id = $1`, jobID, string(status)); err != nil {
		return err
	}
	return addHistory(ctx, tx, jobID, status, actorID, note)
}

// lockedJob is the minimal job state read under SELECT … FOR UPDATE.
type lockedJob struct {
	ID          string
	ClientID    string
	WorkerID    *string
	CategoryID  int
	Title       string
	PricingMode string
	Budget      int
	Status      domain.JobStatus
}

func lockJob(ctx context.Context, tx pgx.Tx, id string) (*lockedJob, error) {
	j := &lockedJob{}
	var status string
	err := tx.QueryRow(ctx, `SELECT id, client_id, worker_id, category_id, title, pricing_mode, budget, status
		FROM jobs WHERE id = $1 FOR UPDATE`, id).Scan(&j.ID, &j.ClientID, &j.WorkerID, &j.CategoryID, &j.Title, &j.PricingMode, &j.Budget, &status)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, notFound("job not found")
	}
	j.Status = domain.JobStatus(status)
	return j, err
}

func (j *lockedJob) isWorker(id string) bool { return j.WorkerID != nil && *j.WorkerID == id }

func formatFCFA(n int) string {
	s := fmt.Sprintf("%d", n)
	var b strings.Builder
	for i, c := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			b.WriteRune(' ')
		}
		b.WriteRune(c)
	}
	return b.String() + " FCFA"
}

// ---- create & matching ----

func (s *Server) createJob(w http.ResponseWriter, r *http.Request) error {
	var req struct {
		CategoryID      int        `json:"category_id"`
		SkillID         *int       `json:"skill_id"`
		Title           string     `json:"title"`
		Description     string     `json:"description"`
		Latitude        float64    `json:"latitude"`
		Longitude       float64    `json:"longitude"`
		Address         string     `json:"address"`
		Zone            string     `json:"zone"`
		PricingMode     string     `json:"pricing_mode"`
		Budget          int        `json:"budget"`
		ScheduledAt     *time.Time `json:"scheduled_at"`
		DurationMinutes int        `json:"duration_minutes"`
		Photos          []string   `json:"photos"`
	}
	if err := decode(r, &req); err != nil {
		return err
	}
	req.Title = strings.TrimSpace(req.Title)
	if req.PricingMode == "" {
		req.PricingMode = "fixed"
	}
	switch {
	case len(req.Title) < 3:
		return badRequest("title is required")
	case req.PricingMode != "fixed" && req.PricingMode != "offers":
		return badRequest("pricing_mode must be fixed or offers")
	case req.PricingMode == "fixed" && req.Budget <= 0:
		return badRequest("a fixed-price job needs a budget")
	case req.Budget < 0:
		return badRequest("budget must be positive")
	case req.Latitude < -90 || req.Latitude > 90 || req.Longitude < -180 || req.Longitude > 180 || (req.Latitude == 0 && req.Longitude == 0):
		return badRequest("a valid location is required")
	case strings.TrimSpace(req.Zone) == "":
		return badRequest("zone is required")
	case len(req.Photos) > 6:
		return badRequest("at most 6 photos")
	}
	if req.DurationMinutes <= 0 {
		req.DurationMinutes = 60
	}
	if req.Photos == nil {
		req.Photos = []string{}
	}
	p := me(r)
	var id string
	err := s.inTx(r.Context(), func(tx pgx.Tx) error {
		var ok bool
		err := tx.QueryRow(r.Context(), `SELECT EXISTS (SELECT 1 FROM categories WHERE id = $1 AND active)
			AND ($2::int IS NULL OR EXISTS (SELECT 1 FROM skills WHERE id = $2 AND category_id = $1 AND active))`,
			req.CategoryID, req.SkillID).Scan(&ok)
		if err != nil {
			return err
		}
		if !ok {
			return badRequest("unknown category or skill")
		}
		err = tx.QueryRow(r.Context(), `INSERT INTO jobs (client_id, category_id, skill_id, title, description, latitude, longitude,
				address, zone, pricing_mode, budget, scheduled_at, duration_minutes, photos)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14) RETURNING id`,
			p.ID, req.CategoryID, req.SkillID, req.Title, req.Description, req.Latitude, req.Longitude,
			req.Address, strings.TrimSpace(req.Zone), req.PricingMode, req.Budget, req.ScheduledAt, req.DurationMinutes, req.Photos).Scan(&id)
		if err != nil {
			return err
		}
		if _, err := tx.Exec(r.Context(), `UPDATE users SET is_client = TRUE WHERE id = $1 AND NOT is_client`, p.ID); err != nil {
			return err
		}
		return addHistory(r.Context(), tx, id, domain.StatusPosted, p.ID, "")
	})
	if err != nil {
		return err
	}
	notified, err := s.dispatch(r.Context(), id)
	if err != nil {
		s.log.Error("dispatch job", "job", id, "err", err)
	}
	job, err := s.jobDetail(r.Context(), id, p)
	if err != nil {
		return err
	}
	return writeJSON(w, http.StatusCreated, map[string]any{"job": job, "notified_workers": notified})
}

type candidate struct {
	workerID string
	distance float64
	score    float64
}

// dispatch finds compatible workers (§12: distance, skill, availability,
// rating, history), records them and notifies the best ones (§13).
func (s *Server) dispatch(ctx context.Context, jobID string) (int, error) {
	var clientID, title, zone, pricing, status string
	var categoryID, budget int
	var skillID *int
	var lat, lng float64
	var scheduled *time.Time
	err := s.db.QueryRow(ctx, `SELECT client_id, category_id, skill_id, title, zone, pricing_mode, budget, latitude, longitude, scheduled_at, status
		FROM jobs WHERE id = $1`, jobID).Scan(&clientID, &categoryID, &skillID, &title, &zone, &pricing, &budget, &lat, &lng, &scheduled, &status)
	if err != nil {
		return 0, err
	}
	if !domain.IsOpen(domain.JobStatus(status)) {
		return 0, nil
	}
	var maxNotified int
	if err := s.db.QueryRow(ctx, `SELECT value::int FROM settings WHERE key = 'matching_max_notified'`).Scan(&maxNotified); err != nil {
		maxNotified = 20
	}

	rows, err := s.db.Query(ctx, `SELECT u.id, u.latitude, u.longitude, wp.radius_km, wp.rating_avg, wp.rating_count, wp.completed_jobs,
			(SELECT count(*) FROM jobs j2 WHERE j2.worker_id = u.id AND j2.category_id = $2 AND j2.status IN ('COMPLETED', 'PAID', 'RATED')),
			EXISTS (SELECT 1 FROM user_skills us WHERE us.user_id = u.id AND us.skill_id = $3)
		FROM users u JOIN worker_profiles wp ON wp.user_id = u.id
		WHERE u.is_worker AND u.status = 'active' AND wp.available AND u.id <> $1 AND u.latitude IS NOT NULL
		  AND EXISTS (SELECT 1 FROM user_skills us JOIN skills sk ON sk.id = us.skill_id WHERE us.user_id = u.id AND sk.category_id = $2)
		  AND NOT EXISTS (SELECT 1 FROM job_dispatches d WHERE d.job_id = $4 AND d.worker_id = u.id AND d.declined)
		  AND NOT EXISTS (SELECT 1 FROM cancellations x WHERE x.job_id = $4 AND x.user_id = u.id AND x.role = 'worker')`,
		clientID, categoryID, skillID, jobID)
	if err != nil {
		return 0, err
	}
	var cands []candidate
	var id string
	var wlat, wlng, radius, rating float64
	var ratingCount, completed, similar int
	var exactSkill bool
	_, err = pgx.ForEachRow(rows, []any{&id, &wlat, &wlng, &radius, &rating, &ratingCount, &completed, &similar, &exactSkill}, func() error {
		d := domain.DistanceKm(lat, lng, wlat, wlng)
		if d > radius {
			return nil
		}
		if exactSkill {
			similar += 3 // the exact skill counts like relevant history
		}
		score := domain.MatchScore(domain.MatchInput{DistanceKm: d, RadiusKm: radius, Rating: rating, RatingCount: ratingCount,
			SimilarJobs: similar, CompletedJobs: completed})
		cands = append(cands, candidate{workerID: id, distance: d, score: score})
		return nil
	})
	if err != nil {
		return 0, err
	}
	sort.Slice(cands, func(i, j int) bool { return cands[i].score > cands[j].score })
	if len(cands) > maxNotified {
		cands = cands[:maxNotified]
	}

	when := "Dès que possible"
	if scheduled != nil {
		when = scheduled.Local().Format("02/01 15:04")
	}
	price := formatFCFA(budget)
	if pricing == "offers" {
		price = "Proposez votre prix"
	}
	var notes []note
	err = s.inTx(ctx, func(tx pgx.Tx) error {
		for _, c := range cands {
			if _, err := tx.Exec(ctx, `INSERT INTO job_dispatches (job_id, worker_id, distance_km, score) VALUES ($1, $2, $3, $4)
				ON CONFLICT (job_id, worker_id) DO UPDATE SET notified_at = now(), score = EXCLUDED.score`,
				jobID, c.workerID, c.distance, c.score); err != nil {
				return err
			}
			notes = append(notes, note{UserID: c.workerID, Kind: "job_nearby", JobID: jobID,
				Title: "🔥 Nouveau job à proximité",
				Body:  fmt.Sprintf("%s · 📍 %s · 💰 %s · 📏 %.1f km · ⏱ %s", title, zone, price, domain.RoundDistance(c.distance), when)})
		}
		if status == string(domain.StatusPosted) {
			return setStatus(ctx, tx, jobID, domain.StatusMatching, "", fmt.Sprintf("%d workers notifiés", len(cands)))
		}
		return nil
	})
	if err != nil {
		return 0, err
	}
	s.send(ctx, notes...)
	return len(cands), nil
}

// ---- reading ----

func (s *Server) jobDetail(ctx context.Context, jobID string, p *principal) (*jobView, error) {
	jobs, err := s.queryJobs(ctx, "WHERE j.id = $1", jobID)
	if err != nil {
		return nil, err
	}
	if len(jobs) == 0 {
		return nil, notFound("job not found")
	}
	j := jobs[0]
	lat, lng, err := s.userLocation(ctx, p.ID)
	if err != nil {
		return nil, err
	}
	v := j.present(p.ID, p.IsAdmin, lat, lng)
	status := domain.JobStatus(v.Status)

	if v.MyRole == "candidate" {
		// Other workers only see open jobs, or jobs they applied to.
		var applied bool
		if err := s.db.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM job_applications WHERE job_id = $1 AND worker_id = $2)`, jobID, p.ID).Scan(&applied); err != nil {
			return nil, err
		}
		if !p.IsWorker || (!domain.IsOpen(status) && !applied) {
			return nil, notFound("job not found")
		}
		v.Worker = nil
	}

	if app, err := s.applicationOf(ctx, jobID, p.ID); err != nil {
		return nil, err
	} else {
		v.MyApplication = app
	}

	if v.MyRole == "candidate" {
		return &v, nil
	}

	// Participants and admins get the full picture.
	a := &assignmentView{}
	err = s.db.QueryRow(ctx, `SELECT assigned_at, on_the_way_at, arrived_at, started_at, finished_at, completed_at,
			worker_latitude, worker_longitude, location_updated_at
		FROM job_assignments WHERE job_id = $1 AND cancelled_at IS NULL`, jobID).
		Scan(&a.AssignedAt, &a.OnTheWayAt, &a.ArrivedAt, &a.StartedAt, &a.FinishedAt, &a.CompletedAt,
			&a.WorkerLatitude, &a.WorkerLongitude, &a.LocationAt)
	if err == nil {
		if a.WorkerLatitude != nil {
			d := domain.RoundDistance(domain.DistanceKm(j.latitude, j.longitude, *a.WorkerLatitude, *a.WorkerLongitude))
			a.WorkerDistanceKm = &d
		}
		v.Assignment = a
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return nil, err
	}

	pay := &paymentView{}
	err = s.db.QueryRow(ctx, `SELECT id, amount, platform_fee, worker_amount, refunded_amount, commission_rate, provider, status, created_at
		FROM payments WHERE job_id = $1 ORDER BY created_at DESC LIMIT 1`, jobID).
		Scan(&pay.ID, &pay.Amount, &pay.PlatformFee, &pay.WorkerAmount, &pay.RefundedAmount, &pay.CommissionRate, &pay.Provider, &pay.Status, &pay.CreatedAt)
	if err == nil {
		v.Payment = pay
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return nil, err
	}

	err = s.db.QueryRow(ctx, `SELECT
			EXISTS (SELECT 1 FROM reviews WHERE job_id = $1 AND reviewer_id = $2),
			EXISTS (SELECT 1 FROM reviews WHERE job_id = $1 AND reviewer_id <> $2),
			EXISTS (SELECT 1 FROM reports WHERE job_id = $1 AND status = 'open'),
			EXISTS (SELECT 1 FROM conversations WHERE job_id = $1 AND (client_id = $2 OR worker_id = $2 OR $3))`,
		jobID, p.ID, p.IsAdmin).Scan(&v.ReviewedByMe, &v.ReviewedByOther, &v.OpenReport, &v.HasConversation)
	if err != nil {
		return nil, err
	}

	rows, err := s.db.Query(ctx, `SELECT status, note, created_at FROM job_status_history WHERE job_id = $1 ORDER BY id`, jobID)
	if err != nil {
		return nil, err
	}
	v.History, err = pgx.CollectRows(rows, func(row pgx.CollectableRow) (historyItem, error) {
		var h historyItem
		err := row.Scan(&h.Status, &h.Note, &h.CreatedAt)
		return h, err
	})
	if err != nil {
		return nil, err
	}
	return &v, nil
}

func (s *Server) getJob(w http.ResponseWriter, r *http.Request) error {
	v, err := s.jobDetail(r.Context(), r.PathValue("id"), me(r))
	if err != nil {
		return err
	}
	return writeJSON(w, http.StatusOK, v)
}

func (s *Server) presentAll(r *http.Request, jobs []*jobRow) ([]jobView, error) {
	p := me(r)
	lat, lng, err := s.userLocation(r.Context(), p.ID)
	if err != nil {
		return nil, err
	}
	out := make([]jobView, 0, len(jobs))
	for _, j := range jobs {
		out = append(out, j.present(p.ID, p.IsAdmin, lat, lng))
	}
	return out, nil
}

// listMyJobs is the client's "Mes jobs" (§27): active | done | cancelled.
func (s *Server) listMyJobs(w http.ResponseWriter, r *http.Request) error {
	where := "WHERE j.client_id = $1 AND j.status NOT IN ('COMPLETED', 'PAID', 'RATED', 'CANCELLED')"
	switch r.URL.Query().Get("scope") {
	case "done":
		where = "WHERE j.client_id = $1 AND j.status IN ('COMPLETED', 'PAID', 'RATED')"
	case "cancelled":
		where = "WHERE j.client_id = $1 AND j.status = 'CANCELLED'"
	case "all":
		where = "WHERE j.client_id = $1"
	}
	jobs, err := s.queryJobs(r.Context(), where+" ORDER BY j.created_at DESC LIMIT 100", me(r).ID)
	if err != nil {
		return err
	}
	out, err := s.presentAll(r, jobs)
	if err != nil {
		return err
	}
	return writeJSON(w, http.StatusOK, out)
}

// listMissions is the worker's "Mes missions" (§27): active | done | applied.
func (s *Server) listMissions(w http.ResponseWriter, r *http.Request) error {
	where := "WHERE j.worker_id = $1 AND j.status IN ('ASSIGNED', 'WORKER_ON_THE_WAY', 'ARRIVED', 'STARTED', 'AWAITING_CONFIRMATION', 'DISPUTED')"
	switch r.URL.Query().Get("scope") {
	case "done":
		where = "WHERE j.worker_id = $1 AND j.status IN ('COMPLETED', 'PAID', 'RATED')"
	case "applied":
		where = `WHERE j.status IN ('POSTED', 'MATCHING') AND EXISTS (SELECT 1 FROM job_applications a
			WHERE a.job_id = j.id AND a.worker_id = $1 AND a.status = 'pending')`
	}
	jobs, err := s.queryJobs(r.Context(), where+" ORDER BY j.updated_at DESC LIMIT 100", me(r).ID)
	if err != nil {
		return err
	}
	out, err := s.presentAll(r, jobs)
	if err != nil {
		return err
	}
	return writeJSON(w, http.StatusOK, out)
}

// listAvailableJobs is the worker feed: open jobs in their categories and
// radius, nearest first.
func (s *Server) listAvailableJobs(w http.ResponseWriter, r *http.Request) error {
	p := me(r)
	if !p.IsWorker {
		return forbidden("create a worker profile first")
	}
	var lat, lng *float64
	var radius float64
	err := s.db.QueryRow(r.Context(), `SELECT u.latitude, u.longitude, wp.radius_km FROM users u
		JOIN worker_profiles wp ON wp.user_id = u.id WHERE u.id = $1`, p.ID).Scan(&lat, &lng, &radius)
	if err != nil {
		return err
	}
	if lat == nil {
		return badRequest("set your location to see nearby jobs")
	}
	jobs, err := s.queryJobs(r.Context(), `WHERE j.status IN ('POSTED', 'MATCHING') AND j.client_id <> $1
		AND j.category_id IN (SELECT sk.category_id FROM user_skills us JOIN skills sk ON sk.id = us.skill_id WHERE us.user_id = $1)
		AND NOT EXISTS (SELECT 1 FROM job_dispatches d WHERE d.job_id = j.id AND d.worker_id = $1 AND d.declined)
		AND NOT EXISTS (SELECT 1 FROM cancellations x WHERE x.job_id = j.id AND x.user_id = $1 AND x.role = 'worker')
		ORDER BY j.created_at DESC LIMIT 300`, p.ID)
	if err != nil {
		return err
	}
	out := []jobView{}
	for _, j := range jobs {
		if domain.DistanceKm(*lat, *lng, j.latitude, j.longitude) > radius {
			continue
		}
		v := j.present(p.ID, false, lat, lng)
		v.Worker = nil
		out = append(out, v)
	}
	sort.SliceStable(out, func(i, k int) bool { return *out[i].DistanceKm < *out[k].DistanceKm })
	return writeJSON(w, http.StatusOK, out)
}

// ---- assignment ----

// workerEligible checks the worker can take a job in this category.
func workerEligible(ctx context.Context, tx pgx.Tx, workerID string, categoryID int, jobID string) error {
	var isWorker bool
	var status string
	var hasSkill, cancelledBefore bool
	err := tx.QueryRow(ctx, `SELECT u.is_worker, u.status,
			EXISTS (SELECT 1 FROM user_skills us JOIN skills sk ON sk.id = us.skill_id WHERE us.user_id = u.id AND sk.category_id = $2),
			EXISTS (SELECT 1 FROM cancellations x WHERE x.job_id = $3 AND x.user_id = u.id AND x.role = 'worker')
		FROM users u WHERE u.id = $1`, workerID, categoryID, jobID).Scan(&isWorker, &status, &hasSkill, &cancelledBefore)
	if err != nil {
		return err
	}
	switch {
	case !isWorker:
		return forbidden("create a worker profile first")
	case status != "active":
		return forbidden("worker account " + status)
	case !hasSkill:
		return forbidden("you need a skill in this category to take this job")
	case cancelledBefore:
		return forbidden("you already cancelled this job")
	}
	return nil
}

func commissionRate(ctx context.Context, tx pgx.Tx, categoryID int) (float64, error) {
	var rate float64
	err := tx.QueryRow(ctx, `SELECT COALESCE(c.commission_rate,
			(SELECT value::numeric FROM settings WHERE key = 'default_commission_rate'))
		FROM categories c WHERE c.id = $1`, categoryID).Scan(&rate)
	return rate, err
}

func systemMessage(ctx context.Context, tx pgx.Tx, jobID, workerID, body string) error {
	_, err := tx.Exec(ctx, `INSERT INTO messages (conversation_id, kind, body)
		SELECT id, 'system', $3 FROM conversations WHERE job_id = $1 AND worker_id = $2`, jobID, workerID, body)
	return err
}

// assign gives the job to a worker (§15): status ASSIGNED, a conversation
// (§18) and the client's payment held in escrow (§22).
func (s *Server) assign(ctx context.Context, tx pgx.Tx, j *lockedJob, workerID string, price int) ([]note, error) {
	var workerName string
	if err := tx.QueryRow(ctx, `SELECT first_name FROM users WHERE id = $1`, workerID).Scan(&workerName); err != nil {
		return nil, err
	}
	if _, err := tx.Exec(ctx, `UPDATE jobs SET worker_id = $2, agreed_price = $3, updated_at = now() WHERE id = $1`, j.ID, workerID, price); err != nil {
		return nil, err
	}
	if err := setStatus(ctx, tx, j.ID, domain.StatusAssigned, workerID, workerName+" a accepté le job"); err != nil {
		return nil, err
	}
	if _, err := tx.Exec(ctx, `INSERT INTO job_assignments (job_id, worker_id, price) VALUES ($1, $2, $3)`, j.ID, workerID, price); err != nil {
		return nil, err
	}
	if _, err := tx.Exec(ctx, `INSERT INTO conversations (job_id, client_id, worker_id) VALUES ($1, $2, $3)
		ON CONFLICT (job_id, worker_id) DO NOTHING`, j.ID, j.ClientID, workerID); err != nil {
		return nil, err
	}
	if err := systemMessage(ctx, tx, j.ID, workerID, fmt.Sprintf("✅ %s a accepté le job « %s ». Prix convenu : %s.", workerName, j.Title, formatFCFA(price))); err != nil {
		return nil, err
	}

	rate, err := commissionRate(ctx, tx, j.CategoryID)
	if err != nil {
		return nil, err
	}
	split, err := domain.SplitPayment(price, rate)
	if err != nil {
		return nil, err
	}
	ref, err := s.payments.Hold(ctx, payments.HoldRequest{JobID: j.ID, PayerID: j.ClientID, Amount: price, Currency: "XOF"})
	if err != nil {
		return nil, errStatus(http.StatusPaymentRequired, "payment failed: "+err.Error())
	}
	if _, err := tx.Exec(ctx, `INSERT INTO payments (job_id, payer_id, worker_id, amount, platform_fee, worker_amount, commission_rate, provider, provider_ref, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, 'held')`,
		j.ID, j.ClientID, workerID, split.Amount, split.PlatformFee, split.WorkerAmount, rate, s.payments.Name(), ref); err != nil {
		return nil, err
	}

	if _, err := tx.Exec(ctx, `UPDATE worker_profiles SET accepted_jobs = accepted_jobs + 1 WHERE user_id = $1`, workerID); err != nil {
		return nil, err
	}
	if _, err := tx.Exec(ctx, `UPDATE job_applications SET status = CASE WHEN worker_id = $2 THEN 'accepted' ELSE 'rejected' END
		WHERE job_id = $1 AND status = 'pending'`, j.ID, workerID); err != nil {
		return nil, err
	}
	return []note{
		{UserID: j.ClientID, Kind: "worker_found", JobID: j.ID, Title: "✅ " + workerName + " a accepté votre job", Body: j.Title + " · " + formatFCFA(price) + " retenus en séquestre"},
		{UserID: workerID, Kind: "job_accepted", JobID: j.ID, Title: "✅ Job accepté", Body: j.Title},
	}, nil
}

// acceptJob is the one-tap ACCEPT JOB for fixed-price jobs. The row lock
// guarantees only one worker wins (§15).
func (s *Server) acceptJob(w http.ResponseWriter, r *http.Request) error {
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
			return forbidden("you cannot accept your own job")
		case !domain.IsOpen(j.Status):
			return conflict("job is no longer available")
		case j.PricingMode != "fixed":
			return badRequest("this job takes offers: send a price instead")
		}
		if err := workerEligible(r.Context(), tx, p.ID, j.CategoryID, j.ID); err != nil {
			return err
		}
		notes, err = s.assign(r.Context(), tx, j, p.ID, j.Budget)
		return err
	})
	if err != nil {
		return err
	}
	s.send(r.Context(), notes...)
	s.jobChanged(r.PathValue("id"), string(domain.StatusAssigned), clientID)
	return s.getJob(w, r)
}

func (s *Server) declineJob(w http.ResponseWriter, r *http.Request) error {
	p := me(r)
	jobID := r.PathValue("id")
	err := s.inTx(r.Context(), func(tx pgx.Tx) error {
		if _, err := tx.Exec(r.Context(), `INSERT INTO job_dispatches (job_id, worker_id, distance_km, score, declined)
			VALUES ($1, $2, 0, 0, TRUE) ON CONFLICT (job_id, worker_id) DO UPDATE SET declined = TRUE`, jobID, p.ID); err != nil {
			return err
		}
		_, err := tx.Exec(r.Context(), `UPDATE job_applications SET status = 'withdrawn' WHERE job_id = $1 AND worker_id = $2 AND status = 'pending'`, jobID, p.ID)
		return err
	})
	if err != nil {
		return err
	}
	w.WriteHeader(http.StatusNoContent)
	return nil
}

// ---- progress ----

var statusColumn = map[domain.JobStatus]string{
	domain.StatusWorkerOnTheWay:       "on_the_way_at",
	domain.StatusArrived:              "arrived_at",
	domain.StatusStarted:              "started_at",
	domain.StatusAwaitingConfirmation: "finished_at",
}

// advanceJob lets the assigned worker report progress (§20-21).
func (s *Server) advanceJob(w http.ResponseWriter, r *http.Request) error {
	var req struct {
		Status string `json:"status"`
	}
	if err := decode(r, &req); err != nil {
		return err
	}
	to := domain.JobStatus(req.Status)
	p := me(r)
	var notes []note
	var clientID string
	err := s.inTx(r.Context(), func(tx pgx.Tx) error {
		j, err := lockJob(r.Context(), tx, r.PathValue("id"))
		if err != nil {
			return err
		}
		clientID = j.ClientID
		if !j.isWorker(p.ID) {
			return forbidden("only the assigned worker can update progress")
		}
		if !domain.CanAdvance(j.Status, to, domain.ActorWorker) {
			return conflict(fmt.Sprintf("cannot go from %s to %s", j.Status, to))
		}
		if err := setStatus(r.Context(), tx, j.ID, to, p.ID, ""); err != nil {
			return err
		}
		col := statusColumn[to]
		if _, err := tx.Exec(r.Context(), `UPDATE job_assignments SET `+col+` = now() WHERE job_id = $1 AND cancelled_at IS NULL`, j.ID); err != nil {
			return err
		}
		var title, body string
		switch to {
		case domain.StatusWorkerOnTheWay:
			title, body = "🚗 "+p.Name+" est en route", j.Title
		case domain.StatusArrived:
			title, body = "📍 "+p.Name+" est arrivé", j.Title
		case domain.StatusStarted:
			title, body = "🛠 Le job a commencé", j.Title
		case domain.StatusAwaitingConfirmation:
			title, body = "🏁 Le worker indique que le job est terminé", "Confirmez la fin pour libérer le paiement."
		}
		if err := systemMessage(r.Context(), tx, j.ID, p.ID, title); err != nil {
			return err
		}
		notes = append(notes, note{UserID: j.ClientID, Kind: "job_" + strings.ToLower(string(to)), JobID: j.ID, Title: title, Body: body})
		return nil
	})
	if err != nil {
		return err
	}
	s.send(r.Context(), notes...)
	s.jobChanged(r.PathValue("id"), string(to), clientID)
	return s.getJob(w, r)
}

// settle releases held funds to the worker, minus an optional refund to
// the client, and marks the job PAID (§22).
func (s *Server) settle(ctx context.Context, tx pgx.Tx, jobID, actorID string, refund int) (note, error) {
	var payID, ref, workerID, title string
	var amount int
	var rate float64
	err := tx.QueryRow(ctx, `SELECT p.id, p.provider_ref, p.worker_id, p.amount, p.commission_rate, j.title
		FROM payments p JOIN jobs j ON j.id = p.job_id
		WHERE p.job_id = $1 AND p.status = 'held' FOR UPDATE OF p`, jobID).Scan(&payID, &ref, &workerID, &amount, &rate, &title)
	if errors.Is(err, pgx.ErrNoRows) {
		return note{}, conflict("no funds held for this job")
	}
	if err != nil {
		return note{}, err
	}
	split, err := domain.PartialRefund(amount, refund, rate)
	if err != nil {
		return note{}, badRequest(err.Error())
	}
	if refund > 0 {
		if err := s.payments.Refund(ctx, ref, refund); err != nil {
			return note{}, err
		}
	}
	if err := s.payments.Release(ctx, ref, split.Amount); err != nil {
		return note{}, err
	}
	status := "released"
	if refund > 0 {
		status = "partially_refunded"
	}
	if _, err := tx.Exec(ctx, `UPDATE payments SET status = $2, platform_fee = $3, worker_amount = $4, refunded_amount = $5, updated_at = now()
		WHERE id = $1`, payID, status, split.PlatformFee, split.WorkerAmount, refund); err != nil {
		return note{}, err
	}
	if _, err := tx.Exec(ctx, `INSERT INTO wallet_transactions (user_id, amount, kind, job_id, description) VALUES ($1, $2, 'earning', $3, $4)`,
		workerID, split.WorkerAmount, jobID, title); err != nil {
		return note{}, err
	}
	if _, err := tx.Exec(ctx, `UPDATE worker_profiles SET completed_jobs = completed_jobs + 1 WHERE user_id = $1`, workerID); err != nil {
		return note{}, err
	}
	if _, err := tx.Exec(ctx, `UPDATE job_assignments SET completed_at = COALESCE(completed_at, now()) WHERE job_id = $1 AND cancelled_at IS NULL`, jobID); err != nil {
		return note{}, err
	}
	if err := setStatus(ctx, tx, jobID, domain.StatusPaid, actorID, fmt.Sprintf("Worker : %s · Plateforme : %s", formatFCFA(split.WorkerAmount), formatFCFA(split.PlatformFee))); err != nil {
		return note{}, err
	}
	return note{UserID: workerID, Kind: "payment", JobID: jobID, Title: "💰 Paiement reçu : " + formatFCFA(split.WorkerAmount), Body: title}, nil
}

// refundHeld returns held funds to the client (cancellation or dispute).
func (s *Server) refundHeld(ctx context.Context, tx pgx.Tx, jobID string) error {
	var payID, ref string
	var amount int
	err := tx.QueryRow(ctx, `SELECT id, provider_ref, amount FROM payments WHERE job_id = $1 AND status = 'held' FOR UPDATE`, jobID).Scan(&payID, &ref, &amount)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	if err := s.payments.Refund(ctx, ref, amount); err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `UPDATE payments SET status = 'refunded', refunded_amount = amount, updated_at = now() WHERE id = $1`, payID)
	return err
}

// confirmJob is the client's CONFIRM COMPLETION: the job is COMPLETED and
// the escrow is released to the worker's wallet.
func (s *Server) confirmJob(w http.ResponseWriter, r *http.Request) error {
	p := me(r)
	var notes []note
	var workerID string
	err := s.inTx(r.Context(), func(tx pgx.Tx) error {
		j, err := lockJob(r.Context(), tx, r.PathValue("id"))
		if err != nil {
			return err
		}
		if j.ClientID != p.ID {
			return forbidden("only the client can confirm completion")
		}
		if j.Status != domain.StatusAwaitingConfirmation {
			return conflict("the worker has not marked the job as finished")
		}
		workerID = *j.WorkerID
		if err := setStatus(r.Context(), tx, j.ID, domain.StatusCompleted, p.ID, "Fin confirmée par le client"); err != nil {
			return err
		}
		n, err := s.settle(r.Context(), tx, j.ID, p.ID, 0)
		if err != nil {
			return err
		}
		if err := systemMessage(r.Context(), tx, j.ID, workerID, "✅ Job terminé et paiement libéré. Pensez à laisser une évaluation."); err != nil {
			return err
		}
		notes = append(notes, note{UserID: workerID, Kind: "job_completed", JobID: j.ID, Title: "✅ Le client a confirmé la fin du job", Body: j.Title}, n)
		return nil
	})
	if err != nil {
		return err
	}
	s.send(r.Context(), notes...)
	s.jobChanged(r.PathValue("id"), string(domain.StatusPaid), workerID)
	return s.getJob(w, r)
}

// cancelJob records who cancelled, why and when (§24). A client cancel ends
// the job and refunds the escrow; a worker cancel sends it back to matching.
func (s *Server) cancelJob(w http.ResponseWriter, r *http.Request) error {
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
	var redispatch bool
	var notify []string
	var newStatus domain.JobStatus
	err := s.inTx(r.Context(), func(tx pgx.Tx) error {
		j, err := lockJob(r.Context(), tx, r.PathValue("id"))
		if err != nil {
			return err
		}
		var actor domain.Actor
		switch {
		case j.ClientID == p.ID:
			actor = domain.ActorClient
		case j.isWorker(p.ID):
			actor = domain.ActorWorker
		default:
			return forbidden("not your job")
		}
		if !domain.CanCancel(j.Status, actor) {
			return conflict("this job can no longer be cancelled; report a problem instead")
		}
		impact := domain.CancellationHurtsReputation(j.Status, actor)
		if _, err := tx.Exec(r.Context(), `INSERT INTO cancellations (job_id, user_id, role, reason, status_at_cancel, reputation_impact)
			VALUES ($1, $2, $3, $4, $5, $6)`, j.ID, p.ID, string(actor), req.Reason, string(j.Status), impact); err != nil {
			return err
		}
		if j.WorkerID != nil {
			if err := s.refundHeld(r.Context(), tx, j.ID); err != nil {
				return err
			}
			if _, err := tx.Exec(r.Context(), `UPDATE job_assignments SET cancelled_at = now() WHERE job_id = $1 AND cancelled_at IS NULL`, j.ID); err != nil {
				return err
			}
		}

		if actor == domain.ActorClient {
			newStatus = domain.StatusCancelled
			if err := setStatus(r.Context(), tx, j.ID, newStatus, p.ID, "Annulé par le client : "+req.Reason); err != nil {
				return err
			}
			if _, err := tx.Exec(r.Context(), `UPDATE job_applications SET status = 'rejected' WHERE job_id = $1 AND status = 'pending'`, j.ID); err != nil {
				return err
			}
			if j.WorkerID != nil {
				if err := systemMessage(r.Context(), tx, j.ID, *j.WorkerID, "❌ Le client a annulé le job."); err != nil {
					return err
				}
				notes = append(notes, note{UserID: *j.WorkerID, Kind: "job_cancelled", JobID: j.ID, Title: "❌ Job annulé par le client", Body: j.Title + " · " + req.Reason})
				notify = append(notify, *j.WorkerID)
			}
			return nil
		}

		// Worker drops the job: it goes back to matching without them.
		newStatus = domain.StatusMatching
		redispatch = true
		notify = append(notify, j.ClientID)
		if _, err := tx.Exec(r.Context(), `UPDATE jobs SET worker_id = NULL, agreed_price = NULL WHERE id = $1`, j.ID); err != nil {
			return err
		}
		if err := setStatus(r.Context(), tx, j.ID, newStatus, p.ID, "Le worker a annulé : "+req.Reason); err != nil {
			return err
		}
		if _, err := tx.Exec(r.Context(), `UPDATE worker_profiles SET cancelled_jobs = cancelled_jobs + 1 WHERE user_id = $1`, p.ID); err != nil {
			return err
		}
		if _, err := tx.Exec(r.Context(), `UPDATE job_applications SET status = 'withdrawn' WHERE job_id = $1 AND worker_id = $2`, j.ID, p.ID); err != nil {
			return err
		}
		if err := systemMessage(r.Context(), tx, j.ID, p.ID, "❌ Le worker a annulé. Nous recherchons un autre worker."); err != nil {
			return err
		}
		notes = append(notes, note{UserID: j.ClientID, Kind: "job_cancelled", JobID: j.ID,
			Title: "⚠️ " + p.Name + " a annulé", Body: "Votre paiement est remboursé et nous recherchons un autre worker."})
		return nil
	})
	if err != nil {
		return err
	}
	s.send(r.Context(), notes...)
	s.jobChanged(r.PathValue("id"), string(newStatus), notify...)
	if redispatch {
		if _, err := s.dispatch(r.Context(), r.PathValue("id")); err != nil {
			s.log.Error("re-dispatch", "job", r.PathValue("id"), "err", err)
		}
	}
	return s.getJob(w, r)
}

// updateWorkerLocation shares the assigned worker's live position with the
// client while they travel (§17).
func (s *Server) updateWorkerLocation(w http.ResponseWriter, r *http.Request) error {
	var req struct {
		Latitude  float64 `json:"latitude"`
		Longitude float64 `json:"longitude"`
	}
	if err := decode(r, &req); err != nil {
		return err
	}
	p := me(r)
	var clientID, status string
	err := s.db.QueryRow(r.Context(), `SELECT client_id, status FROM jobs WHERE id = $1 AND worker_id = $2`, r.PathValue("id"), p.ID).Scan(&clientID, &status)
	if errors.Is(err, pgx.ErrNoRows) {
		return forbidden("only the assigned worker can share a location")
	}
	if err != nil {
		return err
	}
	switch domain.JobStatus(status) {
	case domain.StatusAssigned, domain.StatusWorkerOnTheWay, domain.StatusArrived:
	default:
		return conflict("location sharing is only active before the job starts")
	}
	_, err = s.db.Exec(r.Context(), `UPDATE job_assignments SET worker_latitude = $2, worker_longitude = $3, location_updated_at = now()
		WHERE job_id = $1 AND cancelled_at IS NULL`, r.PathValue("id"), req.Latitude, req.Longitude)
	if err != nil {
		return err
	}
	s.jobChanged(r.PathValue("id"), status, clientID)
	w.WriteHeader(http.StatusNoContent)
	return nil
}
