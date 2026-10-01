// Package seed creates the admin account and optional demo data for the
// Cocody launch zone (§42).
package seed

import (
	"context"
	"log/slog"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/myahse/microjob/backend/internal/auth"
)

// DemoPassword is the password of every demo account.
const DemoPassword = "microjob123"

// Admin creates the admin account if it does not exist yet.
func Admin(ctx context.Context, db *pgxpool.Pool, email, password string) error {
	if email == "" || password == "" {
		return nil
	}
	hash, err := auth.HashPassword(password)
	if err != nil {
		return err
	}
	tag, err := db.Exec(ctx, `INSERT INTO users (first_name, last_name, username, phone, email, password_hash, is_admin,
			email_verified, identity_verified)
		VALUES ('Admin', 'MicroJob', 'admin', '+2250000000000', $1, $2, TRUE, TRUE, TRUE)
		ON CONFLICT (email) DO NOTHING`, strings.ToLower(email), hash)
	if err != nil {
		return err
	}
	if tag.RowsAffected() > 0 {
		slog.Info("admin account created", "email", email)
	}
	return nil
}

type demoWorker struct {
	first, last, username, phone, headline, zone string
	lat, lng                                     float64
	rate                                         int
	skills                                       []string
}

// Demo adds a few workers and a client around Cocody. Idempotent.
func Demo(ctx context.Context, db *pgxpool.Pool) error {
	hash, err := auth.HashPassword(DemoPassword)
	if err != nil {
		return err
	}
	workers := []demoWorker{
		{"Mohammed", "Koné", "mohammed", "+2250700000001", "💻 Développeur / Technicien", "Riviera", 5.3600, -3.9700, 5000,
			[]string{"Installation Windows", "Réparation PC", "Configuration Wi-Fi", "Programmation"}},
		{"Awa", "Traoré", "awa", "+2250700000002", "🎨 Graphiste", "Angré", 5.3950, -3.9900, 7000,
			[]string{"Flyer", "Logo", "Affiche", "Retouche photo"}},
		{"Yao", "Kouassi", "yao", "+2250700000003", "🏠 Bricoleur", "II Plateaux", 5.3700, -3.9950, 3000,
			[]string{"Montage", "Petites réparations", "Déplacement d'objet", "Nettoyage"}},
		{"Fatou", "Diallo", "fatou", "+2250700000004", "📦 Coursière", "Cocody Centre", 5.3480, -3.9870, 2500,
			[]string{"Livraison", "Courses"}},
		{"Ismaël", "Bamba", "ismael", "+2250700000005", "🎓 Tuteur maths & langues", "Riviera Palmeraie", 5.3720, -3.9450, 4000,
			[]string{"Mathématiques", "Langues", "Tutorat"}},
	}
	return pgx.BeginFunc(ctx, db, func(tx pgx.Tx) error {
		for _, w := range workers {
			var id string
			err := tx.QueryRow(ctx, `INSERT INTO users (first_name, last_name, username, phone, email, password_hash, is_client, is_worker,
					phone_verified, identity_verified, latitude, longitude, zone)
				VALUES ($1, $2, $3, $4, $3 || '@demo.microjob.ci', $5, TRUE, TRUE, TRUE, TRUE, $6, $7, $8)
				ON CONFLICT (username) DO NOTHING RETURNING id`,
				w.first, w.last, w.username, w.phone, hash, w.lat, w.lng, w.zone).Scan(&id)
			if err == pgx.ErrNoRows {
				continue // already seeded
			}
			if err != nil {
				return err
			}
			if _, err := tx.Exec(ctx, `INSERT INTO worker_profiles (user_id, headline, available, hourly_rate, radius_km) VALUES ($1, $2, TRUE, $3, 10)`,
				id, w.headline, w.rate); err != nil {
				return err
			}
			if _, err := tx.Exec(ctx, `INSERT INTO user_skills (user_id, skill_id, verified) SELECT $1, id, TRUE FROM skills WHERE name = ANY($2)`,
				id, w.skills); err != nil {
				return err
			}
		}
		_, err := tx.Exec(ctx, `INSERT INTO users (first_name, last_name, username, phone, email, password_hash, is_client,
				phone_verified, latitude, longitude, zone)
			VALUES ('Aïcha', 'Yapi', 'aicha', '+2250700000010', 'aicha@demo.microjob.ci', $1, TRUE, TRUE, 5.3550, -3.9800, 'Cocody')
			ON CONFLICT (username) DO NOTHING`, hash)
		return err
	})
}
