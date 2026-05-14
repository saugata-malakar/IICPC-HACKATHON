# ─────────────────────────────────────────────────────────────────────────────
# IICPC Platform v2 — Windows PowerShell Startup Script
# ─────────────────────────────────────────────────────────────────────────────

Write-Host "🚀 Starting IICPC Platform v2..." -ForegroundColor Cyan
Write-Host ""

# Check Docker
Write-Host "Checking Docker..." -ForegroundColor Yellow
try {
    $dockerVersion = docker --version
    Write-Host "✓ Docker found: $dockerVersion" -ForegroundColor Green
} catch {
    Write-Host "❌ Docker not found. Please install Docker Desktop for Windows." -ForegroundColor Red
    Write-Host "   Download from: https://www.docker.com/products/docker-desktop" -ForegroundColor Yellow
    exit 1
}

# Check Docker Compose
Write-Host "Checking Docker Compose..." -ForegroundColor Yellow
try {
    $composeVersion = docker compose version
    Write-Host "✓ Docker Compose found: $composeVersion" -ForegroundColor Green
} catch {
    Write-Host "❌ Docker Compose not found. Please update Docker Desktop." -ForegroundColor Red
    exit 1
}

# Check if Docker is running
Write-Host "Checking if Docker is running..." -ForegroundColor Yellow
try {
    docker ps | Out-Null
    Write-Host "✓ Docker is running" -ForegroundColor Green
} catch {
    Write-Host "❌ Docker is not running. Please start Docker Desktop." -ForegroundColor Red
    exit 1
}

Write-Host ""
Write-Host "Starting all services..." -ForegroundColor Cyan
Write-Host "This will take ~60 seconds for first-time setup..." -ForegroundColor Yellow
Write-Host ""

# Start services
docker compose up -d --build

Write-Host ""
Write-Host "Waiting for services to be healthy..." -ForegroundColor Yellow
Start-Sleep -Seconds 30

Write-Host ""
Write-Host "═══════════════════════════════════════════════════════════════" -ForegroundColor Cyan
Write-Host "  ✓ IICPC Platform v2 is starting!" -ForegroundColor Green
Write-Host "═══════════════════════════════════════════════════════════════" -ForegroundColor Cyan
Write-Host ""
Write-Host "  📱 Frontend          → http://localhost:3000" -ForegroundColor White
Write-Host "  🔌 API Gateway       → http://localhost:8080" -ForegroundColor White
Write-Host "  📊 Leaderboard WS    → ws://localhost:8003/ws" -ForegroundColor White
Write-Host "  🤖 Bot Fleet         → http://localhost:8002" -ForegroundColor White
Write-Host "  📈 Predictor API     → http://localhost:8006" -ForegroundColor White
Write-Host "  🎛️  Kafka UI          → http://localhost:8090" -ForegroundColor White
Write-Host "  📉 Grafana           → http://localhost:3001  (admin/iicpc2026)" -ForegroundColor White
Write-Host ""
Write-Host "═══════════════════════════════════════════════════════════════" -ForegroundColor Cyan
Write-Host ""
Write-Host "⏳ Services are still starting up. Wait 30 more seconds..." -ForegroundColor Yellow
Write-Host ""
Write-Host "To view logs:        docker compose logs -f" -ForegroundColor Gray
Write-Host "To stop platform:    docker compose down" -ForegroundColor Gray
Write-Host "To restart:          docker compose restart" -ForegroundColor Gray
Write-Host ""

# Open browser
Write-Host "Opening browser..." -ForegroundColor Cyan
Start-Sleep -Seconds 5
Start-Process "http://localhost:3000"

Write-Host ""
Write-Host "✨ Platform is ready! Check your browser." -ForegroundColor Green
Write-Host ""
