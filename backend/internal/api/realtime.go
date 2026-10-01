package api

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/coder/websocket"
)

const maxUpload = 5 << 20

var imageExt = map[string]string{
	"image/jpeg": ".jpg",
	"image/png":  ".png",
	"image/webp": ".webp",
	"image/gif":  ".gif",
}

// upload stores an image (avatar, job photo, chat photo) and returns its
// URL. Swap for object storage (S3, GCS) in production.
func (s *Server) upload(w http.ResponseWriter, r *http.Request) error {
	r.Body = http.MaxBytesReader(w, r.Body, maxUpload+1024)
	file, _, err := r.FormFile("file")
	if err != nil {
		return badRequest("multipart field 'file' is required (max 5 MB)")
	}
	defer file.Close()
	head := make([]byte, 512)
	n, _ := io.ReadFull(file, head)
	ext, ok := imageExt[http.DetectContentType(head[:n])]
	if !ok {
		return badRequest("only JPEG, PNG, WebP or GIF images are accepted")
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return err
	}
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	name := hex.EncodeToString(b) + ext
	if err := os.MkdirAll(s.uploadDir, 0o755); err != nil {
		return err
	}
	out, err := os.Create(filepath.Join(s.uploadDir, name))
	if err != nil {
		return err
	}
	defer out.Close()
	if _, err := io.Copy(out, file); err != nil {
		return err
	}
	return writeJSON(w, http.StatusCreated, map[string]string{"url": "/uploads/" + name})
}

// websocket streams realtime events (notifications, chat, job updates).
// Browsers cannot set headers on WebSocket requests, so the token travels
// in the query string; auth is token-based, so any origin may connect.
func (s *Server) websocket(w http.ResponseWriter, r *http.Request) error {
	p, err := s.authenticate(r.Context(), r.URL.Query().Get("token"))
	if err != nil {
		return err
	}
	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{InsecureSkipVerify: true})
	if err != nil {
		return nil // Accept already wrote the response
	}
	defer conn.CloseNow()

	events, stop := s.hub.Subscribe(p.ID)
	defer stop()
	ctx := conn.CloseRead(r.Context())
	ping := time.NewTicker(25 * time.Second)
	defer ping.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case e := <-events:
			b, err := json.Marshal(e)
			if err != nil {
				return nil
			}
			wctx, cancel := context.WithTimeout(ctx, 5*time.Second)
			err = conn.Write(wctx, websocket.MessageText, b)
			cancel()
			if err != nil {
				return nil
			}
		case <-ping.C:
			wctx, cancel := context.WithTimeout(ctx, 5*time.Second)
			err := conn.Ping(wctx)
			cancel()
			if err != nil {
				return nil
			}
		}
	}
}
