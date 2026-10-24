param(
    [Parameter(Mandatory)][string]$Version,
    [Parameter(Mandatory)][string]$Tag,
    [Parameter(Mandatory)][string]$Repository,
    [Parameter(Mandatory)][string]$Commit
)
$ErrorActionPreference = 'Stop'
. (Join-Path $PSScriptRoot 'version.ps1')
$versionInfo = Get-UploaderReleaseVersion -Tag $Tag
if ($versionInfo.Version -cne $Version) { throw 'Release tag/version mismatch' }
$channel = if ([string]::IsNullOrEmpty($env:RELEASE_CHANNEL)) { 'stable' } else { $env:RELEASE_CHANNEL }
Assert-UploaderReleaseChannel -Version $Version -Channel $channel
$description = if ($channel -ceq 'prerelease') { 'Unsigned test prerelease; real PC and external acceptance pending' } else { 'Unsigned Windows uploader; reviewed acceptance included' }
if ($Repository -cnotmatch '\A[a-zA-Z0-9_.-]+/[a-zA-Z0-9_.-]+\z' -or $Commit -cnotmatch '\A[0-9a-f]{40}\z') { throw 'Invalid release provenance' }
if ([string]::IsNullOrEmpty($env:GH_TOKEN)) { throw 'Missing release publication token' }
$download = Join-Path ([IO.Path]::GetTempPath()) ([guid]::NewGuid().ToString())
Push-Location (Split-Path $PSScriptRoot -Parent)
try {
    $env:RELEASE_VERSION = $Version
    $env:SOURCE_COMMIT = $Commit
    go run ./scripts/validate-release dist/release-evidence.json
    if ($LASTEXITCODE -ne 0) { throw 'Release acceptance evidence failed' }
    if ((Get-FileHash THIRD_PARTY_NOTICES.txt).Hash -cne (Get-FileHash dist/THIRD_PARTY_NOTICES.txt).Hash) { throw 'Dependency notices mismatch' }
    $provenance = Get-Content dist/provenance.txt
    if ($provenance -cnotcontains "Source: $Commit" -or $provenance -cnotcontains "Version: $Version" -or $provenance -cnotcontains "Server origin: $env:LLVY_SERVER_ORIGIN") { throw 'Build provenance mismatch' }
} finally { Pop-Location }
Push-Location (Join-Path $PSScriptRoot '../dist')
try {
    $files = @($versionInfo.FileName, 'SHA256SUMS.txt', 'build-info.txt', 'provenance.txt', 'THIRD_PARTY_NOTICES.txt', 'release-evidence.json')
    foreach ($file in $files) { if (-not (Test-Path -LiteralPath $file -PathType Leaf)) { throw "Missing release file: $file" } }
    $checksum = (Get-FileHash $versionInfo.FileName -Algorithm SHA256).Hash.ToLowerInvariant() + '  ' + $versionInfo.FileName
    if ((Get-Content -Raw SHA256SUMS.txt).TrimEnd() -cne $checksum) { throw 'Executable checksum mismatch before publication' }
    $releaseJSON = gh release view $Tag --repo $Repository --json tagName,isDraft,isPrerelease,assets,url
    if ($LASTEXITCODE -ne 0) { throw 'Cannot read release metadata' }
    $release = $releaseJSON | ConvertFrom-Json
    if ($release.tagName -cne $Tag -or $release.isDraft -ne $false -or $release.isPrerelease -ne ($channel -ceq 'prerelease')) { throw 'Release metadata/channel mismatch' }
    $tagCommit = gh api "repos/$Repository/commits/$Tag" --jq .sha
    if ($LASTEXITCODE -ne 0 -or $tagCommit -cne $Commit) { throw 'Release tag commit mismatch' }
    New-Item -ItemType Directory -Path $download | Out-Null
    $missing = @()
    # Check every existing file before adding anything. Never replace published bytes.
    foreach ($file in $files) {
        if ($release.assets.name -ccontains $file) {
            gh release download $Tag --repo $Repository --pattern $file --dir $download
            if ($LASTEXITCODE -ne 0) { throw "Existing release download failed: $file" }
            if ((Get-FileHash -LiteralPath $file -Algorithm SHA256).Hash -cne (Get-FileHash -LiteralPath (Join-Path $download $file) -Algorithm SHA256).Hash) { throw "Existing release asset differs: $file" }
            Remove-Item -LiteralPath (Join-Path $download $file)
        } else { $missing += $file }
    }
    if ($missing.Count -gt 0) {
        gh release upload $Tag @missing --repo $Repository
        if ($LASTEXITCODE -ne 0) { throw 'Release asset upload failed' }
    }
    foreach ($file in $files) {
        gh release download $Tag --repo $Repository --pattern $file --dir $download
        if ($LASTEXITCODE -ne 0) { throw "Published release download failed: $file" }
        if ((Get-FileHash -LiteralPath $file -Algorithm SHA256).Hash -cne (Get-FileHash -LiteralPath (Join-Path $download $file) -Algorithm SHA256).Hash) { throw "Published file checksum mismatch: $file" }
    }
    if ($env:GITHUB_STEP_SUMMARY) {
        @"
Published Windows executable: $($release.url)

Executable: $($versionInfo.FileName)

Verified SHA-256: $checksum

Download: https://github.com/$Repository/releases/download/$Tag/$($versionInfo.FileName)

$description. Verification details are recorded in release-evidence.json.
"@ | Add-Content $env:GITHUB_STEP_SUMMARY
    }
} finally {
    Remove-Item $download -Recurse -Force -ErrorAction SilentlyContinue
    Pop-Location
}
