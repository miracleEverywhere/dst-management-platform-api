$ErrorActionPreference = "Stop"

function Fail-Dmp {
    Write-Host "==>dmp@@ 更新失败 @@dmp<=="
    exit 1
}

trap {
    Write-Host $_
    Fail-Dmp
}

$DmpHome = $PSScriptRoot
$SteamDir = Join-Path $DmpHome "steamcmd"
$DstDir = Join-Path $DmpHome "dst"
$SteamExe = Join-Path $SteamDir "steamcmd.exe"

if (-not (Test-Path $SteamExe)) {
    Fail-Dmp
}

& $SteamExe "+login" "anonymous" "+force_install_dir" $DstDir "+app_update" "343050" "validate" "+quit"
if ($LASTEXITCODE -ne 0) {
    Fail-Dmp
}

Write-Host "==>dmp@@ 更新完成 @@dmp<=="
