$ErrorActionPreference = 'Stop'
$root = Join-Path ([IO.Path]::GetTempPath()) ([guid]::NewGuid().ToString())
$names = @('GH_TOKEN','RELEASE_CHANNEL','RELEASE_VERSION','SOURCE_COMMIT','LLVY_SERVER_ORIGIN','GITHUB_STEP_SUMMARY')
$saved = @{}
$mock = @{}
foreach ($name in $names) { $saved[$name] = [Environment]::GetEnvironmentVariable($name) }
try {
    New-Item -ItemType Directory -Path "$root/scripts", "$root/dist", "$root/remote" | Out-Null
    Copy-Item "$PSScriptRoot/version.ps1", "$PSScriptRoot/publish-release.ps1" "$root/scripts"
    $env:GH_TOKEN = 'test-only'
    $env:RELEASE_CHANNEL = 'prerelease'
    $env:LLVY_SERVER_ORIGIN = 'https://test.invalid'
    $env:GITHUB_STEP_SUMMARY = ''
    $mock.commit = 'a' * 40
    $mock.remote = "$root/remote"
    $exe = 'llvy-uploader-1.2.3-rc.1-windows-x64.exe'
    Set-Content "$root/dist/$exe" 'test executable fixture'
    $hash = (Get-FileHash "$root/dist/$exe").Hash.ToLowerInvariant()
    Set-Content "$root/dist/SHA256SUMS.txt" "$hash  $exe"
    Set-Content "$root/dist/provenance.txt" "Source: $($mock.commit)`nVersion: 1.2.3-rc.1`nServer origin: $env:LLVY_SERVER_ORIGIN"
    Set-Content "$root/THIRD_PARTY_NOTICES.txt" 'test notices'
    Copy-Item "$root/THIRD_PARTY_NOTICES.txt" "$root/dist"
    Set-Content "$root/dist/build-info.txt" 'test build'
    Set-Content "$root/dist/release-evidence.json" '{}'
    # Go evidence validation has its own tests. Mock only external processes here.
    function go { $global:LASTEXITCODE = 0 }
    function gh {
        $global:LASTEXITCODE = 0
        if ($args[0] -eq 'api') { return $mock.remoteCommit }
        switch ($args[1]) {
            'view' {
                return (@{ tagName='v1.2.3-rc.1'; isDraft=$false; isPrerelease=$mock.prerelease; url='https://github.com/test/repo/releases/tag/v1.2.3-rc.1'; assets=@(Get-ChildItem $mock.remote | ForEach-Object { @{name=$_.Name} }) } | ConvertTo-Json -Depth 5 -Compress)
            }
            'upload' {
                $mock.uploads++
                $end = [array]::IndexOf($args, '--repo')
                foreach ($file in $args[3..($end-1)]) { Copy-Item $file (Join-Path $mock.remote $file) }
            }
            'download' {
                if ($mock.failDownload) { $global:LASTEXITCODE = 1; return }
                $file = $args[([array]::IndexOf($args, '--pattern')+1)]
                $dir = $args[([array]::IndexOf($args, '--dir')+1)]
                Copy-Item (Join-Path $mock.remote $file) $dir
                if ($mock.corruptDownload) { Add-Content (Join-Path $dir $file) 'corrupted' }
            }
            default { throw 'Unexpected gh call' }
        }
    }
    function Publish { & "$root/scripts/publish-release.ps1" -Version '1.2.3-rc.1' -Tag 'v1.2.3-rc.1' -Repository 'test/repo' -Commit $mock.commit }
    function Reject([string]$message) {
        $caught = $false
        try { Publish } catch { if ($_.Exception.Message -notlike "*$message*") { throw }; $caught = $true }
        if (-not $caught) { throw "Expected rejection: $message" }
    }
    $mock.remoteCommit = $mock.commit
    $mock.prerelease = $true
    $mock.uploads = 0
    $mock.corruptDownload = $false
    $mock.failDownload = $false
    Publish
    if ($mock.uploads -ne 1 -or @(Get-ChildItem $mock.remote).Count -ne 6) { throw 'Initial publication failed' }
    Publish
    if ($mock.uploads -ne 1) { throw 'Identical rerun uploaded again' }
    Remove-Item "$($mock.remote)/build-info.txt"
    Publish
    if ($mock.uploads -ne 2) { throw 'Partial publication was not repaired' }
    Add-Content "$($mock.remote)/$exe" 'conflict'
    Remove-Item "$($mock.remote)/build-info.txt"
    Reject 'Existing release asset differs'
    if ($mock.uploads -ne 2) { throw 'Conflicting rerun changed remote assets' }
    Copy-Item "$root/dist/$exe" $mock.remote -Force
    $mock.remoteCommit = 'b' * 40
    Reject 'Release tag commit mismatch'
    $mock.remoteCommit = $mock.commit
    $mock.prerelease = $false
    Reject 'Release metadata/channel mismatch'
    $mock.prerelease = $true
    $mock.failDownload = $true
    Reject 'Existing release download failed'
    $mock.failDownload = $false
    Remove-Item "$($mock.remote)/*"
    $mock.corruptDownload = $true
    Reject 'Published file checksum mismatch'
    Write-Output 'Release publication, identical/partial rerun, conflict, commit/channel, download failure and hash readback checks passed.'
} finally {
    foreach ($name in $names) { [Environment]::SetEnvironmentVariable($name, $saved[$name]) }
    Remove-Item $root -Recurse -Force -ErrorAction SilentlyContinue
}
