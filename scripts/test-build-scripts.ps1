$ErrorActionPreference = 'Stop'
. (Join-Path $PSScriptRoot 'version.ps1')
foreach ($script in Get-ChildItem $PSScriptRoot -Filter '*.ps1') {
    $parseErrors = $null
    [void][System.Management.Automation.Language.Parser]::ParseFile($script.FullName, [ref]$null, [ref]$parseErrors)
    if ($parseErrors.Count) { throw "PowerShell syntax errors in $($script.Name): $parseErrors" }
}
foreach ($tag in @('v1.2.3', 'uploader-v1.2.3', 'v1.2.3-rc.1', 'v0.0.0', 'v65535.65535.65535')) {
    $info = Get-UploaderReleaseVersion -Tag $tag
    $expected = $tag -creplace '^(uploader-)?v', ''
    if ($info.Version -cne $expected -or $info.FileName -cne "llvy-uploader-$expected-windows-x64.exe") { throw "Wrong version for $tag" }
}
foreach ($tag in @('1.2.3', 'V1.2.3', 'v01.2.3', 'v1.2', 'v1.2.3+meta', 'v65536.0.0', 'v999999999999.0.0', 'v1.2.3-01', 'v1.2.3-rc.01', "v1.2.3`n", 'v1.2.3/../oops', 'v1.2.3 -X main.version=bad', 'v1.2.3-$(whoami)', ('v1.2.3-' + 'x' * 100))) {
    $rejected = $false
    try { Get-UploaderReleaseVersion -Tag $tag | Out-Null } catch { $rejected = $true }
    if (-not $rejected) { throw "Accepted invalid release tag: $tag" }
}
$tempResource = [IO.Path]::GetTempFileName()
try {
    $info = Get-UploaderVersion -Version '1.2.3-rc.1'
    New-UploaderVersionResource -VersionInfo $info -Source (Join-Path $PSScriptRoot '../build/windows/winres.json') -Destination $tempResource
    $resource = Get-Content -Raw $tempResource | ConvertFrom-Json
    $version = $resource.RT_VERSION.'#1'.'0409'
    if ($version.fixed.file_version -ne '1.2.3.0' -or $version.fixed.product_version -ne '1.2.3.0' -or $version.info.'0409'.ProductVersion -ne '1.2.3-rc.1' -or $version.info.'0409'.FileVersion -ne '1.2.3-rc.1' -or $version.info.'0409'.OriginalFilename -ne $info.FileName) { throw 'Resource version mismatch' }
    if ($resource.RT_MANIFEST.'#1'.'0409'.'execution-level' -ne 'as invoker') { throw 'Manifest changed unexpectedly' }
} finally {
    Remove-Item $tempResource
}
Write-Output 'Release version, filenames, resources and PowerShell syntax checks passed.'

Assert-UploaderReleaseChannel -Version '1.2.3' -Channel 'stable'
Assert-UploaderReleaseChannel -Version '1.2.3-rc.1' -Channel 'prerelease'
foreach ($pair in @(@('1.2.3', 'prerelease'), @('1.2.3-rc.1', 'stable'), @('1.2.3', 'unknown'))) {
    $rejected = $false
    try { Assert-UploaderReleaseChannel -Version $pair[0] -Channel $pair[1] } catch { $rejected = $true }
    if (-not $rejected) { throw 'Accepted mismatched release channel' }
}
