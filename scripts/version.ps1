function Get-UploaderVersion {
    param([Parameter(Mandatory)][string]$Version)
    # Keep the same safe value in filenames, linker flags, PE resources and OCI tags.
    if ($Version.Length -gt 100 -or $Version -cnotmatch '\A(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(?:-([0-9A-Za-z-]+(?:\.[0-9A-Za-z-]+)*))?\z') {
        throw 'Expected X.Y.Z or X.Y.Z-prerelease, without build metadata.'
    }
    $parts = @($Matches[1], $Matches[2], $Matches[3])
    foreach ($part in $parts) {
        if ($part.Length -gt 5 -or [int]$part -gt 65535) { throw 'Windows version components must be between 0 and 65535.' }
    }
    if ($Matches[4]) {
        foreach ($identifier in $Matches[4].Split('.')) {
            if ($identifier -cmatch '\A0[0-9]+\z') { throw 'Numeric prerelease identifiers cannot have leading zeros.' }
        }
    }
    [pscustomobject]@{
        Version = $Version
        FileName = "llvy-uploader-$Version-windows-x64.exe"
        WindowsVersion = ($parts -join '.') + '.0'
    }
}

function Get-UploaderReleaseVersion {
    param([Parameter(Mandatory)][string]$Tag)
    if ($Tag -cnotmatch '\A(?:uploader-)?v(.+)\z') { throw 'Release tag must start with v or uploader-v.' }
    Get-UploaderVersion -Version $Matches[1]
}

function New-UploaderVersionResource {
    param([Parameter(Mandatory)]$VersionInfo, [Parameter(Mandatory)][string]$Source, [Parameter(Mandatory)][string]$Destination)
    $resource = Get-Content -Raw $Source | ConvertFrom-Json
    $version = $resource.RT_VERSION.'#1'.'0409'
    $version.fixed.file_version = $VersionInfo.WindowsVersion
    $version.fixed.product_version = $VersionInfo.WindowsVersion
    $version.info.'0409'.FileVersion = $VersionInfo.Version
    $version.info.'0409'.ProductVersion = $VersionInfo.Version
    $version.info.'0409'.OriginalFilename = $VersionInfo.FileName
    $version.info.'0409'.FileDescription = 'LLVY Uploader - unsigned development build'
    $resource.RT_MANIFEST.'#1'.'0409'.description = 'LLVY Uploader'
    $resource | ConvertTo-Json -Depth 20 | Set-Content -Encoding utf8NoBOM $Destination
}

function Assert-UploaderReleaseChannel {
    param([Parameter(Mandatory)][string]$Version, [Parameter(Mandatory)][string]$Channel)
    $info = Get-UploaderVersion -Version $Version
    if ($Channel -cnotin @('stable', 'prerelease')) { throw 'Unknown release channel' }
    if (($Channel -ceq 'prerelease') -ne $info.Version.Contains('-')) { throw 'Release channel must match the version suffix' }
}
