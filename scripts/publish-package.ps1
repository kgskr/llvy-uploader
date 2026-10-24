param(
    [Parameter(Mandatory)][string]$Version,
    [Parameter(Mandatory)][string]$Repository,
    [Parameter(Mandatory)][string]$Commit
)
$ErrorActionPreference = 'Stop'
. (Join-Path $PSScriptRoot 'version.ps1')
$versionInfo = Get-UploaderVersion -Version $Version
$channel = if ([string]::IsNullOrEmpty($env:RELEASE_CHANNEL)) { 'stable' } else { $env:RELEASE_CHANNEL }
Assert-UploaderReleaseChannel -Version $Version -Channel $channel
$description = if ($channel -ceq 'prerelease') { 'Unsigned test prerelease; real PC and external acceptance pending' } else { 'Unsigned Windows uploader; reviewed acceptance included' }
if ($Repository -cnotmatch '\A[a-zA-Z0-9_.-]+/[a-zA-Z0-9_.-]+\z' -or $Commit -cnotmatch '\A[0-9a-f]{40}\z') { throw 'Invalid package provenance' }
if ([string]::IsNullOrEmpty($env:GH_TOKEN)) { throw 'Missing package publication token' }
$reference = "ghcr.io/$($Repository.ToLowerInvariant()):$Version"
$registryConfig = Join-Path ([IO.Path]::GetTempPath()) ([guid]::NewGuid().ToString() + '.json')
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
    foreach ($file in $files) { if (-not (Test-Path -LiteralPath $file -PathType Leaf)) { throw "Missing package file: $file" } }
    $checksum = (Get-FileHash $versionInfo.FileName -Algorithm SHA256).Hash.ToLowerInvariant() + '  ' + $versionInfo.FileName
    if ((Get-Content -Raw SHA256SUMS.txt).TrimEnd() -cne $checksum) { throw 'Executable checksum mismatch before publication' }
    $env:GH_TOKEN | oras login ghcr.io --username $env:GITHUB_ACTOR --password-stdin --registry-config $registryConfig
    if ($LASTEXITCODE -ne 0) { throw 'GHCR login failed' }
    $layers = @("$($versionInfo.FileName):application/octet-stream", 'SHA256SUMS.txt:text/plain', 'build-info.txt:text/plain', 'provenance.txt:text/plain', 'THIRD_PARTY_NOTICES.txt:text/plain', 'release-evidence.json:application/json')
    oras push --registry-config $registryConfig --artifact-type application/vnd.llvy.uploader.v1 --artifact-platform windows/amd64 --annotation "org.opencontainers.image.source=https://github.com/$Repository" --annotation "org.opencontainers.image.version=$Version" --annotation "org.opencontainers.image.revision=$Commit" --annotation "org.opencontainers.image.description=$description" $reference @layers
    if ($LASTEXITCODE -ne 0) { throw 'GHCR publication failed' }
    $digest = oras resolve --registry-config $registryConfig $reference
    if ($LASTEXITCODE -ne 0 -or $digest -cnotmatch '\Asha256:[0-9a-f]{64}\z') { throw 'Cannot resolve published package digest' }
    $immutableReference = "ghcr.io/$($Repository.ToLowerInvariant())@$digest"
    oras pull --registry-config $registryConfig $immutableReference --output $download
    if ($LASTEXITCODE -ne 0) { throw 'Published package download failed' }
    foreach ($file in $files) {
        if ((Get-FileHash -LiteralPath $file -Algorithm SHA256).Hash -cne (Get-FileHash -LiteralPath (Join-Path $download $file) -Algorithm SHA256).Hash) { throw "Published file checksum mismatch: $file" }
    }
    if ($env:GITHUB_STEP_SUMMARY) {
        @"
Published GitHub Packages artifact: $reference

Executable: $($versionInfo.FileName)

Verified digest: $digest

Download: oras pull $immutableReference

$description. Verification details are recorded in release-evidence.json.
"@ | Add-Content $env:GITHUB_STEP_SUMMARY
    }
} finally {
    Remove-Item $registryConfig -ErrorAction SilentlyContinue
    Remove-Item $download -Recurse -Force -ErrorAction SilentlyContinue
    Pop-Location
}
