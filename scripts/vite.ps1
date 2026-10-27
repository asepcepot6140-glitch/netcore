# Dipanggil oleh air (pre_cmd/post_cmd) agar Vite ikut hidup dan mati bersama air.
#   vite.ps1 start : jalankan Vite bila belum jalan (aman dipanggil di setiap rebuild)
#   vite.ps1 stop  : hentikan Vite beserta proses turunannya
param(
    [Parameter(Mandatory)][ValidateSet('start', 'stop')][string]$Action
)

$root = Split-Path $PSScriptRoot -Parent
$pidFile = Join-Path $root 'tmp\vite.pid'
$port = 5173

function Get-ViteProcess {
    if (-not (Test-Path $pidFile)) { return $null }
    $id = [int](Get-Content $pidFile -ErrorAction SilentlyContinue)
    Get-Process -Id $id -ErrorAction SilentlyContinue
}

switch ($Action) {
    'start' {
        if (Get-ViteProcess) { return }
        if (Get-NetTCPConnection -State Listen -LocalPort $port -ErrorAction SilentlyContinue) {
            Write-Host "[vite] port $port sudah dipakai, Vite tidak dijalankan lagi."
            return
        }
        New-Item -ItemType Directory -Path (Split-Path $pidFile) -Force | Out-Null
        $vite = Start-Process -FilePath 'pnpm.cmd' -ArgumentList 'dev' `
            -WorkingDirectory (Join-Path $root 'frontend') -NoNewWindow -PassThru
        $vite.Id | Set-Content $pidFile
    }
    'stop' {
        $vite = Get-ViteProcess
        if ($vite) {
            # /T menutup seluruh proses turunan (pnpm -> node -> vite).
            taskkill /PID $vite.Id /T /F | Out-Null
        }
        Remove-Item $pidFile -Force -ErrorAction SilentlyContinue
    }
}
