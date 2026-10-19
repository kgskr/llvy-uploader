param([string]$Version = '0.0.0-dev', [switch]$AllowUnconfigured)
$ErrorActionPreference = 'Stop'
. (Join-Path $PSScriptRoot 'version.ps1')
$versionInfo = Get-UploaderVersion -Version $Version
$resourcePath = [IO.Path]::GetTempFileName()
Push-Location (Split-Path $PSScriptRoot -Parent)
try {
    $serverOrigin = $env:LLVY_SERVER_ORIGIN
    if ([string]::IsNullOrEmpty($serverOrigin)) {
        if (-not $AllowUnconfigured) { throw 'Set the LLVY_SERVER_ORIGIN GitHub Actions variable or environment variable before building.' }
    } else {
        go run ./scripts/validate-origin
        if ($LASTEXITCODE -ne 0) { throw 'Invalid LLVY_SERVER_ORIGIN. Expected an HTTPS origin without path, query or credentials.' }
    }
    New-UploaderVersionResource -VersionInfo $versionInfo -Source build/windows/winres.json -Destination $resourcePath
    go -C tools tool go-winres make --in $resourcePath --out ../cmd/llvy-uploader/rsrc --arch amd64
    if ($LASTEXITCODE -ne 0) { throw 'Resource generation failed' }
    New-Item -ItemType Directory -Force dist | Out-Null
    $exePath = Join-Path dist $versionInfo.FileName
    $sourceCommit = git rev-parse --verify HEAD 2>$null
    if ($LASTEXITCODE -ne 0) { $sourceCommit = 'uncommitted' }
    if (git status --porcelain) { $sourceCommit += '-dirty' }
    $env:CGO_ENABLED = '0'
    $env:GOOS = 'windows'
    $env:GOARCH = 'amd64'
    go build -trimpath -ldflags "-H windowsgui -X main.version=$Version -X main.commit=$sourceCommit -X llvy-uploader/internal/desktop.ServerOrigin=$serverOrigin" -o $exePath ./cmd/llvy-uploader
    if ($LASTEXITCODE -ne 0) { throw 'Build failed' }
    (Get-FileHash $exePath -Algorithm SHA256).Hash.ToLower() + '  ' + $versionInfo.FileName | Set-Content dist/SHA256SUMS.txt
    go version -m $exePath | Set-Content dist/build-info.txt
    if ($LASTEXITCODE -ne 0) { throw 'Build information extraction failed' }
    "Version: $Version`nFile: $($versionInfo.FileName)`nSource: $sourceCommit`nServer origin: $serverOrigin`nUnconfigured self-test only: $([string]::IsNullOrEmpty($serverOrigin))`nStatus: unsigned development build; real PC and external acceptance pending" | Set-Content dist/provenance.txt
} finally {
    Remove-Item $resourcePath -ErrorAction SilentlyContinue
    Remove-Item cmd/llvy-uploader/rsrc_windows_amd64.syso -ErrorAction SilentlyContinue
    Pop-Location
}
