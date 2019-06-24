# borrowed: https://github.com/PowerShell/vscode-powershell/blob/develop/scripts/Install-VSCode.ps1
# edited to support a custom install directory and fix an issue with ssl and
# simplify the script - we don't need update only a temp installation.

param([string]$InstallDir="$env:TEMP")

If(!(test-path $InstallDir))
{
      New-Item -ItemType Directory -Force -Path $InstallDir
}

$filename=-join($InstallDir, "\", "git-installer.exe")

[Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12

foreach ($asset in (Invoke-RestMethod https://api.github.com/repos/git-for-windows/git/releases/latest).assets) {
    if ($asset.name -match 'Git-\d*\.\d*\.\d*-64-bit\.exe') {
        $dlurl = $asset.browser_download_url
        $newver = $asset.name
    }
}

try {
    $ProgressPreference = 'SilentlyContinue'
    Write-Host "`nDownloading latest stable git..." -ForegroundColor Yellow
    Remove-Item -Force $filename -ErrorAction SilentlyContinue
    Invoke-WebRequest -Uri $dlurl -OutFile $filename

    Write-Host "`nInstalling git..." -ForegroundColor Yellow
    Start-Process -Wait $filename -ArgumentList /VERYSILENT, /DIR="$InstallDir", \FORCECLOSEAPPLICATIONS
    # or SILENT

    Write-Host "`nInstallation complete!`n`n" -ForegroundColor Green
}
finally {
    $ProgressPreference = 'Continue'
}

$s = get-process ssh-agent -ErrorAction SilentlyContinue
if ($s) {$true}