#!/usr/bin/env bash
# Deploy the screensaver hack to a Push 3 over SSH, standalone (this hack lives
# outside the push-hack monorepo, so scripts/install.sh there won't pick it up).
#
# Requires: push-manager + push-display already installed and running on the
# device — the screensaver draws through push-manager's display API.
set -euo pipefail

HOST="${PUSH_HOST:-push.local}"
ID="screensaver"
SVC="push-hack-${ID}"
REMOTE_DIR="/data/push-hack/hacks/${ID}"
LOG="/data/push-hack/logs/${ID}.log"
CONFIG="${REMOTE_DIR}/hack.json"
BIN="${REMOTE_DIR}/${ID}"

cd "$(dirname "$0")"
echo "==> building"
make >/dev/null

echo "==> stopping ${SVC} (if running)"
ssh -n "root@${HOST}" "/etc/init.d/${SVC} stop 2>/dev/null || true"

echo "==> copying binary + config to ${HOST}:${REMOTE_DIR}"
ssh -n "ableton@${HOST}" "mkdir -p ${REMOTE_DIR}"
scp -q build/${ID} "ableton@${HOST}:${BIN}"
scp -q hack.json  "ableton@${HOST}:${CONFIG}"
ssh -n "ableton@${HOST}" "chmod +x ${BIN}"

echo "==> installing init.d service"
ssh "root@${HOST}" "cat > /etc/init.d/${SVC}" <<EOF
#!/bin/sh
### BEGIN INIT INFO
# Provides:          ${SVC}
# Required-Start:    \$network \$local_fs
# Required-Stop:     \$network
# Default-Start:     2 3 4 5
# Default-Stop:      0 1 6
# Short-Description: push-hack: ${ID}
### END INIT INFO
DAEMON="${BIN}"
PIDFILE="/var/run/${SVC}.pid"
LOGFILE="${LOG}"
CONFIG="${CONFIG}"
[ -x "\$DAEMON" ] || exit 0
start() {
    if [ -f "\$PIDFILE" ] && kill -0 \$(cat "\$PIDFILE") 2>/dev/null; then
        echo "${SVC} already running"; return 0
    fi
    mkdir -p "\$(dirname "\$LOGFILE")"
    nice -n 19 "\$DAEMON" -config "\$CONFIG" >> "\$LOGFILE" 2>&1 &
    echo \$! > "\$PIDFILE"
    echo "${SVC} started (PID \$(cat \$PIDFILE))"
}
stop() {
    [ -f "\$PIDFILE" ] && kill \$(cat "\$PIDFILE") 2>/dev/null || true
    rm -f "\$PIDFILE"
    echo "${SVC} stopped"
}
case "\$1" in
    start) start ;;
    stop) stop ;;
    restart) stop; sleep 1; start ;;
    status) [ -f "\$PIDFILE" ] && kill -0 \$(cat "\$PIDFILE") 2>/dev/null && echo running || echo stopped ;;
    *) echo "Usage: \$0 {start|stop|restart|status}"; exit 1 ;;
esac
EOF

ssh -n "root@${HOST}" "chmod +x /etc/init.d/${SVC}
for n in 2 3 4 5; do ln -sf /etc/init.d/${SVC} /etc/rc\${n}.d/S99${SVC} 2>/dev/null || true; done
for n in 0 1 6; do ln -sf /etc/init.d/${SVC} /etc/rc\${n}.d/K01${SVC} 2>/dev/null || true; done
/etc/init.d/${SVC} start"

echo "==> done. config API: http://${HOST}:7706/  (configure via push-manager Shadow UI -> SAVER tab)"
