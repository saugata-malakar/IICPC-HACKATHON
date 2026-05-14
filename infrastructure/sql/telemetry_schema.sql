-- ─────────────────────────────────────────────────────────────────────────────
-- IICPC Platform v2 — TimescaleDB Schema (Telemetry Database)
-- ─────────────────────────────────────────────────────────────────────────────

CREATE EXTENSION IF NOT EXISTS timescaledb;

-- ─── Raw Order Events ─────────────────────────────────────────────────────────

CREATE TABLE order_events (
    time TIMESTAMPTZ NOT NULL,
    submission_id UUID NOT NULL,
    order_id VARCHAR(100) NOT NULL,
    client_id VARCHAR(50),
    symbol VARCHAR(20),
    side VARCHAR(10),
    order_type VARCHAR(20),
    price DECIMAL(20,8),
    quantity BIGINT,
    bot_persona VARCHAR(50),
    latency_ns BIGINT,
    status VARCHAR(20)
);

SELECT create_hypertable('order_events', 'time', chunk_time_interval => INTERVAL '1 minute');

CREATE INDEX idx_order_events_submission_id ON order_events (submission_id, time DESC);
CREATE INDEX idx_order_events_order_id ON order_events (order_id);

-- ─── Fill Events ──────────────────────────────────────────────────────────────

CREATE TABLE fill_events (
    time TIMESTAMPTZ NOT NULL,
    submission_id UUID NOT NULL,
    order_id VARCHAR(100) NOT NULL,
    fill_id VARCHAR(100) NOT NULL,
    fill_price DECIMAL(20,8),
    fill_quantity BIGINT,
    latency_ns BIGINT,
    is_anomaly BOOLEAN DEFAULT FALSE
);

SELECT create_hypertable('fill_events', 'time', chunk_time_interval => INTERVAL '1 minute');

CREATE INDEX idx_fill_events_submission_id ON fill_events (submission_id, time DESC);
CREATE INDEX idx_fill_events_order_id ON fill_events (order_id);

-- ─── Latency Measurements ─────────────────────────────────────────────────────

CREATE TABLE latency_measurements (
    time TIMESTAMPTZ NOT NULL,
    submission_id UUID NOT NULL,
    measurement_type VARCHAR(50), -- 'order_ack', 'fill', 'cancel_ack'
    latency_ns BIGINT NOT NULL,
    p50_ns BIGINT,
    p90_ns BIGINT,
    p99_ns BIGINT,
    p999_ns BIGINT,
    max_ns BIGINT,
    count BIGINT
);

SELECT create_hypertable('latency_measurements', 'time', chunk_time_interval => INTERVAL '5 seconds');

CREATE INDEX idx_latency_submission_id ON latency_measurements (submission_id, time DESC);

-- ─── Throughput Metrics ───────────────────────────────────────────────────────

CREATE TABLE throughput_metrics (
    time TIMESTAMPTZ NOT NULL,
    submission_id UUID NOT NULL,
    orders_per_sec BIGINT,
    fills_per_sec BIGINT,
    cancels_per_sec BIGINT,
    total_tps BIGINT,
    error_rate DECIMAL(5,4)
);

SELECT create_hypertable('throughput_metrics', 'time', chunk_time_interval => INTERVAL '5 seconds');

CREATE INDEX idx_throughput_submission_id ON throughput_metrics (submission_id, time DESC);

-- ─── Correctness Metrics ──────────────────────────────────────────────────────

CREATE TABLE correctness_metrics (
    time TIMESTAMPTZ NOT NULL,
    submission_id UUID NOT NULL,
    total_fills BIGINT,
    price_time_violations BIGINT,
    price_deviation_sum DECIMAL(20,8),
    side_imbalance DECIMAL(10,6),
    fill_accuracy DECIMAL(5,4)
);

SELECT create_hypertable('correctness_metrics', 'time', chunk_time_interval => INTERVAL '5 seconds');

CREATE INDEX idx_correctness_submission_id ON correctness_metrics (submission_id, time DESC);

-- ─── Chaos Metrics ────────────────────────────────────────────────────────────

CREATE TABLE chaos_metrics (
    time TIMESTAMPTZ NOT NULL,
    submission_id UUID NOT NULL,
    experiment_id UUID NOT NULL,
    fault_type VARCHAR(50),
    severity INT,
    p99_latency_ms DECIMAL(10,3),
    tps BIGINT,
    error_rate DECIMAL(5,4),
    recovery_time_ms BIGINT
);

SELECT create_hypertable('chaos_metrics', 'time', chunk_time_interval => INTERVAL '1 minute');

CREATE INDEX idx_chaos_metrics_submission_id ON chaos_metrics (submission_id, time DESC);
CREATE INDEX idx_chaos_metrics_experiment_id ON chaos_metrics (experiment_id);

-- ─── Continuous Aggregates (5-second rollups) ─────────────────────────────────

CREATE MATERIALIZED VIEW latency_5s
WITH (timescaledb.continuous) AS
SELECT
    time_bucket('5 seconds', time) AS bucket,
    submission_id,
    percentile_agg(latency_ns) AS latency_agg
FROM order_events
GROUP BY bucket, submission_id
WITH NO DATA;

SELECT add_continuous_aggregate_policy('latency_5s',
    start_offset => INTERVAL '1 hour',
    end_offset => INTERVAL '5 seconds',
    schedule_interval => INTERVAL '5 seconds');

-- ─── Continuous Aggregates (1-minute rollups) ─────────────────────────────────

CREATE MATERIALIZED VIEW latency_1m
WITH (timescaledb.continuous) AS
SELECT
    time_bucket('1 minute', time) AS bucket,
    submission_id,
    percentile_agg(latency_ns) AS latency_agg,
    COUNT(*) AS total_orders
FROM order_events
GROUP BY bucket, submission_id
WITH NO DATA;

SELECT add_continuous_aggregate_policy('latency_1m',
    start_offset => INTERVAL '1 day',
    end_offset => INTERVAL '1 minute',
    schedule_interval => INTERVAL '1 minute');

-- ─── Retention Policies ───────────────────────────────────────────────────────

-- Keep raw data for 24 hours
SELECT add_retention_policy('order_events', INTERVAL '24 hours');
SELECT add_retention_policy('fill_events', INTERVAL '24 hours');

-- Keep 5-second aggregates for 7 days
SELECT add_retention_policy('latency_5s', INTERVAL '7 days');

-- Keep 1-minute aggregates for 30 days
SELECT add_retention_policy('latency_1m', INTERVAL '30 days');

-- ─── Compression Policies ─────────────────────────────────────────────────────

ALTER TABLE order_events SET (
    timescaledb.compress,
    timescaledb.compress_segmentby = 'submission_id',
    timescaledb.compress_orderby = 'time DESC'
);

SELECT add_compression_policy('order_events', INTERVAL '1 hour');

ALTER TABLE fill_events SET (
    timescaledb.compress,
    timescaledb.compress_segmentby = 'submission_id',
    timescaledb.compress_orderby = 'time DESC'
);

SELECT add_compression_policy('fill_events', INTERVAL '1 hour');

-- ─── Helper Functions ─────────────────────────────────────────────────────────

-- Get latest metrics for a submission
CREATE OR REPLACE FUNCTION get_latest_metrics(sub_id UUID)
RETURNS TABLE (
    p50_latency_ms DECIMAL,
    p90_latency_ms DECIMAL,
    p99_latency_ms DECIMAL,
    tps BIGINT,
    correctness DECIMAL
) AS $$
BEGIN
    RETURN QUERY
    SELECT
        (percentile_approx(latency_ns, 0.50) / 1000000.0)::DECIMAL AS p50_latency_ms,
        (percentile_approx(latency_ns, 0.90) / 1000000.0)::DECIMAL AS p90_latency_ms,
        (percentile_approx(latency_ns, 0.99) / 1000000.0)::DECIMAL AS p99_latency_ms,
        COUNT(*)::BIGINT AS tps,
        0.95::DECIMAL AS correctness
    FROM order_events
    WHERE submission_id = sub_id
        AND time > NOW() - INTERVAL '1 minute';
END;
$$ LANGUAGE plpgsql;

-- Calculate composite score
CREATE OR REPLACE FUNCTION calculate_score(
    p99_ms DECIMAL,
    tps_val BIGINT,
    correctness_val DECIMAL,
    chaos_bonus_val DECIMAL DEFAULT 0
)
RETURNS DECIMAL AS $$
DECLARE
    lat_score DECIMAL;
    tps_score DECIMAL;
    composite DECIMAL;
BEGIN
    -- Latency score: 1.0 at p99=100μs, 0.0 at p99=100ms
    lat_score := GREATEST(0, 1 - LN(p99_ms / 0.1) / LN(1000));
    
    -- TPS score: logarithmic; 1.0 at 1M TPS
    tps_score := LEAST(1, LN(tps_val) / LN(1000000));
    
    -- Composite
    composite := (0.40 * lat_score + 0.35 * tps_score + 0.25 * correctness_val) * (1 + chaos_bonus_val);
    
    RETURN composite;
END;
$$ LANGUAGE plpgsql;

COMMENT ON TABLE order_events IS 'Raw order events from bot fleet';
COMMENT ON TABLE fill_events IS 'Fill confirmations from contestant exchanges';
COMMENT ON TABLE latency_measurements IS 'Aggregated latency percentiles';
COMMENT ON TABLE throughput_metrics IS 'TPS and throughput measurements';
COMMENT ON TABLE correctness_metrics IS 'Order book correctness validation';
COMMENT ON TABLE chaos_metrics IS 'Chaos experiment telemetry';
