$ErrorActionPreference = "Stop"

function Fail-Dmp {
    Write-Host "==>dmp@@ 安装失败 @@dmp<=="
    exit 1
}

trap {
    Write-Host $_
    Fail-Dmp
}

$DmpHome = $PSScriptRoot
$SteamDir = Join-Path $DmpHome "steamcmd"
$DstDir = Join-Path $DmpHome "dst"
$ZipPath = Join-Path $DmpHome "steamcmd.zip"
$SteamExe = Join-Path $SteamDir "steamcmd.exe"

New-Item -ItemType Directory -Force -Path $SteamDir | Out-Null
New-Item -ItemType Directory -Force -Path $DstDir | Out-Null

if (-not (Test-Path $SteamExe)) {
    if (Test-Path $ZipPath) {
        Remove-Item -LiteralPath $ZipPath -Force
    }
    Invoke-WebRequest -Uri "https://steamcdn-a.akamaihd.net/client/installer/steamcmd.zip" -OutFile $ZipPath
    Expand-Archive -LiteralPath $ZipPath -DestinationPath $SteamDir -Force
    Remove-Item -LiteralPath $ZipPath -Force
}

& $SteamExe "+force_install_dir" $DstDir "+login" "anonymous" "+app_update" "343050" "validate" "+quit"
if ($LASTEXITCODE -ne 0) {
    & $SteamExe "+force_install_dir" $DstDir "+login" "anonymous" "+app_update" "343050" "validate" "+quit"
}
if ($LASTEXITCODE -ne 0) {
    Fail-Dmp
}

$Manifest = Join-Path $DstDir "steamapps\appmanifest_343050.acf"
if (Test-Path $Manifest) {
    Remove-Item -LiteralPath $Manifest -Force
}

Write-Host "==>dmp@@ 安装完成 @@dmp<=="
