$ErrorActionPreference = "Stop"

$Repo = "charlie0077/tmux-manager"
$Bin  = "tmux-manager.exe"

# Detect arch
$Arch = if ([System.Runtime.InteropServices.RuntimeInformation]::OSArchitecture -eq "Arm64") { "arm64" } else { "amd64" }

# Get latest release tag
$Latest = (Invoke-RestMethod "https://api.github.com/repos/$Repo/releases/latest").tag_name
if (-not $Latest) {
    Write-Error "Could not determine latest release. Check https://github.com/$Repo/releases"
    exit 1
}

$Filename = "tmux-manager_windows_$Arch.exe"
$Url      = "https://github.com/$Repo/releases/download/$Latest/$Filename"

# Choose install dir: prefer a writable dir already on PATH, else use LocalAppData
$InstallDir = "$env:LOCALAPPDATA\tmux-manager"
New-Item -ItemType Directory -Force -Path $InstallDir | Out-Null

$Dest = Join-Path $InstallDir $Bin

Write-Host "Installing tmux-manager $Latest (windows/$Arch)..."
Invoke-WebRequest -Uri $Url -OutFile $Dest

# Add to user PATH if not already present
$UserPath = [Environment]::GetEnvironmentVariable("PATH", "User")
if ($UserPath -notlike "*$InstallDir*") {
    [Environment]::SetEnvironmentVariable("PATH", "$UserPath;$InstallDir", "User")
    Write-Host "Added $InstallDir to your PATH (restart your terminal to take effect)"
}

Write-Host "Installed to $Dest"
Write-Host "Run: tmux-manager"
