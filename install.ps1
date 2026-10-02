# GoFetch installer for Windows (PowerShell 5.1+ / PowerShell 7+).
#
# Builds the sources in this directory (no network) and installs `gofetch.exe`
# into a per-user bin directory that is added to the user PATH.
#
# Usage:
#   ./install.ps1
#   $env:GOFETCH_BIN_DIR = "C:\Tools"; ./install.ps1

$ErrorActionPreference = "Stop"

$RootDir = Split-Path -Parent $MyInvocation.MyCommand.Path

if (-not (Get-Command go -ErrorAction SilentlyContinue)) {
    Write-Error "Go non è installato o non è nel PATH. Installa Go 1.23+ da https://go.dev/dl/ e riprova."
    exit 1
}

if (-not (Get-Command git -ErrorAction SilentlyContinue)) {
    Write-Error "Git non è installato o non è nel PATH."
    exit 1
}

if ($env:GOFETCH_BIN_DIR) {
    $BinDir = $env:GOFETCH_BIN_DIR
} else {
    $BinDir = Join-Path $HOME ".local\bin"
}

New-Item -ItemType Directory -Force -Path $BinDir | Out-Null

$Commit = "locale"
$Date = "non disponibile"
if (Test-Path (Join-Path $RootDir ".git")) {
    $Commit = (& git -C $RootDir log -1 --format="%h" 2>$null)
    if (-not $Commit) { $Commit = "locale" }
    $Date = (& git -C $RootDir log -1 --format="%cI" 2>$null)
    if (-not $Date) { $Date = "non disponibile" }
}

Write-Host "Installazione del commit $Commit ($Date) in: $BinDir"

$TempBinary = Join-Path ([System.IO.Path]::GetTempPath()) ("gofetch-" + [System.Guid]::NewGuid().ToString("N") + ".exe")
try {
    Push-Location $RootDir
    & go build -trimpath -ldflags="-s -w" -o $TempBinary .
    if ($LASTEXITCODE -ne 0) {
        throw "go build non riuscito (exit $LASTEXITCODE)"
    }
} finally {
    Pop-Location
}

$Target = Join-Path $BinDir "gofetch.exe"
Move-Item -Force $TempBinary $Target
Write-Host "GoFetch installato: $Target"

# Add the bin directory to the persistent user PATH if it is not there yet.
$UserPath = [Environment]::GetEnvironmentVariable("Path", "User")
if (-not $UserPath) { $UserPath = "" }
$Entries = $UserPath.Split(";", [System.StringSplitOptions]::RemoveEmptyEntries)
if ($Entries -notcontains $BinDir) {
    $NewPath = (($Entries + $BinDir) -join ";")
    [Environment]::SetEnvironmentVariable("Path", $NewPath, "User")
    Write-Host "PATH aggiornato (utente). Apri un nuovo terminale per usarlo."
}

Write-Host ""
Write-Host "Avvia con: gofetch"
