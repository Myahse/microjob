# MicroJob

**L'Uber des petits jobs**: a marketplace for small local on-demand services.

> J'ai besoin de quelqu'un. Je poste. Quelqu'un proche accepte. Le job est fait.

A client posts a small job (install Windows, move a piece of furniture, design a flyer…). The backend matches compatible workers nearby and notifies them. The first worker to accept gets the job (fixed price), or the client picks an offer ("Proposez votre prix"). The two sides then chat. The worker reports progress, the client confirms completion, the escrowed payment goes to the worker's wallet minus the platform commission, and both sides rate each other.

| Folder | Stack | What it is |
|---|---|---|
| [`backend/`](backend) | Go 1.26, PostgreSQL, `net/http`, pgx, WebSocket | REST + realtime API, matching, escrow, admin |
| [`web/`](web) | React 19, TypeScript, Vite, Tailwind CSS 4 | Web app for clients and workers, plus the admin dashboard |
| [`mobile/`](mobile) | Flutter (Android, iOS) | Mobile app for clients and workers |

## Quick start

### With Docker

```bash
docker compose up --build
```

- Web app: http://localhost:3000. The admin account is `admin@microjob.ci` / `admin12345`.
- API: http://localhost:8080
- Demo data for the Cocody launch zone is seeded (`SEED_DEMO=true`). The client is `aicha`; the workers are `mohammed` (tech), `awa` (design), `yao` (maison), `fatou` (courses) and `ismael` (études). Every demo password is `microjob123`.

### Without Docker

```bash
# 1. PostgreSQL 16 with a database "microjob" (user/password microjob)

# 2. API (migrations run at startup)
cd backend
SEED_DEMO=true ADMIN_EMAIL=admin@microjob.ci ADMIN_PASSWORD=admin12345 go run ./cmd/api

# 3. Web app: Vite proxies /api to :8080
cd web && npm install && npm run dev        # http://localhost:5173

# 4. Mobile app
cd mobile && flutter pub get
flutter run --dart-define=API_URL=http://10.0.2.2:8080     # Android emulator
flutter run --dart-define=API_URL=http://<your-LAN-IP>:8080 # physical phone
```

### Tests

```bash
cd backend && go test ./...    # domain unit tests
# Full HTTP integration tests (they wipe that database):
MICROJOB_TEST_DATABASE_URL=postgres://microjob:microjob@localhost:5432/microjob_test?sslmode=disable go test ./...

cd web && npm run build        # typecheck + build
cd mobile && flutter analyze && flutter test
```

CI (`.github/workflows/ci.yml`) runs all three.

## Job lifecycle

```
POSTED → MATCHING → ASSIGNED → WORKER_ON_THE_WAY → ARRIVED → STARTED
       → AWAITING_CONFIRMATION → COMPLETED → PAID → RATED
side exits: CANCELLED · DISPUTED (funds stay blocked until an admin decides)
```

| Step | Who | What happens |
|---|---|---|
| Post | client | Job created. The matcher scores compatible workers and notifies the best ones: `🔥 Nouveau job à proximité`. |
| Accept / choose | worker / client | Fixed price: first tap on **ACCEPT JOB** wins; a row lock prevents double assignment. Offers: the client picks an offer. Either way, the payment is **held in escrow**, a chat opens and the exact address is revealed to the worker. |
| Progress | worker | En route → **ARRIVED** → **START JOB** → **COMPLETE JOB**. Remote jobs can start directly. While travelling, the worker can share a live position and the client sees "Mohammed est à 1.2 km". |
| Confirm | client | **CONFIRM COMPLETION** releases the escrow. Example: worker 9 000 FCFA, platform 1 000 FCFA. |
| Rate | both | One review per side per job. When both have rated, the job becomes `RATED`. |
| Cancel | either | The system records who cancelled, why, when, and whether it hurts reputation. A client cancel refunds the escrow. A worker cancel refunds it and sends the job back to matching without that worker. |
| Report | either | **Report a problem** freezes the escrow (`DISPUTED`). The admin can refund the client, pay the worker, split the amount, or dismiss the report. |

### Matching (§12)

Candidates are active, **available** workers who have a skill in the job's category, are within their own radius of the job, haven't declined it and haven't previously cancelled it. They are ranked by:

```
score = 0.5 · proximity + 0.3 · reputation + 0.2 · relevant history
```

Reputation is the rating shrunk toward a neutral prior when there are few reviews, so a single 5★ can't outrank an established 4.8★. Relevant history counts completed jobs in the same category, plus a bonus when the worker has the job's exact skill. The top `matching_max_notified` workers (default 20) are notified.

### Trust & privacy

- **Location privacy:** before assignment, workers see only the zone and a distance rounded to 100 m. The exact address and coordinates go to the client and the assigned worker only. The map shows counts ("12 workers disponibles dans cette zone"), never positions.
- **Reputation:** ratings, completed jobs, completion rate and badges are computed from stored reviews and finished jobs. Users can't edit them. Badges are *Identité vérifiée*, *Top Worker* and *<Catégorie> vérifié*.
- **Verification:** admins verify identity, phone, email and individual skills.
- **Money:** amounts are integers in FCFA. A withdrawal debits the wallet immediately, under a per-user lock, so a balance can't be spent twice. A rejected withdrawal is credited back.

## Architecture

```
          ┌──────────────┐     ┌──────────────┐
          │ Flutter app  │     │ React web    │  (client · worker · admin)
          └──────┬───────┘     └──────┬───────┘
          REST + WebSocket (JWT)      │
                 └──────────┬─────────┘
                    ┌───────▼────────┐
                    │   Go API       │  matching · escrow · chat · notifications
                    │ internal/api   │──► payments.Provider (simulated → Mobile Money / card)
                    │ internal/domain│
                    └───────┬────────┘
                    ┌───────▼────────┐
                    │  PostgreSQL    │
                    └────────────────┘
```

- `internal/domain` holds the pure rules: status transitions, cancellation rules, commission split, distances and match scores. It is unit-tested.
- `internal/api` contains the HTTP handlers. Every state change runs in a transaction that locks the job row. Notifications are stored and pushed after commit.
- `internal/payments` defines the `Provider` interface (`Hold`, `Release`, `Refund`). The MVP uses `simulated`, as §38 allows. Orange Money, MTN MoMo, Wave or card acquirers plug in behind the same interface.
- `internal/realtime` is an in-process WebSocket hub that pushes `notification`, `message` and `job` events. Run a single API instance, or swap it for Redis/PG `LISTEN/NOTIFY` to scale out.
- **Commission** is configurable globally (`default_commission_rate`, 10 %) and per category. The rate is frozen on each payment when the money is held.
- **Uploads** are stored on local disk under `UPLOAD_DIR`. Swap for S3/GCS in production.

## API overview

All endpoints are under `/api`. Authenticate with `Authorization: Bearer <token>`.

| Area | Endpoints |
|---|---|
| Auth | `POST auth/register`, `POST auth/login`, `GET/PATCH me`, `PUT me/worker`, `PUT me/availability`, `GET users/{id}` |
| Discovery | `GET categories`, `GET workers/search?q=&lat=&lng=`, `GET workers/nearby-count?lat=&lng=&radius_km=` |
| Jobs | `POST jobs`, `GET jobs/mine?scope=active\|done\|cancelled`, `GET jobs/missions?scope=active\|done\|applied`, `GET jobs/available`, `GET jobs/{id}` |
| Job actions | `POST jobs/{id}/accept`, `/decline`, `GET\|POST /applications`, `POST /applications/{appId}/choose`, `POST /status`, `/confirm`, `/cancel`, `PUT /location`, `POST /reviews`, `/reports` |
| Chat | `GET\|POST jobs/{id}/messages` (text, image, location, system), `GET conversations` |
| Wallet | `GET wallet`, `POST wallet/withdrawals` |
| Misc | `GET notifications`, `POST notifications/read`, `POST uploads` (multipart `file`), `GET ws?token=` |
| Admin | `GET admin/stats`, `GET/PATCH admin/users[/{id}]`, `POST admin/users/{id}/skills/{skillId}/verify`, `GET admin/jobs`, `POST admin/jobs/{id}/cancel`, `GET/POST/PATCH admin/categories`, `POST admin/categories/{id}/skills`, `PATCH admin/skills/{id}`, `GET admin/reports`, `POST admin/reports/{id}/resolve`, `GET admin/payments`, `GET/POST admin/withdrawals`, `GET/PUT admin/settings` |

## MVP scope (§38)

| | Covered |
|---|---|
| **Client** | Account ✅ · post job with location & budget ✅ · fixed price or offers ✅ · see interested workers and choose ✅ · chat (text/photo/location) ✅ · live status + worker distance ✅ · confirm completion ✅ · rate ✅ · report a problem ✅ · search workers ✅ |
| **Worker** | Professional profile ✅ · skills ✅ · availability + radius ✅ · nearby jobs feed ✅ · accept/decline/offer ✅ · chat ✅ · progress statuses ✅ · wallet & earnings ✅ · withdrawals ✅ · rate the client ✅ |
| **Admin** | Users: suspend/ban/verify identity & skills ✅ · jobs: moderate/cancel ✅ · categories & skills CRUD + commission per category ✅ · disputes: refund / pay / partial ✅ · payments & withdrawals ✅ · KPIs (GMV, commission, acceptance/completion rate, time to assignment, active users) ✅ |

**Not built yet (V2+):** real Mobile Money/card provider, phone push notifications (FCM/APNs), SMS/email verification flows, masked phone calls, an interactive map (Mapbox), direct hire from search, Worker Pro subscriptions, and recurring services.

## Launch strategy reminder (§42)

Start with **one zone**, Cocody, and four categories: Tech, Design, Services, Maison. The neighbourhood list lives in `web/src/lib/format.ts` and `mobile/lib/util/format.dart`. Recruit workers before acquiring clients; `nearby-count` on the client home shows how dense the supply is.
