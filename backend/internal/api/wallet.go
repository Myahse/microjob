package api

import (
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

type walletTx struct {
	ID          string    `json:"id"`
	Amount      int       `json:"amount"`
	Kind        string    `json:"kind"`
	JobID       *string   `json:"job_id"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
}

type withdrawalView struct {
	ID          string     `json:"id"`
	UserID      string     `json:"user_id,omitempty"`
	UserName    string     `json:"user_name,omitempty"`
	Amount      int        `json:"amount"`
	Method      string     `json:"method"`
	Destination string     `json:"destination"`
	Status      string     `json:"status"`
	CreatedAt   time.Time  `json:"created_at"`
	ProcessedAt *time.Time `json:"processed_at"`
}

// getWallet is the worker wallet (§28): balance, earnings and history.
func (s *Server) getWallet(w http.ResponseWriter, r *http.Request) error {
	id := me(r).ID
	var balance, earned, earnedMonth, pending int
	err := s.db.QueryRow(r.Context(), `SELECT
			COALESCE((SELECT sum(amount) FROM wallet_transactions WHERE user_id = $1), 0),
			COALESCE((SELECT sum(amount) FROM wallet_transactions WHERE user_id = $1 AND kind = 'earning'), 0),
			COALESCE((SELECT sum(amount) FROM wallet_transactions WHERE user_id = $1 AND kind = 'earning'
			          AND created_at >= date_trunc('month', now())), 0),
			COALESCE((SELECT sum(amount) FROM withdrawals WHERE user_id = $1 AND status = 'pending'), 0)`, id).
		Scan(&balance, &earned, &earnedMonth, &pending)
	if err != nil {
		return err
	}
	rows, err := s.db.Query(r.Context(), `SELECT id, amount, kind, job_id, description, created_at FROM wallet_transactions
		WHERE user_id = $1 ORDER BY created_at DESC LIMIT 100`, id)
	if err != nil {
		return err
	}
	txs, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (walletTx, error) {
		var t walletTx
		err := row.Scan(&t.ID, &t.Amount, &t.Kind, &t.JobID, &t.Description, &t.CreatedAt)
		return t, err
	})
	if err != nil {
		return err
	}
	rows, err = s.db.Query(r.Context(), `SELECT id, amount, method, destination, status, created_at, processed_at FROM withdrawals
		WHERE user_id = $1 ORDER BY created_at DESC LIMIT 50`, id)
	if err != nil {
		return err
	}
	wds, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (withdrawalView, error) {
		var v withdrawalView
		err := row.Scan(&v.ID, &v.Amount, &v.Method, &v.Destination, &v.Status, &v.CreatedAt, &v.ProcessedAt)
		return v, err
	})
	if err != nil {
		return err
	}
	return writeJSON(w, http.StatusOK, map[string]any{
		"balance":             balance,
		"total_earned":        earned,
		"earned_this_month":   earnedMonth,
		"pending_withdrawals": pending,
		"transactions":        txs,
		"withdrawals":         wds,
	})
}

// requestWithdrawal debits the wallet immediately (so the balance cannot be
// spent twice) and queues the payout for an admin (§29).
func (s *Server) requestWithdrawal(w http.ResponseWriter, r *http.Request) error {
	var req struct {
		Amount      int    `json:"amount"`
		Method      string `json:"method"`
		Destination string `json:"destination"`
	}
	if err := decode(r, &req); err != nil {
		return err
	}
	if req.Amount < 500 {
		return badRequest("minimum withdrawal is 500 FCFA")
	}
	if req.Method != "mobile_money" && req.Method != "bank" {
		return badRequest("method must be mobile_money or bank")
	}
	if strings.TrimSpace(req.Destination) == "" {
		return badRequest("destination is required")
	}
	id := me(r).ID
	err := s.inTx(r.Context(), func(tx pgx.Tx) error {
		// Serialise withdrawals per user.
		if _, err := tx.Exec(r.Context(), `SELECT 1 FROM users WHERE id = $1 FOR UPDATE`, id); err != nil {
			return err
		}
		var balance int
		if err := tx.QueryRow(r.Context(), `SELECT COALESCE(sum(amount), 0) FROM wallet_transactions WHERE user_id = $1`, id).Scan(&balance); err != nil {
			return err
		}
		if req.Amount > balance {
			return badRequest("insufficient balance")
		}
		if _, err := tx.Exec(r.Context(), `INSERT INTO withdrawals (user_id, amount, method, destination) VALUES ($1, $2, $3, $4)`,
			id, req.Amount, req.Method, strings.TrimSpace(req.Destination)); err != nil {
			return err
		}
		label := "Retrait Mobile Money"
		if req.Method == "bank" {
			label = "Retrait bancaire"
		}
		_, err := tx.Exec(r.Context(), `INSERT INTO wallet_transactions (user_id, amount, kind, description) VALUES ($1, $2, 'withdrawal', $3)`,
			id, -req.Amount, label)
		return err
	})
	if err != nil {
		return err
	}
	return s.getWallet(w, r)
}

func (s *Server) listNotifications(w http.ResponseWriter, r *http.Request) error {
	rows, err := s.db.Query(r.Context(), `SELECT id, kind, title, body, job_id, read_at, created_at FROM notifications
		WHERE user_id = $1 ORDER BY created_at DESC LIMIT 100`, me(r).ID)
	if err != nil {
		return err
	}
	out, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (notificationView, error) {
		var n notificationView
		err := row.Scan(&n.ID, &n.Kind, &n.Title, &n.Body, &n.JobID, &n.ReadAt, &n.CreatedAt)
		return n, err
	})
	if err != nil {
		return err
	}
	var unread int
	if err := s.db.QueryRow(r.Context(), `SELECT count(*) FROM notifications WHERE user_id = $1 AND read_at IS NULL`, me(r).ID).Scan(&unread); err != nil {
		return err
	}
	return writeJSON(w, http.StatusOK, map[string]any{"items": out, "unread": unread})
}

func (s *Server) markNotificationsRead(w http.ResponseWriter, r *http.Request) error {
	var req struct {
		IDs []string `json:"ids"` // empty = all
	}
	if err := decode(r, &req); err != nil {
		return err
	}
	_, err := s.db.Exec(r.Context(), `UPDATE notifications SET read_at = now()
		WHERE user_id = $1 AND read_at IS NULL AND (cardinality($2::uuid[]) = 0 OR id = ANY($2::uuid[]))`, me(r).ID, req.IDs)
	if err != nil {
		return err
	}
	w.WriteHeader(http.StatusNoContent)
	return nil
}
