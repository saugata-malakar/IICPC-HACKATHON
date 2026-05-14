-- ─────────────────────────────────────────────────────────────────────────────
-- IICPC Platform v2 — PostgreSQL Schema (Main Database)
-- ─────────────────────────────────────────────────────────────────────────────

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- ─── Users & Teams ────────────────────────────────────────────────────────────

CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    team_name VARCHAR(100) NOT NULL,
    role VARCHAR(20) DEFAULT 'contestant' CHECK (role IN ('contestant', 'admin')),
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_team_name ON users(team_name);

-- ─── Submissions ──────────────────────────────────────────────────────────────

CREATE TABLE submissions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    team_name VARCHAR(100) NOT NULL,
    filename VARCHAR(255) NOT NULL,
    file_size_bytes BIGINT NOT NULL,
    file_hash VARCHAR(64) NOT NULL,
    sandbox_type VARCHAR(20) DEFAULT 'docker' CHECK (sandbox_type IN ('docker', 'gvisor', 'wasm')),
    status VARCHAR(20) DEFAULT 'pending' CHECK (status IN ('pending', 'scanning', 'building', 'deploying', 'running', 'completed', 'failed')),
    container_id VARCHAR(100),
    endpoint VARCHAR(255),
    protocol VARCHAR(20) DEFAULT 'rest' CHECK (protocol IN ('rest', 'websocket', 'fix')),
    cpu_limit DECIMAL(4,2) DEFAULT 2.0,
    memory_limit_mb INT DEFAULT 512,
    created_at TIMESTAMP DEFAULT NOW(),
    deployed_at TIMESTAMP,
    completed_at TIMESTAMP,
    error_message TEXT
);

CREATE INDEX idx_submissions_user_id ON submissions(user_id);
CREATE INDEX idx_submissions_status ON submissions(status);
CREATE INDEX idx_submissions_created_at ON submissions(created_at DESC);

-- ─── Bot Runs ─────────────────────────────────────────────────────────────────

CREATE TABLE bot_runs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    submission_id UUID NOT NULL REFERENCES submissions(id) ON DELETE CASCADE,
    bot_count INT NOT NULL,
    duration_seconds INT NOT NULL,
    status VARCHAR(20) DEFAULT 'pending' CHECK (status IN ('pending', 'running', 'completed', 'failed')),
    orders_sent BIGINT DEFAULT 0,
    fills_received BIGINT DEFAULT 0,
    errors BIGINT DEFAULT 0,
    started_at TIMESTAMP,
    completed_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_bot_runs_submission_id ON bot_runs(submission_id);
CREATE INDEX idx_bot_runs_status ON bot_runs(status);

-- ─── Chaos Experiments ────────────────────────────────────────────────────────

CREATE TABLE chaos_experiments (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    submission_id UUID NOT NULL REFERENCES submissions(id) ON DELETE CASCADE,
    fault_type VARCHAR(50) NOT NULL CHECK (fault_type IN ('latency', 'packet_loss', 'cpu_stress', 'memory_pressure', 'network_partition', 'process_freeze')),
    severity INT NOT NULL CHECK (severity BETWEEN 1 AND 10),
    duration_seconds INT NOT NULL,
    status VARCHAR(20) DEFAULT 'pending' CHECK (status IN ('pending', 'injecting', 'active', 'recovering', 'completed', 'failed')),
    p99_before_ms DECIMAL(10,3),
    p99_during_ms DECIMAL(10,3),
    p99_after_ms DECIMAL(10,3),
    recovery_time_ms BIGINT,
    started_at TIMESTAMP,
    completed_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_chaos_submission_id ON chaos_experiments(submission_id);
CREATE INDEX idx_chaos_status ON chaos_experiments(status);

-- ─── Leaderboard Snapshots ────────────────────────────────────────────────────

CREATE TABLE leaderboard_snapshots (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    submission_id UUID NOT NULL REFERENCES submissions(id) ON DELETE CASCADE,
    rank INT NOT NULL,
    score DECIMAL(10,6) NOT NULL,
    p50_latency_ms DECIMAL(10,3),
    p90_latency_ms DECIMAL(10,3),
    p99_latency_ms DECIMAL(10,3),
    tps BIGINT,
    correctness DECIMAL(5,4),
    chaos_bonus DECIMAL(5,4) DEFAULT 0,
    snapshot_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_leaderboard_submission_id ON leaderboard_snapshots(submission_id);
CREATE INDEX idx_leaderboard_snapshot_at ON leaderboard_snapshots(snapshot_at DESC);
CREATE INDEX idx_leaderboard_rank ON leaderboard_snapshots(rank);

-- ─── Audit Log ────────────────────────────────────────────────────────────────

CREATE TABLE audit_log (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    action VARCHAR(100) NOT NULL,
    resource_type VARCHAR(50),
    resource_id UUID,
    ip_address INET,
    user_agent TEXT,
    metadata JSONB,
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_audit_user_id ON audit_log(user_id);
CREATE INDEX idx_audit_created_at ON audit_log(created_at DESC);
CREATE INDEX idx_audit_action ON audit_log(action);

-- ─── Functions ────────────────────────────────────────────────────────────────

-- Update updated_at timestamp
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER update_users_updated_at
    BEFORE UPDATE ON users
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- ─── Seed Data ────────────────────────────────────────────────────────────────

-- Admin user (password: admin2026)
INSERT INTO users (email, password_hash, team_name, role)
VALUES (
    'admin@iicpc.org',
    crypt('admin2026', gen_salt('bf')),
    'IICPC Admin',
    'admin'
);

-- Sample contestant (password: contestant2026)
INSERT INTO users (email, password_hash, team_name, role)
VALUES (
    'team1@example.com',
    crypt('contestant2026', gen_salt('bf')),
    'Team Alpha',
    'contestant'
);

COMMENT ON TABLE users IS 'Platform users and teams';
COMMENT ON TABLE submissions IS 'Contestant code submissions';
COMMENT ON TABLE bot_runs IS 'Bot fleet execution records';
COMMENT ON TABLE chaos_experiments IS 'Chaos engineering experiments';
COMMENT ON TABLE leaderboard_snapshots IS 'Historical leaderboard data';
COMMENT ON TABLE audit_log IS 'Security and compliance audit trail';
