$INSTALL_DIR=$args[0]
$VERSION=$args[1]
# param([String]$VERSION="12.4.0",[String]$INSTALL_DIR="../nodejs_bin")

If(!(test-path $INSTALL_DIR))
{
      New-Item -ItemType Directory -Force -Path $INSTALL_DIR
}

$url = "https://nodejs.org/dist/v$VERSION/node-v$VERSION-x64.msi"

# i.e https://nodejs.org/dist/v10.16.0/node-v10.16.0-x64.msi
write-host "`n----------------------------"
write-host "  downloading node            "
write-host "----------------------------`n"
write-host "url : $url"

$filename = "node.msi"
$node_msi = "$INSTALL_DIR\$filename"
$start_time = Get-Date
$wc = New-Object System.Net.WebClient
$wc.DownloadFile($url, $node_msi)
write-Output "Download of $filename finished at: $((Get-Date).Subtract($start_time).Seconds) second(s)"

write-host "`n----------------------------"
write-host " installing node              "
write-host "----------------------------`n"

$node_msi = $node_msi.substring(2)

$INSTALL_DIR=[System.IO.Path]::GetFullPath($INSTALL_DIR)
write-host "installation directory: $INSTALL_DIR"

$params = '/i', "$node_msi",
          'INSTALLDIR="$INSTALL_DIR"',
          '/qn',
          '/norestart'
$p = Start-Process 'msiexec.exe' -ArgumentList $params -Wait -PassThru