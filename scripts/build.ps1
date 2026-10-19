param([switch]$AllowUnconfigured)
$ErrorActionPreference = 'Stop'
Push-Location (Split-Path $PSScriptRoot -Parent)
try {
    $serverOrigin = $env:LLVY_SERVER_ORIGIN
    if ([string]::IsNullOrEmpty($serverOrigin)) {
        if (-not $AllowUnconfigured) { throw 'Set the LLVY_SERVER_ORIGIN GitHub Actions variable or environment variable before building.' }
    } else {
        go run ./scripts/validate-origin
        if ($LASTEXITCODE -ne 0) { throw 'Invalid LLVY_SERVER_ORIGIN. Expected an HTTPS origin without path, query or credentials.' }
    }
    go -C tools tool go-winres make --in ../build/windows/winres.json --out ../cmd/llvy-uploader/rsrc --arch amd64
    if ($LASTEXITCODE -ne 0) { throw 'Resource generation failed' }
    New-Item -ItemType Directory -Force dist | Out-Null
    $sourceCommit = git rev-parse --verify HEAD 2>$null
    if ($LASTEXITCODE -ne 0) { $sourceCommit = 'uncommitted' }
    if (git status --porcelain) { $sourceCommit += '-dirty' }
    $env:CGO_ENABLED = '0'
    $env:GOOS = 'windows'
    $env:GOARCH = 'amd64'
    go build -trimpath -ldflags "-H windowsgui -X main.version=0.0.0-dev -X main.commit=$sourceCommit -X llvy-uploader/internal/desktop.ServerOrigin=$serverOrigin" -o dist/llvy-uploader.exe ./cmd/llvy-uploader
    if ($LASTEXITCODE -ne 0) { throw 'Build failed' }
    (Get-FileHash dist/llvy-uploader.exe -Algorithm SHA256).Hash.ToLower() + '  llvy-uploader.exe' | Set-Content dist/SHA256SUMS.txt
    go version -m dist/llvy-uploader.exe | Set-Content dist/build-info.txt
    "Source: $sourceCommit`nServer origin: $serverOrigin`nUnconfigured self-test only: $([string]::IsNullOrEmpty($serverOrigin))`nStatus: unsigned development build; real PC and external acceptance pending" | Set-Content dist/provenance.txt
} finally {
    Remove-Item cmd/llvy-uploader/rsrc_windows_amd64.syso -ErrorAction SilentlyContinue
    Pop-Location
}
