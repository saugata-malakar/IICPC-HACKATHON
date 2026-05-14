# ─────────────────────────────────────────────────────────────────────────────
# IICPC Platform v2 — Windows PowerShell Stop Script
# ─────────────────────────────────────────────────────────────────────────────

Write-Host "🛑 Stopping IICPC Platform v2..." -ForegroundColor Yellow
Write-Host ""

docker compose down -v --remove-orphans

Write-Host ""
Write-Host "✓ Platform stopped and cleaned up." -ForegroundColor Green
Write-Host ""
