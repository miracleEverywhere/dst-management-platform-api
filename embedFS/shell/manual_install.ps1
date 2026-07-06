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
$Manifest = Join-Path $DstDir "steamapps\appmanifest_343050.acf"
$ServerExe = Join-Path $DstDir "bin64\dontstarve_dedicated_server_nullrenderer_x64.exe"

function Get-AcfValue {
    param(
        [string]$Path,
        [string]$Key
    )
    if (-not (Test-Path $Path)) {
        return $null
    }
    $line = Select-String -LiteralPath $Path -Pattern "`"$Key`"\s+`"([^`"]+)`"" | Select-Object -First 1
    if (-not $line) {
        return $null
    }
    return $line.Matches[0].Groups[1].Value
}

function Wait-SteamCmd {
    while ($true) {
        $running = Get-Process steamcmd -ErrorAction SilentlyContinue | Where-Object { $_.Path -eq $SteamExe }
        if (-not $running) {
            break
        }

        $downloaded = [int64](Get-AcfValue -Path $Manifest -Key "BytesDownloaded")
        $total = [int64](Get-AcfValue -Path $Manifest -Key "BytesToDownload")
        if ($total -gt 0) {
            $percent = [math]::Round(($downloaded * 100.0 / $total), 1)
            Write-Host "SteamCMD仍在运行，下载进度 $percent% ($downloaded / $total)"
        } else {
            Write-Host "SteamCMD仍在运行，正在下载或校验..."
        }
        Start-Sleep -Seconds 15
    }
}

function Test-DstInstalled {
    if (-not (Test-Path $ServerExe)) {
        return $false
    }
    if (-not (Test-Path $Manifest)) {
        return $true
    }
    $buildID = [int64](Get-AcfValue -Path $Manifest -Key "buildid")
    return $buildID -gt 0
}

function Invoke-DstInstall {
    Wait-SteamCmd
    if (Test-DstInstalled) {
        return
    }

    for ($attempt = 1; $attempt -le 3; $attempt++) {
        & $SteamExe "+force_install_dir" $DstDir "+login" "anonymous" "+app_update" "343050" "validate" "+quit"
        $exitCode = $LASTEXITCODE
        Wait-SteamCmd

        if (Test-DstInstalled) {
            return
        }

        Write-Host "SteamCMD本次返回码 $exitCode，安装尚未完成，准备重试..."
        Start-Sleep -Seconds 3
    }

    Fail-Dmp
}

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

Invoke-DstInstall

Write-Host "==>dmp@@ 安装完成 @@dmp<=="
