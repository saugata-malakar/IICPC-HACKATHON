# Getting Started with IICPC Platform v2

## 🚀 Quick Start (5 minutes)

### Prerequisites Check

```bash
# Check Docker
docker --version
# Required: Docker 26.0+

# Check Docker Compose
docker compose version
# Required: Compose V2

# Check available memory
docker info | grep "Total Memory"
# Recommended: 16GB+ RAM
```

### Start the Platform

```bash
# 1. Navigate to project directory
cd iicpc-championship-v2/iicpc-final

# 2. Start all services (one command!)
make up

# 3. Wait for services to be healthy (~60 seconds)
# You'll see output like:
#   ✓ Frontend          → http://localhost:3000
#   ✓ API Gateway       → http://localhost:8080
#   ✓ Leaderboard WS    → ws://localhost:8003/ws
#   ✓ Predictor API     → http://localhost:8006
#   ✓ Kafka UI          → http://localhost:8090
#   ✓ Grafana           → http://localhost:3001  (admin/iicpc2026)

# 4. Verify all services are healthy
make smoke
```

### Access the Platform

Open your browser and visit:

1. **Frontend Dashboard:** http://localhost:3000
   - Home page with countdown timer
   - Live leaderboard
   - Code submission interface
   - Analytics dashboard

2. **Kafka UI:** http://localhost:8090
   - View topics, partitions, messages
   - Monitor consumer groups

3. **Grafana:** http://localhost:3001
   - Username: `admin`
   - Password: `iicpc2026`
   - Pre-configured dashboards

---

## 📝 Your First Submission

### Step 1: Create a Sample Trading Engine

Create a simple REST API that accepts orders:

```python
# sample_exchange.py
from flask import Flask, request, jsonify
import time

app = Flask(__name__)
orders = {}

@app.route('/health', methods=['GET'])
def health():
    return jsonify({"status": "healthy"})

@app.route('/order', methods=['POST'])
def create_order():
    order = request.json
    order_id = order.get('order_id')
    orders[order_id] = {
        **order,
        'status': 'filled',
        'fill_time': time.time_ns()
    }
    return jsonify(orders[order_id]), 201

if __name__ == '__main__':
    app.run(host='0.0.0.0', port=8888)
```

### Step 2: Package as Docker Image

```dockerfile
# Dockerfile
FROM python:3.11-slim
WORKDIR /app
RUN pip install flask
COPY sample_exchange.py .
EXPOSE 8888
CMD ["python", "sample_exchange.py"]
```

Build and save:

```bash
docker build -t my-exchange:latest .
docker save my-exchange:latest | gzip > my-exchange.tar.gz
```

### Step 3: Submit via Web Interface

1. Go to http://localhost:3000/submit
2. Drag and drop `my-exchange.tar.gz`
3. Click "Submit to Platform"
4. Watch the deployment pipeline:
   - ✅ Uploading
   - ✅ Security scanning
   - ✅ Building container
   - ✅ Deploying to sandbox
   - ✅ Completed!

### Step 4: View Results

1. Go to http://localhost:3000/leaderboard
2. See your submission appear in real-time
3. Watch metrics update live:
   - p99 latency
   - TPS (transactions per second)
   - Correctness score
   - Rank

---

## 🔧 Development Workflow

### View Logs

```bash
# All services
make logs

# Specific service
docker compose logs -f api-gateway
docker compose logs -f bot-fleet
docker compose logs -f telemetry-ingester
```

### Restart a Service

```bash
# Restart specific service
docker compose restart api-gateway

# Rebuild and restart
docker compose up -d --build api-gateway
```

### Access Databases

```bash
# PostgreSQL (main database)
make db-pg
# Then: \dt to list tables, \d users to describe table

# TimescaleDB (telemetry)
make db
# Then: SELECT * FROM order_events LIMIT 10;

# Redis
docker compose exec redis redis-cli
# Then: KEYS *, ZRANGE leaderboard:live 0 -1 WITHSCORES
```

### View Kafka Topics

```bash
make kafka-topics

# Or use Kafka UI: http://localhost:8090
```

---

## 🧪 Testing

### Run All Tests

```bash
make test
```

### Run Specific Test Suites

```bash
# Go services
make test-go

# Rust telemetry ingester
make test-rust

# Python predictor
make test-python
```

### Manual Load Test

```bash
# First, get a JWT token by logging in
TOKEN="your-jwt-token"

# Get your submission ID from the leaderboard
SID="your-submission-id"

# Run load test
make load-test TOKEN=$TOKEN SID=$SID EP="http://your-submission:8888"
```

---

## 🐛 Troubleshooting

### Services Won't Start

```bash
# Check Docker resources
docker system df

# Clean up old containers/volumes
make clean

# Restart from scratch
make down
make up
```

### Port Already in Use

```bash
# Find process using port 3000 (example)
lsof -i :3000

# Kill process
kill -9 <PID>
```

### Out of Memory

```bash
# Check Docker memory limit
docker info | grep Memory

# Increase Docker memory:
# Docker Desktop → Settings → Resources → Memory → 16GB
```

### Kafka Connection Issues

```bash
# Check Kafka health
docker compose exec kafka kafka-broker-api-versions --bootstrap-server localhost:9092

# Restart Kafka
docker compose restart kafka zookeeper
```

### Database Connection Issues

```bash
# Check PostgreSQL
docker compose exec postgres pg_isready -U platform

# Check TimescaleDB
docker compose exec timescaledb pg_isready -U telemetry

# Restart databases
docker compose restart postgres timescaledb
```

---

## 📚 Next Steps

### 1. Explore the Architecture

Read `ARCHITECTURE.md` to understand:
- 5 groundbreaking concepts
- System architecture
- Scoring formula
- Tech stack details

### 2. Study the Code

Key files to explore:
- `services/bot-fleet/disruptor/ring_buffer.go` — LMAX Disruptor
- `services/chaos-engine/chaos.go` — Fault injection
- `services/predictor/predictor.py` — ML models
- `services/telemetry-ingester/src/main.rs` — Rust ingester
- `frontend/app/leaderboard/page.tsx` — Real-time UI

### 3. Customize the Platform

- Add new bot personas in `services/bot-fleet/main.go`
- Add new chaos fault types in `services/chaos-engine/chaos.go`
- Customize scoring weights in `services/leaderboard-service/main.go`
- Add new ML models in `services/predictor/predictor.py`

### 4. Deploy to Cloud

```bash
# AWS deployment
make tf-init
make tf-plan
make tf

# Kubernetes deployment
make k8s
make k8s-status
```

---

## 🎯 Key Concepts

### Bot Fleet

The bot fleet simulates thousands of concurrent traders:
- **6 personas:** Market maker, Arbitrageur, Momentum, Mean revert, Noise trader, HFT
- **Poisson arrival:** Realistic inter-arrival times
- **GBM pricing:** Geometric Brownian Motion for price simulation
- **Disruptor ring buffer:** 50M events/sec throughput

### Chaos Engine

Adversarial fault injection:
- **6 fault types:** Latency, packet loss, CPU stress, memory pressure, network partition, process freeze
- **Adaptive severity:** Escalates based on p99 performance
- **MTTR measurement:** Mean time to recovery as scoring dimension

### ML Predictor

Three models running continuously:
- **Kalman filter:** Predicts next 30s of p99 latency
- **M/D/1 queueing:** Predicts saturation TPS
- **Isolation Forest:** Detects anomalous fills

### Scoring

Composite score formula:
```
score = (0.40 × latency + 0.35 × tps + 0.25 × correctness) × (1 + chaos_bonus)
```

---

## 💡 Tips & Best Practices

### For Contestants

1. **Optimize for p99 latency** — It has the highest weight (40%)
2. **Handle chaos gracefully** — +50% bonus for resilience
3. **Maintain correctness** — Price-time priority is validated
4. **Use efficient data structures** — Lock-free when possible
5. **Profile your code** — Use pprof, perf, or flamegraphs

### For Platform Operators

1. **Monitor Kafka lag** — Use Kafka UI to check consumer groups
2. **Watch TimescaleDB size** — 24h retention policy is active
3. **Scale bot fleet** — Use HPA to handle load
4. **Check eBPF probes** — Requires privileged mode in production
5. **Backup databases** — PostgreSQL and TimescaleDB snapshots

---

## 🆘 Getting Help

### Documentation

- **README.md** — Overview and quick start
- **ARCHITECTURE.md** — Detailed system design
- **SUBMISSION_SUMMARY.md** — Competition submission details
- **API docs** — Coming soon

### Community

- **Email:** team@iicpc.org
- **Discord:** IICPC 2026 Server
- **GitHub Issues:** https://github.com/iicpc/championship-platform-v2/issues

---

## ✅ Checklist

Before submitting your code:

- [ ] Code runs on port 8888
- [ ] Health endpoint at `/health`
- [ ] Accepts orders via REST/WebSocket/FIX
- [ ] Returns fills with timestamps
- [ ] Handles 1000+ concurrent connections
- [ ] Maintains price-time priority
- [ ] Survives chaos experiments
- [ ] Docker image < 50MB (recommended)

---

**Ready to compete? Start with `make up` and build something amazing! 🚀**
