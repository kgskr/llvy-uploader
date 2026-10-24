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
    Copy-Item THIRD_PARTY_NOTICES.txt dist/THIRD_PARTY_NOTICES.txt -Force
    Remove-Item dist/release-evidence.json -ErrorAction SilentlyContinue
    $exePath = Join-Path dist $versionInfo.FileName
    $sourceCommit = git rev-parse --verify HEAD 2>$null
    if ($LASTEXITCODE -ne 0) { $sourceCommit = 'uncommitted' }
    if (git status --porcelain) { $sourceCommit += '-dirty' }
    if (-not [string]::IsNullOrEmpty($env:LLVY_RELEASE_EVIDENCE_JSON)) {
        $env:RELEASE_VERSION = $Version
        $env:SOURCE_COMMIT = $sourceCommit
        $env:RELEASE_EVIDENCE_OUTPUT = 'dist/release-evidence.json'
        try {
            go run ./scripts/validate-release
            if ($LASTEXITCODE -ne 0) { throw 'Release acceptance evidence failed' }
        } finally { Remove-Item Env:RELEASE_EVIDENCE_OUTPUT -ErrorAction SilentlyContinue }
    }
    $env:CGO_ENABLED = '0'
    $env:GOOS = 'windows'
    $env:GOARCH = 'amd64'
    go build -trimpath -ldflags "-H windowsgui -X main.version=$Version -X main.commit=$sourceCommit -X llvy-uploader/internal/desktop.ServerOrigin=$serverOrigin" -o $exePath ./cmd/llvy-uploader
    if ($LASTEXITCODE -ne 0) { throw 'Build failed' }
    (Get-FileHash $exePath -Algorithm SHA256).Hash.ToLower() + '  ' + $versionInfo.FileName | Set-Content dist/SHA256SUMS.txt
    go version -m $exePath | Set-Content dist/build-info.txt
    if ($LASTEXITCODE -ne 0) { throw 'Build information extraction failed' }
    $acceptance = if (-not (Test-Path dist/release-evidence.json)) { 'development build; real PC and external acceptance pending; publication blocked' } elseif ($env:RELEASE_CHANNEL -ceq 'prerelease') { 'test prerelease; real PC and external acceptance pending; see release-evidence.json' } else { 'reviewed acceptance: release-evidence.json' }
    "Version: $Version`nFile: $($versionInfo.FileName)`nSource: $sourceCommit`nServer origin: $serverOrigin`nUnconfigured self-test only: $([string]::IsNullOrEmpty($serverOrigin))`nStatus: unsigned; $acceptance" | Set-Content dist/provenance.txt
} finally {
    Remove-Item $resourcePath -ErrorAction SilentlyContinue
    Remove-Item cmd/llvy-uploader/rsrc_windows_amd64.syso -ErrorAction SilentlyContinue
    Pop-Location
}
