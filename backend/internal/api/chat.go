package api

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/myahse/microjob/backend/internal/realtime"
)

type messageView struct {
	ID            string    `json:"id"`
	JobID         string    `json:"job_id"`
	SenderID      *string   `json:"sender_id"`
	Kind          string    `json:"kind"`
	Body          string    `json:"body"`
	AttachmentURL string    `json:"attachment_url"`
	Latitude      *float64  `json:"latitude"`
	Longitude     *float64  `json:"longitude"`
	CreatedAt     time.Time `json:"created_at"`
}

type conversation struct {
	ID       string
	ClientID string
	WorkerID string
	Current  bool // the conversation's worker is still assigned
}

// conversationFor finds the viewer's conversation on a job: the client
// talks to the currently assigned worker, a worker to the client.
func (s *Server) conversationFor(ctx context.Context, jobID string, p *principal) (*conversation, error) {
	c := &conversation{}
	err := s.db.QueryRow(ctx, `SELECT c.id, c.client_id, c.worker_id, j.worker_id IS NOT DISTINCT FROM c.worker_id
		FROM conversations c JOIN jobs j ON j.id = c.job_id
		WHERE c.job_id = $1 AND (c.worker_id = $2 OR (c.client_id = $2 AND c.worker_id = j.worker_id)
		      OR ($3 AND c.worker_id = j.worker_id))
		ORDER BY c.created_at DESC LIMIT 1`, jobID, p.ID, p.IsAdmin).Scan(&c.ID, &c.ClientID, &c.WorkerID, &c.Current)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, notFound("no conversation for this job yet")
	}
	return c, err
}

func (s *Server) listMessages(w http.ResponseWriter, r *http.Request) error {
	c, err := s.conversationFor(r.Context(), r.PathValue("id"), me(r))
	if err != nil {
		return err
	}
	rows, err := s.db.Query(r.Context(), `SELECT id, sender_id, kind, body, attachment_url, latitude, longitude, created_at
		FROM messages WHERE conversation_id = $1 ORDER BY created_at, id LIMIT 1000`, c.ID)
	if err != nil {
		return err
	}
	msgs, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (messageView, error) {
		m := messageView{JobID: r.PathValue("id")}
		err := row.Scan(&m.ID, &m.SenderID, &m.Kind, &m.Body, &m.AttachmentURL, &m.Latitude, &m.Longitude, &m.CreatedAt)
		return m, err
	})
	if err != nil {
		return err
	}
	return writeJSON(w, http.StatusOK, msgs)
}

// sendMessage posts text, a photo or a location in the job chat (§18).
func (s *Server) sendMessage(w http.ResponseWriter, r *http.Request) error {
	var req struct {
		Kind          string   `json:"kind"`
		Body          string   `json:"body"`
		AttachmentURL string   `json:"attachment_url"`
		Latitude      *float64 `json:"latitude"`
		Longitude     *float64 `json:"longitude"`
	}
	if err := decode(r, &req); err != nil {
		return err
	}
	if req.Kind == "" {
		req.Kind = "text"
	}
	req.Body = strings.TrimSpace(req.Body)
	switch req.Kind {
	case "text":
		if req.Body == "" {
			return badRequest("message is empty")
		}
	case "image":
		if req.AttachmentURL == "" {
			return badRequest("attachment_url is required")
		}
	case "location":
		if req.Latitude == nil || req.Longitude == nil {
			return badRequest("latitude and longitude are required")
		}
	default:
		return badRequest("kind must be text, image or location")
	}
	if len(req.Body) > 4000 {
		return badRequest("message too long")
	}
	p := me(r)
	c, err := s.conversationFor(r.Context(), r.PathValue("id"), p)
	if err != nil {
		return err
	}
	if p.ID != c.ClientID && p.ID != c.WorkerID {
		return forbidden("admins cannot post in user conversations")
	}
	if !c.Current {
		return forbidden("this conversation is closed")
	}
	m := messageView{JobID: r.PathValue("id"), SenderID: &p.ID, Kind: req.Kind, Body: req.Body,
		AttachmentURL: req.AttachmentURL, Latitude: req.Latitude, Longitude: req.Longitude}
	err = s.db.QueryRow(r.Context(), `INSERT INTO messages (conversation_id, sender_id, kind, body, attachment_url, latitude, longitude)
		VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING id, created_at`,
		c.ID, p.ID, req.Kind, req.Body, req.AttachmentURL, req.Latitude, req.Longitude).Scan(&m.ID, &m.CreatedAt)
	if err != nil {
		return err
	}
	other := c.ClientID
	if p.ID == c.ClientID {
		other = c.WorkerID
	}
	event := realtime.Event{Type: "message", Data: m}
	s.hub.Publish(other, event)
	s.hub.Publish(p.ID, event)
	preview := req.Body
	switch req.Kind {
	case "image":
		preview = "📷 Photo"
	case "location":
		preview = "📍 Position partagée"
	}
	if len(preview) > 120 {
		preview = preview[:120] + "…"
	}
	s.send(r.Context(), note{UserID: other, Kind: "message", JobID: m.JobID, Title: "💬 " + p.Name, Body: preview})
	return writeJSON(w, http.StatusCreated, m)
}

type conversationView struct {
	JobID       string     `json:"job_id"`
	JobTitle    string     `json:"job_title"`
	JobStatus   string     `json:"job_status"`
	Other       personRef  `json:"other"`
	LastMessage *string    `json:"last_message"`
	LastAt      *time.Time `json:"last_at"`
	Closed      bool       `json:"closed"`
}

func (s *Server) listConversations(w http.ResponseWriter, r *http.Request) error {
	p := me(r)
	rows, err := s.db.Query(r.Context(), `SELECT c.job_id, j.title, j.status,
			o.id, o.first_name, o.last_name, o.avatar_url,
			lm.body, lm.created_at, j.worker_id IS DISTINCT FROM c.worker_id
		FROM conversations c
		JOIN jobs j ON j.id = c.job_id
		JOIN users o ON o.id = CASE WHEN c.client_id = $1 THEN c.worker_id ELSE c.client_id END
		LEFT JOIN LATERAL (SELECT CASE m.kind WHEN 'image' THEN '📷 Photo' WHEN 'location' THEN '📍 Position' ELSE m.body END AS body, m.created_at
			FROM messages m WHERE m.conversation_id = c.id ORDER BY m.created_at DESC LIMIT 1) lm ON TRUE
		WHERE c.worker_id = $1 OR (c.client_id = $1 AND c.worker_id = j.worker_id)
		ORDER BY COALESCE(lm.created_at, c.created_at) DESC LIMIT 100`, p.ID)
	if err != nil {
		return err
	}
	out, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (conversationView, error) {
		var v conversationView
		err := row.Scan(&v.JobID, &v.JobTitle, &v.JobStatus, &v.Other.ID, &v.Other.FirstName, &v.Other.LastName, &v.Other.AvatarURL,
			&v.LastMessage, &v.LastAt, &v.Closed)
		return v, err
	})
	if err != nil {
		return err
	}
	return writeJSON(w, http.StatusOK, out)
}
