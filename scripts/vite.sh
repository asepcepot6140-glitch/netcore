#!/usr/bin/env bash
# Versi Linux/macOS dari vite.ps1. Dipanggil oleh air (pre_cmd/post_cmd).
#   vite.sh start : jalankan Vite bila belum jalan (aman dipanggil di setiap rebuild)
#   vite.sh stop  : hentikan Vite beserta proses turunannya
set -u

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
PID_FILE="$ROOT/tmp/vite.pid"
PORT=5173

is_running() {
  [[ -f "$PID_FILE" ]] && kill -0 "$(cat "$PID_FILE")" 2>/dev/null
}

port_in_use() {
  (exec 3<>"/dev/tcp/localhost/$PORT") 2>/dev/null
}

case "${1:-}" in
  start)
    is_running && exit 0
    if port_in_use; then
      echo "[vite] port $PORT sudah dipakai, Vite tidak dijalankan lagi."
      exit 0
    fi
    mkdir -p "$(dirname "$PID_FILE")"
    # Job control membuat proses latar punya process group sendiri (PGID = PID),
    # sehingga seluruh turunannya (pnpm -> node -> vite) bisa dimatikan sekaligus.
    set -m
    pnpm --dir "$ROOT/frontend" dev </dev/null &
    echo $! >"$PID_FILE"
    ;;
  stop)
    if is_running; then
      pgid="$(cat "$PID_FILE")"
      kill -TERM -- "-$pgid" 2>/dev/null
      for _ in 1 2 3 4 5; do
        kill -0 "$pgid" 2>/dev/null || break
        sleep 0.5
      done
      kill -KILL -- "-$pgid" 2>/dev/null
    fi
    rm -f "$PID_FILE"
    ;;
  *)
    echo "usage: $0 {start|stop}" >&2
    exit 2
    ;;
esac
