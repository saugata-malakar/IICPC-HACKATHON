.PHONY: up down build push test proto smoke logs clean k8s tf

REGISTRY ?= ghcr.io/iicpc
TAG      ?= latest
SERVICES := api-gateway submission-service bot-fleet telemetry-ingester leaderboard-service chaos-engine predictor

# ─── One-command dev stack ────────────────────────────────────────────────────
up:
	@echo "▶  Booting IICPC Platform v2..."
	docker compose up --build -d
	@echo ""
	@echo "  ✓ Frontend          → http://localhost:3000"
	@echo "  ✓ API Gateway       → http://localhost:8080"
	@echo "  ✓ Leaderboard WS    → ws://localhost:8003/ws"
	@echo "  ✓ Predictor API     → http://localhost:8006"
	@echo "  ✓ Kafka UI          → http://localhost:8090"
	@echo "  ✓ Grafana           → http://localhost:3001  (admin/iicpc2026)"
	@echo ""

down:
	docker compose down -v --remove-orphans

logs:
	docker compose logs -f --tail=60

# ─── Build ────────────────────────────────────────────────────────────────────
build:
	@for s in $(SERVICES); do \
		echo "▶  Building $$s ..."; \
		docker build -t $(REGISTRY)/$$s:$(TAG) ./services/$$s; \
	done
	docker build -t $(REGISTRY)/frontend:$(TAG) ./frontend

push: build
	@for s in $(SERVICES); do docker push $(REGISTRY)/$$s:$(TAG); done
	docker push $(REGISTRY)/frontend:$(TAG)

# ─── Tests ────────────────────────────────────────────────────────────────────
test-go:
	@for s in api-gateway bot-fleet leaderboard-service submission-service chaos-engine; do \
		cd services/$$s && go test ./... -race -timeout 60s && cd ../..; \
	done

test-rust:
	cd services/telemetry-ingester && cargo test --release

test-python:
	cd services/predictor && python -m pytest tests/ -v

test: test-go test-rust test-python

# ─── gRPC proto ───────────────────────────────────────────────────────────────
proto:
	protoc \
		--go_out=./services/api-gateway/proto/bot \
		--go_opt=paths=source_relative \
		--go-grpc_out=./services/api-gateway/proto/bot \
		--go-grpc_opt=paths=source_relative \
		./services/api-gateway/proto/bot/bot_fleet.proto
	@echo "✓ Proto generated"

# ─── eBPF ─────────────────────────────────────────────────────────────────────
ebpf:
	cd services/telemetry-ingester && \
		go generate ./... && \
		echo "✓ eBPF objects generated"

# ─── Smoke test ───────────────────────────────────────────────────────────────
smoke:
	@echo "▶  Health checks..."
	@curl -sf http://localhost:8080/health   && echo "  ✓ API Gateway"
	@curl -sf http://localhost:8001/health   && echo "  ✓ Submission Service"
	@curl -sf http://localhost:8002/health   && echo "  ✓ Bot Fleet"
	@curl -sf http://localhost:8003/health   && echo "  ✓ Leaderboard"
	@curl -sf http://localhost:8004/health   && echo "  ✓ Telemetry Ingester"
	@curl -sf http://localhost:8005/health   && echo "  ✓ Chaos Engine"
	@curl -sf http://localhost:8006/health   && echo "  ✓ Predictor"

# ─── Quick load test (manual) ─────────────────────────────────────────────────
load-test:
	curl -X POST http://localhost:8080/api/v1/bots/spawn \
		-H "Content-Type: application/json" \
		-H "Authorization: Bearer $(TOKEN)" \
		-d '{"submission_id":"$(SID)","endpoint":"$(EP)","protocol":"rest","bot_count":100,"duration_seconds":60}'

# ─── K8s ─────────────────────────────────────────────────────────────────────
k8s:
	kubectl apply -f infrastructure/kubernetes/platform.yaml

k8s-status:
	kubectl -n iicpc-platform get pods,svc,hpa -o wide

# ─── Terraform ───────────────────────────────────────────────────────────────
tf-init:
	cd infrastructure/terraform && terraform init

tf-plan:
	cd infrastructure/terraform && terraform plan -out=tfplan

tf:
	cd infrastructure/terraform && terraform apply tfplan

# ─── Database ─────────────────────────────────────────────────────────────────
db:
	docker compose exec timescaledb psql -U telemetry -d telemetry

db-pg:
	docker compose exec postgres psql -U platform -d iicpc

kafka-topics:
	docker compose exec kafka kafka-topics \
		--bootstrap-server localhost:9092 --list

# ─── Clean ────────────────────────────────────────────────────────────────────
clean:
	docker compose down -v --remove-orphans
	docker image prune -f
	find . -name "*.pb.go" -delete
	cd services/telemetry-ingester && cargo clean
	cd frontend && rm -rf .next node_modules
