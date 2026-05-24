$serviceName = "CortadoDaemon"
$binaryPath = Join-Path $env:LOCALAPPDATA "Cortado\bin\cortado-daemon.exe"
$logDir = Join-Path $env:LOCALAPPDATA "Cortado\logs"

New-Item -ItemType Directory -Force -Path $logDir | Out-Null

nssm install $serviceName $binaryPath
nssm set $serviceName AppDirectory (Split-Path $binaryPath)
nssm set $serviceName AppStdout (Join-Path $logDir "daemon.stdout.log")
nssm set $serviceName AppStderr (Join-Path $logDir "daemon.stderr.log")
nssm set $serviceName Start SERVICE_AUTO_START
nssm start $serviceName
