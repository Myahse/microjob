package api

import (
	"context"
	"time"

	"github.com/myahse/microjob/backend/internal/realtime"
)

// note is a user notification (§30). Notifications are built inside a
// transaction and delivered with s.send once it has committed.
type note struct {
	UserID string
	Kind   string
	Title  string
	Body   string
	JobID  string
}

type notificationView struct {
	ID        string     `json:"id"`
	Kind      string     `json:"kind"`
	Title     string     `json:"title"`
	Body      string     `json:"body"`
	JobID     *string    `json:"job_id"`
	ReadAt    *time.Time `json:"read_at"`
	CreatedAt time.Time  `json:"created_at"`
}

// send stores notifications and pushes them to connected clients. Push
// delivery to phones (FCM/APNs) plugs in here.
func (s *Server) send(ctx context.Context, notes ...note) {
	for _, n := range notes {
		if n.UserID == "" {
			continue
		}
		var jobID *string
		if n.JobID != "" {
			jobID = &n.JobID
		}
		v := notificationView{Kind: n.Kind, Title: n.Title, Body: n.Body, JobID: jobID}
		err := s.db.QueryRow(ctx, `INSERT INTO notifications (user_id, kind, title, body, job_id)
			VALUES ($1, $2, $3, $4, $5) RETURNING id, created_at`,
			n.UserID, n.Kind, n.Title, n.Body, jobID).Scan(&v.ID, &v.CreatedAt)
		if err != nil {
			s.log.Error("store notification", "err", err)
			continue
		}
		s.hub.Publish(n.UserID, realtime.Event{Type: "notification", Data: v})
	}
}

// jobChanged tells participants to refresh a job.
func (s *Server) jobChanged(jobID, status string, userIDs ...string) {
	for _, id := range userIDs {
		if id != "" {
			s.hub.Publish(id, realtime.Event{Type: "job", Data: map[string]string{"job_id": jobID, "status": status}})
		}
	}
}
