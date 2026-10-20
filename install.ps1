param(
    [string]$Version = $env:BRAIN_VERSION,
    [string]$InstallDir = $env:BRAIN_INSTALL_DIR
)

$ErrorActionPreference = "Stop"
$Repo = "WolfCanCode/personal-ai-brain"
if ([string]::IsNullOrWhiteSpace($InstallDir)) {
    $InstallDir = Join-Path $HOME ".local\bin"
}
if ([string]::IsNullOrWhiteSpace($Version)) {
    $Release = Invoke-RestMethod -Headers @{ "User-Agent" = "personal-ai-brain-installer" } -Uri "https://api.github.com/repos/$Repo/releases/latest"
    $Version = $Release.tag_name
}
if ($Version -notmatch '^v[0-9]+\.[0-9]+\.[0-9]+(-[0-9A-Za-z.-]+)?$') {
    throw "brain installer: invalid release tag: $Version"
}

$Asset = "brain_${Version}_windows_amd64.zip"
$BaseUrl = "https://github.com/$Repo/releases/download/$Version"
$TempDir = Join-Path ([System.IO.Path]::GetTempPath()) ("brain-install-" + [guid]::NewGuid())
$Archive = Join-Path $TempDir $Asset
$Sums = Join-Path $TempDir "SHA256SUMS"

try {
    New-Item -ItemType Directory -Path $TempDir -Force | Out-Null
    Write-Host "Downloading Noevyra $Version for windows/amd64..."
    Invoke-WebRequest -Uri "$BaseUrl/$Asset" -OutFile $Archive
    Invoke-WebRequest -Uri "$BaseUrl/SHA256SUMS" -OutFile $Sums

    $Expected = $null
    $Pattern = '^([0-9A-Fa-f]{64})\s+\*?' + [regex]::Escape($Asset) + '$'
    foreach ($Line in Get-Content $Sums) {
        if ($Line -match $Pattern) {
            $Expected = $Matches[1].ToLowerInvariant()
            break
        }
    }
    if (-not $Expected) {
        throw "brain installer: checksum entry missing for $Asset"
    }
    $Actual = (Get-FileHash -Algorithm SHA256 $Archive).Hash.ToLowerInvariant()
    if ($Actual -ne $Expected) {
        throw "brain installer: SHA-256 verification failed for $Asset"
    }

    $ExtractDir = Join-Path $TempDir "extract"
    Expand-Archive -Path $Archive -DestinationPath $ExtractDir -Force
    $Binary = Join-Path $ExtractDir "brain.exe"
    if (-not (Test-Path -Path $Binary -PathType Leaf)) {
        throw "brain installer: archive does not contain brain.exe"
    }
    New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null
    Copy-Item -Path $Binary -Destination (Join-Path $InstallDir "brain.exe") -Force
    & (Join-Path $InstallDir "brain.exe") version | Out-Null

    Write-Host "Installed Noevyra $Version to $InstallDir\brain.exe"
    if (($env:Path -split ';') -notcontains $InstallDir) {
        Write-Host "Add $InstallDir to PATH, then run: brain init"
    }
}
finally {
    if (Test-Path $TempDir) {
        Remove-Item -Path $TempDir -Recurse -Force
    }
}

