$ErrorActionPreference = 'Stop'
Push-Location (Split-Path $PSScriptRoot -Parent)
try {
    go -C tools tool go-winres make --in ../build/windows/winres.json --out ../cmd/llvy-uploader/rsrc --arch amd64
    if ($LASTEXITCODE -ne 0) { throw 'Resource generation failed' }
    New-Item -ItemType Directory -Force dist | Out-Null
    $sourceCommit = git rev-parse --verify HEAD 2>$null
    if ($LASTEXITCODE -ne 0) { $sourceCommit = 'uncommitted' }
    if (git status --porcelain) { $sourceCommit += '-dirty' }
    $env:CGO_ENABLED = '0'
    $env:GOOS = 'windows'
    $env:GOARCH = 'amd64'
    go build -trimpath -ldflags "-H windowsgui -X main.version=0.0.0-dev -X main.commit=$sourceCommit" -o dist/llvy-uploader.exe ./cmd/llvy-uploader
    if ($LASTEXITCODE -ne 0) { throw 'Build failed' }
    (Get-FileHash dist/llvy-uploader.exe -Algorithm SHA256).Hash.ToLower() + '  llvy-uploader.exe' | Set-Content dist/SHA256SUMS.txt
    go version -m dist/llvy-uploader.exe | Set-Content dist/build-info.txt
    "Source: $sourceCommit`nStatus: unsigned connected development build; Windows and external acceptance pending" | Set-Content dist/provenance.txt
} finally {
    Remove-Item cmd/llvy-uploader/rsrc_windows_amd64.syso -ErrorAction SilentlyContinue
    Pop-Location
}
