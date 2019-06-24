[cmdletbinding()]
$Install_Dir=$args[0]
$VERSION=$args[1]

$Install_Dir=[System.IO.Path]::GetFullPath($Install_Dir)

If(!(test-path $Install_Dir))
{
      New-Item -ItemType Directory -Force -Path $Install_Dir
}

function Say($str) {
    Write-Host "node-install: $str"
}

$url = "https://nodejs.org/dist/v$VERSION/node-v$VERSION-x64.msi"

# i.e https://nodejs.org/dist/v10.16.0/node-v10.16.0-x64.msi
# Say "----------------------------"
# Say "  downloading node          "
# Say "----------------------------"
# Say "url : $url"

$filename = "node.msi"
$node_msi = "$Install_Dir\$filename"
$start_time = Get-Date
$wc = New-Object System.Net.WebClient
$wc.DownloadFile($url, $node_msi)
# Say "Download of $filename finished at: $((Get-Date).Subtract($start_time).Seconds) second(s)"

# Say "---------------------------"
# Say "  installing node           "
# Say "---------------------------"

# Say $node_msi

# $Install_Dir = $Install_Dir.replace("\","/")

# Say "installation directory: $Install_Dir"

# $params = '/i', "$node_msi",
#           'INSTALLDIR="$Install_Dir"',
#           '/qn',
#           '/norestart'
#           # '/log $Install_Dir\install.log'

# $p = Start-Process 'msiexec.exe' -ArgumentList $params -PassThru -Wait

# Start-Process 'msiexec.exe' -ArgumentList '/i', "$node_msi", "INSTALLDIR=$Install_Dir", '/norestart' -Wait -PassThru

Start-Process -Wait 'msiexec.exe' -ArgumentList '/i', $node_msi, "INSTALLDIR=$Install_Dir", '/passive'