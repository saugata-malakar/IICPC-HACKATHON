# 🐳 Install Docker Desktop on Windows

## Step-by-Step Installation Guide

### Step 1: Check System Requirements

**Minimum Requirements:**
- Windows 10 64-bit: Pro, Enterprise, or Education (Build 19041 or higher)
- OR Windows 11 64-bit
- 4GB RAM minimum (8GB+ recommended)
- BIOS-level hardware virtualization support enabled

**Check Your Windows Version:**
```powershell
# Run in PowerShell
winver
```

### Step 2: Enable WSL 2 (Windows Subsystem for Linux)

**Open PowerShell as Administrator** (Right-click Start → Windows PowerShell (Admin))

```powershell
# Enable WSL
dism.exe /online /enable-feature /featurename:Microsoft-Windows-Subsystem-Linux /all /norestart

# Enable Virtual Machine Platform
dism.exe /online /enable-feature /featurename:VirtualMachinePlatform /all /norestart

# Restart your computer
Restart-Computer
```

After restart, open PowerShell as Administrator again:

```powershell
# Set WSL 2 as default
wsl --set-default-version 2

# Install Ubuntu (optional but recommended)
wsl --install -d Ubuntu
```

### Step 3: Download Docker Desktop

1. **Visit:** https://www.docker.com/products/docker-desktop
2. **Click:** "Download for Windows"
3. **File:** `Docker Desktop Installer.exe` (~500MB)

### Step 4: Install Docker Desktop

1. **Run** `Docker Desktop Installer.exe`
2. **Check** "Use WSL 2 instead of Hyper-V" (recommended)
3. **Click** "Ok" to start installation
4. **Wait** for installation to complete (~5 minutes)
5. **Click** "Close and restart" when prompted
6. **Restart** your computer

### Step 5: Start Docker Desktop

1. **Open** Docker Desktop from Start Menu
2. **Accept** the service agreement
3. **Wait** for Docker to start (whale icon stops animating)
4. **Skip** the tutorial (or complete it if you're new to Docker)

### Step 6: Verify Installation

Open PowerShell and run:

```powershell
# Check Docker version
docker --version
# Should show: Docker version 26.x.x

# Check Docker Compose version
docker compose version
# Should show: Docker Compose version v2.x.x

# Test Docker
docker run hello-world
# Should download and run a test container
```

### Step 7: Configure Docker Resources

1. **Open** Docker Desktop
2. **Click** Settings (gear icon)
3. **Go to** Resources
4. **Set:**
   - **CPUs:** 4+ (more is better)
   - **Memory:** 8GB minimum (16GB recommended for IICPC platform)
   - **Swap:** 2GB
   - **Disk image size:** 100GB+
5. **Click** "Apply & Restart"

---

## Alternative: Docker Desktop Not Available?

If you're on **Windows 10 Home** or can't install Docker Desktop:

### Option 1: Upgrade to Windows 10 Pro

Windows 10 Home → Pro upgrade:
1. Settings → Update & Security → Activation
2. Click "Go to Store"
3. Purchase Windows 10 Pro upgrade (~$99)

### Option 2: Use Docker Toolbox (Legacy)

**Not recommended** but works on older systems:
1. Download: https://github.com/docker/toolbox/releases
2. Install Docker Toolbox
3. Use Docker Quickstart Terminal

### Option 3: Use Cloud Development Environment

**GitHub Codespaces** (Free tier available):
1. Push code to GitHub
2. Open in Codespaces
3. Docker is pre-installed

**Gitpod** (Free tier available):
1. Push code to GitHub/GitLab
2. Open in Gitpod
3. Docker is pre-installed

---

## After Docker is Installed

### Start the IICPC Platform

```powershell
# Navigate to project
cd C:\Users\trina\Downloads\PROJECTS\IICPC\iicpc-championship-v2\iicpc-final

# Start platform
.\START-WINDOWS.ps1

# Or manually:
docker compose up -d --build
```

### Access the Platform

- **Frontend:** http://localhost:3000
- **API Gateway:** http://localhost:8080
- **Kafka UI:** http://localhost:8090
- **Grafana:** http://localhost:3001

---

## Troubleshooting

### Issue: "Virtualization is not enabled"

**Solution:**
1. Restart computer
2. Enter BIOS (usually F2, F10, or Del during boot)
3. Find "Virtualization Technology" or "Intel VT-x" or "AMD-V"
4. Enable it
5. Save and exit BIOS

### Issue: "WSL 2 installation is incomplete"

**Solution:**
```powershell
# Run as Administrator
wsl --install
wsl --set-default-version 2

# Restart computer
```

### Issue: "Docker Desktop failed to start"

**Solution:**
1. Open Task Manager (Ctrl+Shift+Esc)
2. End all Docker processes
3. Restart Docker Desktop
4. If still fails, reinstall Docker Desktop

### Issue: "Hyper-V is not available"

**Solution:**
- Use WSL 2 backend instead (recommended)
- Or enable Hyper-V:
  ```powershell
  # Run as Administrator
  Enable-WindowsOptionalFeature -Online -FeatureName Microsoft-Hyper-V -All
  ```

---

## Quick Reference

### Docker Commands

```powershell
# Start containers
docker compose up -d

# Stop containers
docker compose down

# View logs
docker compose logs -f

# Check status
docker compose ps

# Restart
docker compose restart

# Clean up
docker system prune -a
```

### Docker Desktop

- **Start:** Windows Start Menu → Docker Desktop
- **Settings:** Click gear icon
- **Restart:** Right-click Docker icon → Restart
- **Quit:** Right-click Docker icon → Quit Docker Desktop

---

## Need Help?

### Official Resources

- **Docker Docs:** https://docs.docker.com/desktop/windows/install/
- **WSL 2 Docs:** https://docs.microsoft.com/en-us/windows/wsl/install
- **Docker Forums:** https://forums.docker.com/

### IICPC Platform Help

- **Windows Guide:** Read `WINDOWS-GUIDE.md`
- **Getting Started:** Read `GETTING_STARTED.md`
- **Architecture:** Read `ARCHITECTURE.md`

---

## Estimated Time

- **WSL 2 Setup:** 10-15 minutes (includes restart)
- **Docker Desktop Download:** 5-10 minutes (depends on internet)
- **Docker Desktop Install:** 5-10 minutes (includes restart)
- **Platform First Start:** 5-10 minutes (downloads images)

**Total:** ~30-45 minutes for first-time setup

---

## Once Docker is Running

You're ready to start the championship platform! 🚀

```powershell
cd iicpc-championship-v2\iicpc-final
.\START-WINDOWS.ps1
```

Then open http://localhost:3000 in your browser!

**Good luck with the hackathon! 🏆**
