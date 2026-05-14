# 🪟 Windows Setup Guide for IICPC Platform v2

## Quick Start (Windows)

### Prerequisites

1. **Install Docker Desktop for Windows**
   - Download: https://www.docker.com/products/docker-desktop
   - Install and restart your computer
   - Open Docker Desktop and wait for it to start
   - Ensure WSL 2 is enabled (Docker Desktop will prompt you)

2. **Verify Installation**
   ```powershell
   docker --version
   docker compose version
   ```

### Start the Platform

**Option 1: Using PowerShell Script (Easiest)**

```powershell
# Navigate to project directory
cd iicpc-championship-v2\iicpc-final

# Run the startup script
.\START-WINDOWS.ps1
```

The script will:
- ✅ Check Docker installation
- ✅ Start all 12 services
- ✅ Wait for services to be healthy
- ✅ Open your browser automatically

**Option 2: Using Docker Compose Directly**

```powershell
# Start all services
docker compose up -d --build

# View logs
docker compose logs -f

# Check status
docker compose ps
```

### Access the Platform

After ~60 seconds, open your browser:

- **Frontend:** http://localhost:3000
- **API Gateway:** http://localhost:8080
- **Kafka UI:** http://localhost:8090
- **Grafana:** http://localhost:3001 (admin/iicpc2026)

### Stop the Platform

**Option 1: Using PowerShell Script**

```powershell
.\STOP-WINDOWS.ps1
```

**Option 2: Using Docker Compose**

```powershell
docker compose down
```

---

## Common Windows Issues

### Issue 1: "make: command not found"

**Solution:** Use the PowerShell scripts instead:
- `.\START-WINDOWS.ps1` instead of `make up`
- `.\STOP-WINDOWS.ps1` instead of `make down`

Or install `make` for Windows:
```powershell
# Using Chocolatey
choco install make

# Using Scoop
scoop install make
```

### Issue 2: Docker Desktop Not Running

**Error:** `error during connect: This error may indicate that the docker daemon is not running`

**Solution:**
1. Open Docker Desktop from Start Menu
2. Wait for the whale icon to stop animating
3. Try again

### Issue 3: Port Already in Use

**Error:** `Bind for 0.0.0.0:3000 failed: port is already allocated`

**Solution:**
```powershell
# Find process using port 3000
netstat -ano | findstr :3000

# Kill process (replace PID with actual process ID)
taskkill /PID <PID> /F
```

### Issue 4: WSL 2 Not Installed

**Error:** `WSL 2 installation is incomplete`

**Solution:**
1. Open PowerShell as Administrator
2. Run: `wsl --install`
3. Restart your computer
4. Open Docker Desktop again

### Issue 5: Out of Memory

**Error:** Services crash or don't start

**Solution:**
1. Open Docker Desktop
2. Go to Settings → Resources
3. Increase Memory to at least 8GB (16GB recommended)
4. Click "Apply & Restart"

### Issue 6: Line Ending Issues (CRLF vs LF)

**Error:** Shell scripts fail with `\r` errors

**Solution:**
```powershell
# Configure Git to use LF
git config --global core.autocrlf false

# Re-clone the repository
```

---

## Windows-Specific Commands

### View Logs

```powershell
# All services
docker compose logs -f

# Specific service
docker compose logs -f api-gateway
docker compose logs -f frontend
```

### Restart Services

```powershell
# Restart all
docker compose restart

# Restart specific service
docker compose restart api-gateway
```

### Check Service Status

```powershell
docker compose ps
```

### Access Databases

```powershell
# PostgreSQL
docker compose exec postgres psql -U platform -d iicpc

# TimescaleDB
docker compose exec timescaledb psql -U telemetry -d telemetry

# Redis
docker compose exec redis redis-cli
```

### Clean Up

```powershell
# Stop and remove everything
docker compose down -v --remove-orphans

# Remove all Docker images
docker system prune -a
```

---

## Development on Windows

### Using VS Code / Cursor

1. **Install Cursor** (recommended)
   - Download: https://cursor.sh/
   - Open the `iicpc-final` folder

2. **Install Extensions:**
   - Go
   - Rust Analyzer
   - Python
   - Docker
   - Kubernetes

3. **Open Integrated Terminal:**
   - Press `` Ctrl+` ``
   - Use PowerShell or WSL bash

### Using WSL 2 (Advanced)

For better performance, use WSL 2:

```powershell
# Install WSL 2
wsl --install

# Set WSL 2 as default
wsl --set-default-version 2

# Install Ubuntu
wsl --install -d Ubuntu

# Open Ubuntu terminal
wsl
```

Then inside WSL:
```bash
cd /mnt/c/Users/YourName/Downloads/PROJECTS/IICPC/iicpc-championship-v2/iicpc-final
make up
```

---

## PowerShell Aliases (Optional)

Add these to your PowerShell profile for convenience:

```powershell
# Open profile
notepad $PROFILE

# Add these lines:
function Start-IICPC { .\START-WINDOWS.ps1 }
function Stop-IICPC { .\STOP-WINDOWS.ps1 }
function Logs-IICPC { docker compose logs -f }

# Save and reload
. $PROFILE
```

Now you can use:
```powershell
Start-IICPC
Stop-IICPC
Logs-IICPC
```

---

## Testing on Windows

### Run Tests

```powershell
# Go tests
docker compose exec api-gateway go test ./...
docker compose exec bot-fleet go test ./...

# Rust tests
docker compose exec telemetry-ingester cargo test

# Python tests
docker compose exec predictor pytest
```

### Load Testing

```powershell
# Using curl (install from https://curl.se/windows/)
curl -X POST http://localhost:8080/api/v1/submissions `
  -H "Content-Type: application/json" `
  -H "Authorization: Bearer YOUR_TOKEN" `
  -d '{\"team_name\":\"Test Team\"}'
```

---

## Performance Tips for Windows

1. **Use WSL 2 Backend**
   - Docker Desktop → Settings → General → Use WSL 2 based engine

2. **Allocate More Resources**
   - Docker Desktop → Settings → Resources
   - CPU: 4+ cores
   - Memory: 8GB+ (16GB recommended)
   - Swap: 2GB
   - Disk: 100GB+

3. **Disable Antivirus Scanning**
   - Add Docker directories to exclusions
   - Improves build and startup times

4. **Use SSD**
   - Store project on SSD, not HDD
   - Significantly faster builds

---

## Troubleshooting Checklist

- [ ] Docker Desktop is installed and running
- [ ] WSL 2 is enabled
- [ ] Docker has enough memory (8GB+)
- [ ] Ports 3000, 8080, 8090, etc. are not in use
- [ ] Firewall allows Docker
- [ ] Antivirus is not blocking Docker
- [ ] Project is on SSD (not HDD)

---

## Getting Help

If you encounter issues:

1. **Check Docker Desktop logs:**
   - Docker Desktop → Troubleshoot → View logs

2. **Check service logs:**
   ```powershell
   docker compose logs -f
   ```

3. **Restart Docker Desktop:**
   - Right-click Docker icon → Restart

4. **Clean restart:**
   ```powershell
   .\STOP-WINDOWS.ps1
   docker system prune -a
   .\START-WINDOWS.ps1
   ```

---

## Next Steps

Once the platform is running:

1. ✅ Visit http://localhost:3000
2. ✅ Explore the leaderboard
3. ✅ Submit your first trading engine
4. ✅ View analytics and metrics

**Happy coding! 🚀**
