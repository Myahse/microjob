// Package api exposes the MicroJob REST + WebSocket API.
package api

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/myahse/microjob/backend/internal/auth"
	"github.com/myahse/microjob/backend/internal/payments"
	"github.com/myahse/microjob/backend/internal/realtime"
)

type Server struct {
	db        *pgxpool.Pool
	tokens    *auth.Tokens
	hub       *realtime.Hub
	payments  payments.Provider
	uploadDir string
	origins   []string
	log       *slog.Logger
}

type Options struct {
	DB             *pgxpool.Pool
	Tokens         *auth.Tokens
	Payments       payments.Provider
	UploadDir      string
	AllowedOrigins []string
	Logger         *slog.Logger
}

func NewServer(o Options) *Server {
	if o.Logger == nil {
		o.Logger = slog.Default()
	}
	return &Server{
		db:        o.DB,
		tokens:    o.Tokens,
		hub:       realtime.NewHub(),
		payments:  o.Payments,
		uploadDir: o.UploadDir,
		origins:   o.AllowedOrigins,
		log:       o.Logger,
	}
}

// Handler builds the HTTP router.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	pub := func(pattern string, h handlerFunc) { mux.Handle(pattern, s.wrap(h)) }
	authed := func(pattern string, h handlerFunc) { mux.Handle(pattern, s.wrap(s.requireUser(h))) }
	admin := func(pattern string, h handlerFunc) { mux.Handle(pattern, s.wrap(s.requireUser(s.requireAdmin(h)))) }

	pub("GET /api/health", func(w http.ResponseWriter, r *http.Request) error {
		return writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	// Auth & profile
	pub("POST /api/auth/register", s.register)
	pub("POST /api/auth/login", s.login)
	authed("GET /api/me", s.getMe)
	authed("PATCH /api/me", s.updateMe)
	authed("PUT /api/me/worker", s.updateWorkerProfile)
	authed("PUT /api/me/availability", s.updateAvailability)
	authed("GET /api/users/{id}", s.getUserProfile)

	// Catalog & discovery
	pub("GET /api/categories", s.listCategories)
	authed("GET /api/workers/search", s.searchWorkers)
	authed("GET /api/workers/nearby-count", s.nearbyWorkerCount)

	// Jobs
	authed("POST /api/jobs", s.createJob)
	authed("GET /api/jobs/mine", s.listMyJobs)
	authed("GET /api/jobs/missions", s.listMissions)
	authed("GET /api/jobs/available", s.listAvailableJobs)
	authed("GET /api/jobs/{id}", s.getJob)
	authed("POST /api/jobs/{id}/accept", s.acceptJob)
	authed("POST /api/jobs/{id}/decline", s.declineJob)
	authed("GET /api/jobs/{id}/applications", s.listApplications)
	authed("POST /api/jobs/{id}/applications", s.applyToJob)
	authed("POST /api/jobs/{id}/applications/{appId}/choose", s.chooseApplication)
	authed("POST /api/jobs/{id}/status", s.advanceJob)
	authed("POST /api/jobs/{id}/confirm", s.confirmJob)
	authed("POST /api/jobs/{id}/cancel", s.cancelJob)
	authed("PUT /api/jobs/{id}/location", s.updateWorkerLocation)
	authed("GET /api/jobs/{id}/messages", s.listMessages)
	authed("POST /api/jobs/{id}/messages", s.sendMessage)
	authed("POST /api/jobs/{id}/reviews", s.createReview)
	authed("POST /api/jobs/{id}/reports", s.createReport)
	authed("GET /api/conversations", s.listConversations)

	// Wallet, notifications, uploads, realtime
	authed("GET /api/wallet", s.getWallet)
	authed("POST /api/wallet/withdrawals", s.requestWithdrawal)
	authed("GET /api/notifications", s.listNotifications)
	authed("POST /api/notifications/read", s.markNotificationsRead)
	authed("POST /api/uploads", s.upload)
	pub("GET /api/ws", s.websocket)
	mux.Handle("GET /uploads/", http.StripPrefix("/uploads/", http.FileServer(http.Dir(s.uploadDir))))

	// Admin
	admin("GET /api/admin/stats", s.adminStats)
	admin("GET /api/admin/users", s.adminListUsers)
	admin("PATCH /api/admin/users/{id}", s.adminUpdateUser)
	admin("POST /api/admin/users/{id}/skills/{skillId}/verify", s.adminVerifySkill)
	admin("GET /api/admin/jobs", s.adminListJobs)
	admin("POST /api/admin/jobs/{id}/cancel", s.adminCancelJob)
	admin("GET /api/admin/categories", s.adminListCategories)
	admin("POST /api/admin/categories", s.adminCreateCategory)
	admin("PATCH /api/admin/categories/{id}", s.adminUpdateCategory)
	admin("POST /api/admin/categories/{id}/skills", s.adminCreateSkill)
	admin("PATCH /api/admin/skills/{id}", s.adminUpdateSkill)
	admin("GET /api/admin/reports", s.adminListReports)
	admin("POST /api/admin/reports/{id}/resolve", s.adminResolveReport)
	admin("GET /api/admin/payments", s.adminListPayments)
	admin("GET /api/admin/withdrawals", s.adminListWithdrawals)
	admin("POST /api/admin/withdrawals/{id}", s.adminProcessWithdrawal)
	admin("GET /api/admin/settings", s.adminGetSettings)
	admin("PUT /api/admin/settings", s.adminUpdateSettings)

	return s.cors(s.logRequests(mux))
}

// ---- handler plumbing ----

type handlerFunc func(http.ResponseWriter, *http.Request) error

type apiError struct {
	Status  int
	Message string
}

func (e *apiError) Error() string { return e.Message }

func errStatus(status int, msg string) error { return &apiError{Status: status, Message: msg} }
func badRequest(msg string) error            { return errStatus(http.StatusBadRequest, msg) }
func forbidden(msg string) error             { return errStatus(http.StatusForbidden, msg) }
func notFound(msg string) error              { return errStatus(http.StatusNotFound, msg) }
func conflict(msg string) error              { return errStatus(http.StatusConflict, msg) }

func (s *Server) wrap(h handlerFunc) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		err := h(w, r)
		if err == nil {
			return
		}
		var ae *apiError
		var pgErr *pgconn.PgError
		switch {
		case errors.As(err, &ae):
			writeJSON(w, ae.Status, map[string]string{"error": ae.Message})
		case errors.Is(err, pgx.ErrNoRows):
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
		case errors.As(err, &pgErr) && pgErr.Code == "23505":
			writeJSON(w, http.StatusConflict, map[string]string{"error": "already exists"})
		case errors.As(err, &pgErr) && pgErr.Code == "22P02":
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid identifier"})
		default:
			s.log.Error("request failed", "method", r.Method, "path", r.URL.Path, "err", err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		}
	})
}

func writeJSON(w http.ResponseWriter, status int, v any) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	return json.NewEncoder(w).Encode(v)
}

func decode(r *http.Request, v any) error {
	r.Body = http.MaxBytesReader(nil, r.Body, 1<<20)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		return badRequest("invalid JSON body: " + err.Error())
	}
	return nil
}

func (s *Server) cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin != "" && (slices.Contains(s.origins, "*") || slices.Contains(s.origins, origin)) {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Vary", "Origin")
			w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		}
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) logRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		if !strings.HasPrefix(r.URL.Path, "/uploads/") {
			s.log.Debug("http", "method", r.Method, "path", r.URL.Path, "dur", time.Since(start))
		}
	})
}

// ---- authentication ----

type principal struct {
	ID       string
	Name     string
	IsClient bool
	IsWorker bool
	IsAdmin  bool
}

type ctxKey struct{}

func me(r *http.Request) *principal {
	return r.Context().Value(ctxKey{}).(*principal)
}

func (s *Server) authenticate(ctx context.Context, token string) (*principal, error) {
	userID, err := s.tokens.Parse(token)
	if err != nil {
		return nil, errStatus(http.StatusUnauthorized, "invalid or expired token")
	}
	p := &principal{ID: userID}
	var status string
	err = s.db.QueryRow(ctx, `SELECT first_name, is_client, is_worker, is_admin, status FROM users WHERE id = $1`, userID).
		Scan(&p.Name, &p.IsClient, &p.IsWorker, &p.IsAdmin, &status)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, errStatus(http.StatusUnauthorized, "account not found")
	}
	if err != nil {
		return nil, err
	}
	if status != "active" {
		return nil, forbidden("account " + status)
	}
	return p, nil
}

func (s *Server) requireUser(h handlerFunc) handlerFunc {
	return func(w http.ResponseWriter, r *http.Request) error {
		token, ok := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
		if !ok || token == "" {
			return errStatus(http.StatusUnauthorized, "missing bearer token")
		}
		p, err := s.authenticate(r.Context(), token)
		if err != nil {
			return err
		}
		return h(w, r.WithContext(context.WithValue(r.Context(), ctxKey{}, p)))
	}
}

func (s *Server) requireAdmin(h handlerFunc) handlerFunc {
	return func(w http.ResponseWriter, r *http.Request) error {
		if !me(r).IsAdmin {
			return forbidden("admin only")
		}
		return h(w, r)
	}
}

// inTx runs fn in a transaction.
func (s *Server) inTx(ctx context.Context, fn func(pgx.Tx) error) error {
	return pgx.BeginFunc(ctx, s.db, fn)
}
