param([string]$Version = '0.0.0-dev')
$ErrorActionPreference = 'Stop'
. (Join-Path $PSScriptRoot 'version.ps1')
$versionInfo = Get-UploaderVersion -Version $Version
Push-Location (Split-Path $PSScriptRoot -Parent)
try {
    $exePath = Join-Path ./dist $versionInfo.FileName
    $metadata = (Get-Item $exePath).VersionInfo
    if ($metadata.FileVersion -cne $Version -or $metadata.ProductVersion -cne $Version -or $metadata.OriginalFilename -cne $versionInfo.FileName) {
        throw 'Embedded Windows version does not match the release.'
    }
    go run ./scripts/inspect $exePath
    if ($LASTEXITCODE -ne 0) { throw 'PE inspection failed' }
    $process = Start-Process $exePath -ArgumentList '--self-test' -PassThru
    if (-not $process.WaitForExit(10000)) {
        $process.Kill()
        throw 'Self-test timed out'
    }
    if ($process.ExitCode -ne 0) { throw 'Self-test failed' }
    $config = Start-Process $exePath -ArgumentList '--check-config' -PassThru
    if (-not $config.WaitForExit(10000)) {
        $config.Kill()
        throw 'Configuration check timed out'
    }
    $expectedExit = if ([string]::IsNullOrEmpty($env:LLVY_SERVER_ORIGIN)) { 1 } else { 0 }
    if ($config.ExitCode -ne $expectedExit) { throw 'Embedded configuration check failed' }
} finally {
    Pop-Location
}
