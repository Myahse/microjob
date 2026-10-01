package api_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/myahse/microjob/backend/internal/api"
	"github.com/myahse/microjob/backend/internal/auth"
	"github.com/myahse/microjob/backend/internal/db"
	"github.com/myahse/microjob/backend/internal/payments"
	"github.com/myahse/microjob/backend/internal/seed"
)

// These tests need a disposable PostgreSQL database; its schema is wiped.
//   MICROJOB_TEST_DATABASE_URL=postgres://microjob:microjob@localhost:5432/microjob_test?sslmode=disable go test ./...

type client struct {
	t     *testing.T
	base  string
	token string
	id    string
}

func (c *client) do(method, path string, body any, want int) map[string]any {
	c.t.Helper()
	raw := c.doRaw(method, path, body, want)
	if len(raw) == 0 {
		return nil
	}
	var out map[string]any
	if raw[0] == '[' {
		var list []any
		if err := json.Unmarshal(raw, &list); err != nil {
			c.t.Fatal(err)
		}
		return map[string]any{"items": list}
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		c.t.Fatalf("%s %s: decode %q: %v", method, path, raw, err)
	}
	return out
}

func (c *client) doRaw(method, path string, body any, want int) []byte {
	c.t.Helper()
	var r io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		r = bytes.NewReader(b)
	}
	req, _ := http.NewRequest(method, c.base+path, r)
	req.Header.Set("Content-Type", "application/json")
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		c.t.Fatal(err)
	}
	defer res.Body.Close()
	raw, _ := io.ReadAll(res.Body)
	if res.StatusCode != want {
		c.t.Fatalf("%s %s: status %d, want %d: %s", method, path, res.StatusCode, want, raw)
	}
	return raw
}

func setup(t *testing.T) string {
	url := os.Getenv("MICROJOB_TEST_DATABASE_URL")
	if url == "" {
		t.Skip("MICROJOB_TEST_DATABASE_URL not set")
	}
	ctx := context.Background()
	pool, err := db.Open(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	if _, err := pool.Exec(ctx, `DROP SCHEMA public CASCADE; CREATE SCHEMA public;`); err != nil {
		t.Fatal(err)
	}
	if err := db.Migrate(ctx, pool); err != nil {
		t.Fatal(err)
	}
	if err := seed.Admin(ctx, pool, "admin@microjob.ci", "adminpass123"); err != nil {
		t.Fatal(err)
	}
	srv := api.NewServer(api.Options{DB: pool, Tokens: auth.NewTokens("test"), Payments: payments.Simulated{},
		UploadDir: t.TempDir(), AllowedOrigins: []string{"*"}})
	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(ts.Close)
	return ts.URL
}

var phoneSeq = 0

func register(t *testing.T, base, username string, worker bool, lat, lng float64) *client {
	phoneSeq++
	c := &client{t: t, base: base}
	res := c.do("POST", "/api/auth/register", map[string]any{
		"first_name": username, "last_name": "Test", "username": username, "email": username + "@test.ci",
		"phone": fmt.Sprintf("+22507000%05d", phoneSeq), "password": "password123",
		"latitude": lat, "longitude": lng, "zone": "Cocody",
		"wants_services": !worker, "offers_services": worker,
	}, http.StatusCreated)
	c.token = res["token"].(string)
	c.id = res["user"].(map[string]any)["id"].(string)
	return c
}

func login(t *testing.T, base, ident, password string) *client {
	c := &client{t: t, base: base}
	res := c.do("POST", "/api/auth/login", map[string]any{"identifier": ident, "password": password}, http.StatusOK)
	c.token = res["token"].(string)
	c.id = res["user"].(map[string]any)["id"].(string)
	return c
}

func skillID(t *testing.T, c *client, name string) (int, int) {
	res := c.do("GET", "/api/categories", nil, http.StatusOK)
	for _, cat := range res["items"].([]any) {
		cm := cat.(map[string]any)
		for _, sk := range cm["skills"].([]any) {
			sm := sk.(map[string]any)
			if sm["name"] == name {
				return int(cm["id"].(float64)), int(sm["id"].(float64))
			}
		}
	}
	t.Fatalf("skill %q not found", name)
	return 0, 0
}

func status(job map[string]any) string { return job["status"].(string) }

func balance(c *client) float64 {
	return c.do("GET", "/api/wallet", nil, http.StatusOK)["balance"].(float64)
}

func TestMarketplaceLoop(t *testing.T) {
	base := setup(t)
	cocody := [2]float64{5.3550, -3.9800}

	aicha := register(t, base, "aicha", false, cocody[0], cocody[1])
	mohammed := register(t, base, "mohammed", true, 5.3600, -3.9700) // ~1.2 km away
	yao := register(t, base, "yao", true, 5.3700, -3.9950)           // ~2.3 km away
	far := register(t, base, "far", true, 5.2500, -4.1000)           // > 10 km away

	catID, skID := skillID(t, aicha, "Installation Windows")
	for _, w := range []*client{mohammed, yao, far} {
		w.do("PUT", "/api/me/worker", map[string]any{"headline": "Technicien", "skill_ids": []int{skID}}, http.StatusOK)
		w.do("PUT", "/api/me/availability", map[string]any{"available": true}, http.StatusOK)
	}

	count := aicha.do("GET", fmt.Sprintf("/api/workers/nearby-count?lat=%f&lng=%f&radius_km=5", cocody[0], cocody[1]), nil, http.StatusOK)
	if count["count"].(float64) != 2 {
		t.Fatalf("nearby count = %v, want 2", count["count"])
	}

	// --- Fixed-price job: post → match → accept → progress → confirm → pay → rate.
	res := aicha.do("POST", "/api/jobs", map[string]any{
		"category_id": catID, "skill_id": skID, "title": "Installer Windows sur mon PC",
		"description": "Installation complète de Windows 11", "latitude": cocody[0], "longitude": cocody[1],
		"address": "Rue des Jardins, villa 12", "zone": "Cocody", "pricing_mode": "fixed", "budget": 10000, "duration_minutes": 120,
	}, http.StatusCreated)
	if res["notified_workers"].(float64) != 2 {
		t.Fatalf("notified %v workers, want 2", res["notified_workers"])
	}
	job := res["job"].(map[string]any)
	jobID := job["id"].(string)
	if status(job) != "MATCHING" {
		t.Fatalf("status %s, want MATCHING", status(job))
	}

	feed := mohammed.do("GET", "/api/jobs/available", nil, http.StatusOK)["items"].([]any)
	if len(feed) != 1 {
		t.Fatalf("feed has %d jobs, want 1", len(feed))
	}
	fj := feed[0].(map[string]any)
	if _, leaked := fj["address"]; leaked {
		t.Fatal("exact address must be hidden before assignment")
	}
	if d := fj["distance_km"].(float64); d < 0.5 || d > 2 {
		t.Fatalf("distance %v", d)
	}
	if n := len(far.do("GET", "/api/jobs/available", nil, http.StatusOK)["items"].([]any)); n != 0 {
		t.Fatalf("far worker sees %d jobs", n)
	}
	notifs := mohammed.do("GET", "/api/notifications", nil, http.StatusOK)
	if notifs["unread"].(float64) < 1 {
		t.Fatal("worker should be notified of the nearby job")
	}

	aicha.do("POST", "/api/jobs/"+jobID+"/accept", nil, http.StatusForbidden)
	job = mohammed.do("POST", "/api/jobs/"+jobID+"/accept", nil, http.StatusOK)
	if status(job) != "ASSIGNED" || job["address"] != "Rue des Jardins, villa 12" {
		t.Fatalf("after accept: %s %v", status(job), job["address"])
	}
	yao.do("POST", "/api/jobs/"+jobID+"/accept", nil, http.StatusConflict)

	job = aicha.do("GET", "/api/jobs/"+jobID, nil, http.StatusOK)
	pay := job["payment"].(map[string]any)
	if pay["status"] != "held" || pay["platform_fee"].(float64) != 1000 || pay["worker_amount"].(float64) != 9000 {
		t.Fatalf("payment %+v", pay)
	}

	aicha.do("POST", "/api/jobs/"+jobID+"/messages", map[string]any{"body": "Bonjour, vous êtes disponible à 18h ?"}, http.StatusCreated)
	mohammed.do("POST", "/api/jobs/"+jobID+"/messages", map[string]any{"body": "Oui, je serai là."}, http.StatusCreated)
	msgs := mohammed.do("GET", "/api/jobs/"+jobID+"/messages", nil, http.StatusOK)["items"].([]any)
	if len(msgs) != 3 { // system + 2
		t.Fatalf("%d messages", len(msgs))
	}
	yao.do("GET", "/api/jobs/"+jobID+"/messages", nil, http.StatusNotFound)

	mohammed.do("POST", "/api/jobs/"+jobID+"/status", map[string]any{"status": "AWAITING_CONFIRMATION"}, http.StatusConflict)
	mohammed.do("POST", "/api/jobs/"+jobID+"/status", map[string]any{"status": "WORKER_ON_THE_WAY"}, http.StatusOK)
	mohammed.do("PUT", "/api/jobs/"+jobID+"/location", map[string]any{"latitude": 5.3580, "longitude": -3.9750}, http.StatusNoContent)
	job = aicha.do("GET", "/api/jobs/"+jobID, nil, http.StatusOK)
	if job["assignment"].(map[string]any)["worker_distance_km"] == nil {
		t.Fatal("client should see the worker's distance")
	}
	mohammed.do("POST", "/api/jobs/"+jobID+"/status", map[string]any{"status": "ARRIVED"}, http.StatusOK)
	mohammed.do("POST", "/api/jobs/"+jobID+"/status", map[string]any{"status": "STARTED"}, http.StatusOK)
	aicha.do("POST", "/api/jobs/"+jobID+"/cancel", map[string]any{"reason": "changed my mind"}, http.StatusConflict)
	aicha.do("POST", "/api/jobs/"+jobID+"/confirm", nil, http.StatusConflict)
	mohammed.do("POST", "/api/jobs/"+jobID+"/status", map[string]any{"status": "AWAITING_CONFIRMATION"}, http.StatusOK)
	mohammed.do("POST", "/api/jobs/"+jobID+"/confirm", nil, http.StatusForbidden)
	job = aicha.do("POST", "/api/jobs/"+jobID+"/confirm", nil, http.StatusOK)
	if status(job) != "PAID" {
		t.Fatalf("status %s, want PAID", status(job))
	}
	if b := balance(mohammed); b != 9000 {
		t.Fatalf("wallet %v, want 9000", b)
	}

	aicha.do("POST", "/api/jobs/"+jobID+"/reviews", map[string]any{"rating": 5, "quality": 5, "comment": "Parfait"}, http.StatusOK)
	aicha.do("POST", "/api/jobs/"+jobID+"/reviews", map[string]any{"rating": 1}, http.StatusConflict)
	job = mohammed.do("POST", "/api/jobs/"+jobID+"/reviews", map[string]any{"rating": 4, "payment": 5}, http.StatusOK)
	if status(job) != "RATED" {
		t.Fatalf("status %s, want RATED", status(job))
	}
	prof := aicha.do("GET", "/api/users/"+mohammed.id, nil, http.StatusOK)["user"].(map[string]any)
	wk := prof["worker"].(map[string]any)
	if wk["rating"].(float64) != 5 || wk["completed_jobs"].(float64) != 1 || wk["completion_rate"].(float64) != 100 {
		t.Fatalf("worker profile %+v", wk)
	}
	if _, leaked := prof["email"]; leaked {
		t.Fatal("public profile must not expose email")
	}

	// --- Wallet withdrawal.
	mohammed.do("POST", "/api/wallet/withdrawals", map[string]any{"amount": 20000, "method": "mobile_money", "destination": "+2250700000001"}, http.StatusBadRequest)
	mohammed.do("POST", "/api/wallet/withdrawals", map[string]any{"amount": 2000, "method": "mobile_money", "destination": "+2250700000001"}, http.StatusOK)
	if b := balance(mohammed); b != 7000 {
		t.Fatalf("wallet %v, want 7000", b)
	}

	// --- Offers job: apply → choose → worker cancels → re-match → dispute.
	res = aicha.do("POST", "/api/jobs", map[string]any{
		"category_id": catID, "title": "Réparer mon PC portable", "latitude": cocody[0], "longitude": cocody[1],
		"zone": "Cocody", "pricing_mode": "offers", "budget": 0,
	}, http.StatusCreated)
	job2 := res["job"].(map[string]any)["id"].(string)
	yao.do("POST", "/api/jobs/"+job2+"/accept", nil, http.StatusBadRequest)
	yao.do("POST", "/api/jobs/"+job2+"/applications", map[string]any{"price": 8000, "message": "Dispo ce soir"}, http.StatusOK)
	apps := aicha.do("GET", "/api/jobs/"+job2+"/applications", nil, http.StatusOK)["items"].([]any)
	if len(apps) != 1 {
		t.Fatalf("%d applications", len(apps))
	}
	appID := apps[0].(map[string]any)["id"].(string)
	job = aicha.do("POST", "/api/jobs/"+job2+"/applications/"+appID+"/choose", nil, http.StatusOK)
	if status(job) != "ASSIGNED" || job["agreed_price"].(float64) != 8000 {
		t.Fatalf("after choose: %s %v", status(job), job["agreed_price"])
	}
	job = yao.do("POST", "/api/jobs/"+job2+"/cancel", map[string]any{"reason": "Empêchement"}, http.StatusOK)
	if status(job) != "MATCHING" {
		t.Fatalf("after worker cancel: %s", status(job))
	}
	// The worker who cancelled can still read the job (they applied) but cannot take it back.
	yao.do("POST", "/api/jobs/"+job2+"/applications", map[string]any{"price": 7000}, http.StatusForbidden)
	job = aicha.do("GET", "/api/jobs/"+job2, nil, http.StatusOK)
	if job["worker"] != nil || job["payment"].(map[string]any)["status"] != "refunded" {
		t.Fatalf("job after worker cancel: worker=%v payment=%v", job["worker"], job["payment"])
	}

	mohammed.do("POST", "/api/jobs/"+job2+"/applications", map[string]any{"price": 5000}, http.StatusOK)
	apps = aicha.do("GET", "/api/jobs/"+job2+"/applications", nil, http.StatusOK)["items"].([]any)
	for _, a := range apps {
		am := a.(map[string]any)
		if am["status"] == "pending" {
			appID = am["id"].(string)
		}
	}
	aicha.do("POST", "/api/jobs/"+job2+"/applications/"+appID+"/choose", nil, http.StatusOK)
	mohammed.do("POST", "/api/jobs/"+job2+"/status", map[string]any{"status": "STARTED"}, http.StatusOK)
	job = aicha.do("POST", "/api/jobs/"+job2+"/reports", map[string]any{"type": "bad_quality", "description": "Le PC ne démarre plus"}, http.StatusOK)
	if status(job) != "DISPUTED" {
		t.Fatalf("status %s, want DISPUTED", status(job))
	}
	mohammed.do("POST", "/api/jobs/"+job2+"/status", map[string]any{"status": "AWAITING_CONFIRMATION"}, http.StatusConflict)

	// --- Admin.
	aicha.do("GET", "/api/admin/stats", nil, http.StatusForbidden)
	admin := login(t, base, "admin@microjob.ci", "adminpass123")
	reports := admin.do("GET", "/api/admin/reports?status=open", nil, http.StatusOK)["items"].([]any)
	if len(reports) != 1 {
		t.Fatalf("%d open reports", len(reports))
	}
	reportID := reports[0].(map[string]any)["id"].(string)
	admin.do("POST", "/api/admin/reports/"+reportID+"/resolve", map[string]any{"resolution": "partial", "refund_amount": 9000}, http.StatusBadRequest)
	admin.do("POST", "/api/admin/reports/"+reportID+"/resolve", map[string]any{"resolution": "partial", "refund_amount": 2000, "note": "Travail partiel"}, http.StatusOK)
	job = aicha.do("GET", "/api/jobs/"+job2, nil, http.StatusOK)
	if status(job) != "PAID" {
		t.Fatalf("status %s, want PAID", status(job))
	}
	if b := balance(mohammed); b != 7000+2700 { // (5000 - 2000) * 90%
		t.Fatalf("wallet %v, want 9700", b)
	}

	stats := admin.do("GET", "/api/admin/stats", nil, http.StatusOK)
	if stats["jobs_posted"].(float64) != 2 || stats["commission"].(float64) != 1300 {
		t.Fatalf("stats posted=%v commission=%v", stats["jobs_posted"], stats["commission"])
	}

	wds := admin.do("GET", "/api/admin/withdrawals?status=pending", nil, http.StatusOK)["items"].([]any)
	admin.do("POST", "/api/admin/withdrawals/"+wds[0].(map[string]any)["id"].(string), map[string]any{"status": "rejected"}, http.StatusOK)
	if b := balance(mohammed); b != 11700 {
		t.Fatalf("wallet %v after rejected withdrawal, want 11700", b)
	}

	admin.do("PATCH", "/api/admin/users/"+yao.id, map[string]any{"status": "suspended"}, http.StatusOK)
	yao.do("GET", "/api/me", nil, http.StatusForbidden)
	admin.do("PUT", "/api/admin/settings", map[string]any{"default_commission_rate": "0.15"}, http.StatusOK)
	admin.do("PUT", "/api/admin/settings", map[string]any{"default_commission_rate": "2"}, http.StatusBadRequest)
	admin.do("PATCH", fmt.Sprintf("/api/admin/categories/%d", catID), map[string]any{"commission_rate": 0.08}, http.StatusOK)

	// Client cancels an open job: free, no payment involved.
	res = aicha.do("POST", "/api/jobs", map[string]any{"category_id": catID, "title": "Configurer mon Wi-Fi",
		"latitude": cocody[0], "longitude": cocody[1], "zone": "Cocody", "budget": 5000}, http.StatusCreated)
	job3 := res["job"].(map[string]any)["id"].(string)
	job = aicha.do("POST", "/api/jobs/"+job3+"/cancel", map[string]any{"reason": "Réglé tout seul"}, http.StatusOK)
	if status(job) != "CANCELLED" {
		t.Fatalf("status %s", status(job))
	}
	mohammed.do("GET", "/api/jobs/"+job3, nil, http.StatusNotFound)
}

func TestAuthValidation(t *testing.T) {
	base := setup(t)
	c := &client{t: t, base: base}
	c.do("POST", "/api/auth/register", map[string]any{"first_name": "A", "last_name": "B", "username": "x", "email": "a@b.c",
		"phone": "+2250102030405", "password": "password123"}, http.StatusBadRequest)
	register(t, base, "dupe", false, 5.35, -3.98)
	c.do("POST", "/api/auth/register", map[string]any{"first_name": "A", "last_name": "B", "username": "dupe", "email": "other@b.c",
		"phone": "+2250102030499", "password": "password123"}, http.StatusConflict)
	c.do("POST", "/api/auth/login", map[string]any{"identifier": "dupe", "password": "wrong-pass"}, http.StatusUnauthorized)
	login(t, base, "dupe@test.ci", "password123")
	c.do("GET", "/api/me", nil, http.StatusUnauthorized)
}
