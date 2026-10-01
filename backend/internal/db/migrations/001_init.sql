-- MicroJob initial schema.

CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE settings (
    key        TEXT PRIMARY KEY,
    value      TEXT NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

INSERT INTO settings (key, value) VALUES
    ('default_commission_rate', '0.10'),
    ('matching_max_notified', '20'),
    ('default_worker_radius_km', '10');

-- Users: one account can be client, worker and/or admin.
CREATE TABLE users (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    first_name        TEXT NOT NULL,
    last_name         TEXT NOT NULL,
    username          TEXT NOT NULL UNIQUE,
    phone             TEXT NOT NULL UNIQUE,
    email             TEXT NOT NULL UNIQUE,
    password_hash     TEXT NOT NULL,
    avatar_url        TEXT NOT NULL DEFAULT '',
    bio               TEXT NOT NULL DEFAULT '',
    is_client         BOOLEAN NOT NULL DEFAULT TRUE,
    is_worker         BOOLEAN NOT NULL DEFAULT FALSE,
    is_admin          BOOLEAN NOT NULL DEFAULT FALSE,
    status            TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'suspended', 'banned')),
    phone_verified    BOOLEAN NOT NULL DEFAULT FALSE,
    email_verified    BOOLEAN NOT NULL DEFAULT FALSE,
    identity_verified BOOLEAN NOT NULL DEFAULT FALSE,
    latitude          DOUBLE PRECISION,
    longitude         DOUBLE PRECISION,
    zone              TEXT NOT NULL DEFAULT '',
    -- Reputation as a client (rated by workers).
    client_rating_avg   NUMERIC(3,2) NOT NULL DEFAULT 0,
    client_rating_count INTEGER NOT NULL DEFAULT 0,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Professional profile, present once a user opts in as worker.
CREATE TABLE worker_profiles (
    user_id          UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    headline         TEXT NOT NULL DEFAULT '',
    available        BOOLEAN NOT NULL DEFAULT FALSE,
    hourly_rate      INTEGER NOT NULL DEFAULT 0,
    radius_km        DOUBLE PRECISION NOT NULL DEFAULT 10,
    rating_avg       NUMERIC(3,2) NOT NULL DEFAULT 0,
    rating_count     INTEGER NOT NULL DEFAULT 0,
    completed_jobs   INTEGER NOT NULL DEFAULT 0,
    accepted_jobs    INTEGER NOT NULL DEFAULT 0,
    cancelled_jobs   INTEGER NOT NULL DEFAULT 0,
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE categories (
    id              SERIAL PRIMARY KEY,
    slug            TEXT NOT NULL UNIQUE,
    name            TEXT NOT NULL,
    icon            TEXT NOT NULL DEFAULT '',
    commission_rate NUMERIC(5,4),           -- NULL = use default_commission_rate
    active          BOOLEAN NOT NULL DEFAULT TRUE,
    sort_order      INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE skills (
    id          SERIAL PRIMARY KEY,
    category_id INTEGER NOT NULL REFERENCES categories(id) ON DELETE CASCADE,
    name        TEXT NOT NULL,
    active      BOOLEAN NOT NULL DEFAULT TRUE,
    UNIQUE (category_id, name)
);

CREATE TABLE user_skills (
    user_id  UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    skill_id INTEGER NOT NULL REFERENCES skills(id) ON DELETE CASCADE,
    verified BOOLEAN NOT NULL DEFAULT FALSE,
    PRIMARY KEY (user_id, skill_id)
);

CREATE TABLE jobs (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    client_id        UUID NOT NULL REFERENCES users(id),
    category_id      INTEGER NOT NULL REFERENCES categories(id),
    skill_id         INTEGER REFERENCES skills(id),
    title            TEXT NOT NULL,
    description      TEXT NOT NULL DEFAULT '',
    latitude         DOUBLE PRECISION NOT NULL,
    longitude        DOUBLE PRECISION NOT NULL,
    address          TEXT NOT NULL DEFAULT '',   -- exact address, private until assignment
    zone             TEXT NOT NULL DEFAULT '',   -- public area, e.g. "Cocody"
    pricing_mode     TEXT NOT NULL DEFAULT 'fixed' CHECK (pricing_mode IN ('fixed', 'offers')),
    budget           INTEGER NOT NULL CHECK (budget >= 0),
    agreed_price     INTEGER,
    scheduled_at     TIMESTAMPTZ,
    duration_minutes INTEGER NOT NULL DEFAULT 60,
    photos           TEXT[] NOT NULL DEFAULT '{}',
    status           TEXT NOT NULL DEFAULT 'POSTED',
    worker_id        UUID REFERENCES users(id),
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX jobs_status_idx ON jobs (status);
CREATE INDEX jobs_client_idx ON jobs (client_id);
CREATE INDEX jobs_worker_idx ON jobs (worker_id);

-- Offers (pricing_mode = offers) and expressions of interest.
CREATE TABLE job_applications (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    job_id     UUID NOT NULL REFERENCES jobs(id) ON DELETE CASCADE,
    worker_id  UUID NOT NULL REFERENCES users(id),
    price      INTEGER NOT NULL CHECK (price >= 0),
    message    TEXT NOT NULL DEFAULT '',
    status     TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'accepted', 'rejected', 'withdrawn')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (job_id, worker_id)
);

-- Workers notified by the matcher, and whether they declined.
CREATE TABLE job_dispatches (
    job_id      UUID NOT NULL REFERENCES jobs(id) ON DELETE CASCADE,
    worker_id   UUID NOT NULL REFERENCES users(id),
    distance_km DOUBLE PRECISION NOT NULL,
    score       DOUBLE PRECISION NOT NULL,
    declined    BOOLEAN NOT NULL DEFAULT FALSE,
    notified_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (job_id, worker_id)
);

CREATE TABLE job_assignments (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    job_id        UUID NOT NULL REFERENCES jobs(id) ON DELETE CASCADE,
    worker_id     UUID NOT NULL REFERENCES users(id),
    price         INTEGER NOT NULL,
    assigned_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    on_the_way_at TIMESTAMPTZ,
    arrived_at    TIMESTAMPTZ,
    started_at    TIMESTAMPTZ,
    finished_at   TIMESTAMPTZ,   -- worker pressed COMPLETE JOB
    completed_at  TIMESTAMPTZ,   -- client confirmed
    cancelled_at  TIMESTAMPTZ,
    worker_latitude   DOUBLE PRECISION,
    worker_longitude  DOUBLE PRECISION,
    location_updated_at TIMESTAMPTZ
);
-- At most one live assignment per job.
CREATE UNIQUE INDEX job_assignments_live_idx ON job_assignments (job_id) WHERE cancelled_at IS NULL;

CREATE TABLE job_status_history (
    id         BIGSERIAL PRIMARY KEY,
    job_id     UUID NOT NULL REFERENCES jobs(id) ON DELETE CASCADE,
    status     TEXT NOT NULL,
    actor_id   UUID REFERENCES users(id),
    note       TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE cancellations (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    job_id            UUID NOT NULL REFERENCES jobs(id) ON DELETE CASCADE,
    user_id           UUID REFERENCES users(id),
    role              TEXT NOT NULL CHECK (role IN ('client', 'worker', 'admin')),
    reason            TEXT NOT NULL DEFAULT '',
    status_at_cancel  TEXT NOT NULL,
    reputation_impact BOOLEAN NOT NULL DEFAULT FALSE,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE conversations (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    job_id     UUID NOT NULL REFERENCES jobs(id) ON DELETE CASCADE,
    client_id  UUID NOT NULL REFERENCES users(id),
    worker_id  UUID NOT NULL REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    -- One conversation per (job, worker): a job re-assigned after a worker
    -- cancels gets a fresh conversation.
    UNIQUE (job_id, worker_id)
);

CREATE TABLE messages (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    conversation_id UUID NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
    sender_id       UUID REFERENCES users(id),     -- NULL = system message
    kind            TEXT NOT NULL DEFAULT 'text' CHECK (kind IN ('text', 'image', 'location', 'system')),
    body            TEXT NOT NULL DEFAULT '',
    attachment_url  TEXT NOT NULL DEFAULT '',
    latitude        DOUBLE PRECISION,
    longitude       DOUBLE PRECISION,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX messages_conversation_idx ON messages (conversation_id, created_at);

-- Escrow: funds are held at assignment and released after client confirmation.
CREATE TABLE payments (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    job_id        UUID NOT NULL REFERENCES jobs(id) ON DELETE CASCADE,
    payer_id      UUID NOT NULL REFERENCES users(id),
    worker_id     UUID NOT NULL REFERENCES users(id),
    amount        INTEGER NOT NULL,
    platform_fee  INTEGER NOT NULL,
    worker_amount INTEGER NOT NULL,
    refunded_amount INTEGER NOT NULL DEFAULT 0,
    commission_rate NUMERIC(5,4) NOT NULL,
    provider      TEXT NOT NULL,
    provider_ref  TEXT NOT NULL DEFAULT '',
    status        TEXT NOT NULL CHECK (status IN ('held', 'released', 'refunded', 'partially_refunded')),
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);
-- A refunded payment (e.g. the worker cancelled) frees the job for a new hold.
CREATE UNIQUE INDEX payments_live_idx ON payments (job_id) WHERE status <> 'refunded';

CREATE TABLE wallet_transactions (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID NOT NULL REFERENCES users(id),
    amount      INTEGER NOT NULL,         -- signed, FCFA
    kind        TEXT NOT NULL CHECK (kind IN ('earning', 'withdrawal', 'withdrawal_reversal', 'adjustment')),
    job_id      UUID REFERENCES jobs(id),
    description TEXT NOT NULL DEFAULT '',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX wallet_transactions_user_idx ON wallet_transactions (user_id, created_at);

CREATE TABLE withdrawals (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID NOT NULL REFERENCES users(id),
    amount      INTEGER NOT NULL CHECK (amount > 0),
    method      TEXT NOT NULL CHECK (method IN ('mobile_money', 'bank')),
    destination TEXT NOT NULL,
    status      TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'paid', 'rejected')),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    processed_at TIMESTAMPTZ
);

CREATE TABLE reviews (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    job_id        UUID NOT NULL REFERENCES jobs(id) ON DELETE CASCADE,
    reviewer_id   UUID NOT NULL REFERENCES users(id),
    reviewee_id   UUID NOT NULL REFERENCES users(id),
    reviewer_role TEXT NOT NULL CHECK (reviewer_role IN ('client', 'worker')),
    rating        SMALLINT NOT NULL CHECK (rating BETWEEN 1 AND 5),
    punctuality   SMALLINT CHECK (punctuality BETWEEN 1 AND 5),
    communication SMALLINT CHECK (communication BETWEEN 1 AND 5),
    quality       SMALLINT CHECK (quality BETWEEN 1 AND 5),
    respect       SMALLINT CHECK (respect BETWEEN 1 AND 5),
    payment       SMALLINT CHECK (payment BETWEEN 1 AND 5),
    comment       TEXT NOT NULL DEFAULT '',
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (job_id, reviewer_id)
);
CREATE INDEX reviews_reviewee_idx ON reviews (reviewee_id);

CREATE TABLE reports (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    job_id          UUID NOT NULL REFERENCES jobs(id) ON DELETE CASCADE,
    reporter_id     UUID NOT NULL REFERENCES users(id),
    type            TEXT NOT NULL CHECK (type IN ('not_done', 'bad_quality', 'payment', 'behavior', 'fraud', 'other')),
    description     TEXT NOT NULL DEFAULT '',
    status          TEXT NOT NULL DEFAULT 'open' CHECK (status IN ('open', 'resolved', 'rejected')),
    status_before   TEXT NOT NULL,
    resolution      TEXT,
    refund_amount   INTEGER,
    resolution_note TEXT NOT NULL DEFAULT '',
    resolved_by     UUID REFERENCES users(id),
    resolved_at     TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE notifications (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    kind       TEXT NOT NULL,
    title      TEXT NOT NULL,
    body       TEXT NOT NULL DEFAULT '',
    job_id     UUID REFERENCES jobs(id) ON DELETE CASCADE,
    read_at    TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX notifications_user_idx ON notifications (user_id, created_at DESC);

-- MVP categories (cahier des charges §35).
INSERT INTO categories (slug, name, icon, sort_order) VALUES
    ('tech', 'Tech', '💻', 1),
    ('design', 'Design', '🎨', 2),
    ('services', 'Services', '📦', 3),
    ('maison', 'Maison', '🏠', 4),
    ('etudes', 'Études', '🎓', 5);

INSERT INTO skills (category_id, name)
SELECT c.id, s.name FROM categories c
JOIN (VALUES
    ('tech', 'Installation Windows'),
    ('tech', 'Réparation PC'),
    ('tech', 'Configuration Wi-Fi'),
    ('tech', 'Installation logiciels'),
    ('design', 'Flyer'),
    ('design', 'Logo'),
    ('design', 'Affiche'),
    ('design', 'Retouche photo'),
    ('services', 'Livraison'),
    ('services', 'Courses'),
    ('services', 'Déplacement d''objet'),
    ('maison', 'Nettoyage'),
    ('maison', 'Montage'),
    ('maison', 'Petites réparations'),
    ('etudes', 'Tutorat'),
    ('etudes', 'Programmation'),
    ('etudes', 'Mathématiques'),
    ('etudes', 'Langues')
) AS s(slug, name) ON s.slug = c.slug;
