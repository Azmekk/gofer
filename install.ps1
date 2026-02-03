$ErrorActionPreference = "Stop"

$Repo = "Azmekk/gofer"

# Detect architecture
try {
    Add-Type -AssemblyName "System.Runtime.InteropServices.RuntimeInformation" -ErrorAction SilentlyContinue
    $OsArch = [System.Runtime.InteropServices.RuntimeInformation]::OSArchitecture.ToString().ToLower()
} catch {
    $OsArch = $env:PROCESSOR_ARCHITECTURE
}

switch ($OsArch) {
    { $_ -in "x64", "amd64" }  { $Arch = "amd64" }
    { $_ -in "arm64", "aarch64" } { $Arch = "arm64" }
    default { Write-Error "Unsupported architecture: $OsArch"; exit 1 }
}

# Determine install directory
if ($env:GOFER_INSTALL_DIR) {
    $InstallDir = $env:GOFER_INSTALL_DIR
} else {
    $InstallDir = Join-Path $env:LOCALAPPDATA "gofer"
}

$Binary = "gofer-windows-${Arch}.exe"

Write-Host "Fetching latest release..."
$Release = Invoke-RestMethod -Uri "https://api.github.com/repos/$Repo/releases/latest" -Headers @{ "User-Agent" = "gofer-installer" }
$Tag = $Release.tag_name
if (-not $Tag) {
    Write-Error "Could not determine latest release tag."
    exit 1
}
Write-Host "Latest version: $Tag"

$DownloadUrl = "https://github.com/$Repo/releases/download/$Tag/$Binary"
$ChecksumsUrl = "https://github.com/$Repo/releases/download/$Tag/checksums.txt"

$TempDir = Join-Path ([System.IO.Path]::GetTempPath()) ([System.Guid]::NewGuid().ToString())
New-Item -ItemType Directory -Path $TempDir | Out-Null

try {
    $TempFile = Join-Path $TempDir "gofer.exe"

    Write-Host "Downloading $Binary..."
    Invoke-WebRequest -Uri $DownloadUrl -OutFile $TempFile -UseBasicParsing

    # Verify checksum
    Write-Host "Verifying checksum..."
    try {
        $ChecksumsContent = (Invoke-WebRequest -Uri $ChecksumsUrl -UseBasicParsing).Content
        $ExpectedHash = ($ChecksumsContent -split "`n" | Where-Object { $_ -match $Binary } | ForEach-Object { ($_ -split "\s+")[0] })

        if ($ExpectedHash) {
            $ActualHash = (Get-FileHash -Path $TempFile -Algorithm SHA256).Hash.ToLower()
            if ($ActualHash -ne $ExpectedHash) {
                Write-Error "Checksum mismatch! Expected: $ExpectedHash, Got: $ActualHash"
                exit 1
            }
            Write-Host "Checksum verified."
        } else {
            Write-Host "Warning: binary not found in checksums.txt, skipping verification."
        }
    } catch {
        Write-Host "Warning: could not download checksums, skipping verification."
    }

    # Install
    if (-not (Test-Path $InstallDir)) {
        New-Item -ItemType Directory -Path $InstallDir | Out-Null
    }

    $DestPath = Join-Path $InstallDir "gofer.exe"
    Move-Item -Path $TempFile -Destination $DestPath -Force

    Write-Host "Installed gofer to $DestPath"

    # Add to PATH if not already there
    $UserPath = [Environment]::GetEnvironmentVariable("Path", "User")
    if ($UserPath -notlike "*$InstallDir*") {
        [Environment]::SetEnvironmentVariable("Path", "$UserPath;$InstallDir", "User")
        Write-Host "Added $InstallDir to your user PATH. Restart your terminal for it to take effect."
    }
} finally {
    Remove-Item -Path $TempDir -Recurse -Force -ErrorAction SilentlyContinue
}
