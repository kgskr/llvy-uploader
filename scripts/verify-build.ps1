$ErrorActionPreference = 'Stop'
Push-Location (Split-Path $PSScriptRoot -Parent)
try {
    go run ./scripts/inspect ./dist/llvy-uploader.exe
    if ($LASTEXITCODE -ne 0) { throw 'PE inspection failed' }
    $process = Start-Process ./dist/llvy-uploader.exe -ArgumentList '--self-test' -PassThru
    if (-not $process.WaitForExit(10000)) {
        $process.Kill()
        throw 'Self-test timed out'
    }
    if ($process.ExitCode -ne 0) { throw 'Self-test failed' }
    $config = Start-Process ./dist/llvy-uploader.exe -ArgumentList '--check-config' -PassThru
    if (-not $config.WaitForExit(10000)) {
        $config.Kill()
        throw 'Configuration check timed out'
    }
    $expectedExit = if ([string]::IsNullOrEmpty($env:LLVY_SERVER_ORIGIN)) { 1 } else { 0 }
    if ($config.ExitCode -ne $expectedExit) { throw 'Embedded configuration check failed' }
} finally {
    Pop-Location
}
