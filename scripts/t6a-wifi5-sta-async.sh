#!/bin/sh /etc/rc.common

START=98
STOP=10

WORKER_PID=/var/run/t6a-wifi5-worker.pid
WORKER_LOG=/tmp/t6a-wifi5-worker.log

setup_wpa_config() {
    lua -e 'package.path="/lib/wifi/?.lua;"..package.path; local m=require("mtkdat"); dofile("/lib/wifi/supplicant.lua"); local u=m.uci_load_wireless(); local v=m.get_uci_vif_by_vif_name(u,"apclii0"); local d=m.get_uci_dev_by_dev_name(u,"MT7990_1_2"); assert(v and d,"5 GHz profile missing"); supp_setup_vif(d,v)'
}

allow_management_host() {
    host="$1"
    iptables -C INPUT -i apclii0 -s "$host" -p tcp --dport 22 -j ACCEPT 2>/dev/null || \
        iptables -I INPUT 1 -i apclii0 -s "$host" -p tcp --dport 22 -j ACCEPT
}

start_worker() {
    trap 'rm -f "$WORKER_PID"' EXIT HUP INT TERM

    ip link set rai0 up
    # The STA and its parent radio share one channel. Keep the parent on the
    # non-DFS channel used by the selected home AP.
    mwctl phy phy1 set channel num=48 bw=80 >/tmp/t6a-wifi5-channel.log 2>&1 || true
    tries=0
    while [ ! -e /sys/class/net/apclii0 ]; do
        tries=$((tries + 1))
        [ "$tries" -ge 30 ] && {
            logger -t t6a_wifi5_sta '5 GHz STA VIF did not appear'
            return 1
        }
        sleep 1
    done

    setup_wpa_config || return 1
    ip link set apclii0 up
    wpa_cli -p /var/run/wpa_supplicant -i global interface_remove apclii0 >/dev/null 2>&1 || true
    wpa_cli -p /var/run/wpa_supplicant -i global interface_add apclii0 /var/run/wpa_supplicant/wpa_supplicant-apclii0.conf >/tmp/t6a-wifi5-wpa.log 2>&1 &
    sleep 8
    wpa_cli -i apclii0 scan >/tmp/t6a-wifi5-scan.log 2>&1 || true
    sleep 8
    wpa_cli -i apclii0 reconnect >/dev/null 2>&1 || true
    # This STA is a management-only path. The vendor default DHCP hook installs
    # the offered router as the system default and deletes the WWAN default.
    /sbin/udhcpc -i apclii0 -b -s /usr/sbin/t6a-management-udhcpc \
        -p /var/run/t6a-wifi5-udhcpc.pid >/tmp/t6a-wifi5-dhcp.log 2>&1
    allow_management_host 192.168.0.26
    allow_management_host 192.168.0.169
}

start() {
    if [ -s "$WORKER_PID" ] && kill -0 "$(cat "$WORKER_PID")" 2>/dev/null; then
        return 0
    fi

    start_worker >"$WORKER_LOG" 2>&1 &
    echo $! >"$WORKER_PID"
    logger -t t6a_wifi5_sta "STA initialization delegated to worker PID $!"
    return 0
}

stop() {
    if [ -s "$WORKER_PID" ]; then
        kill "$(cat "$WORKER_PID")" 2>/dev/null || true
        rm -f "$WORKER_PID"
    fi
    [ -s /var/run/t6a-wifi5-udhcpc.pid ] && kill "$(cat /var/run/t6a-wifi5-udhcpc.pid)" 2>/dev/null || true
    wpa_cli -p /var/run/wpa_supplicant -i global interface_remove apclii0 >/dev/null 2>&1 || true
    ip link set apclii0 down 2>/dev/null || true
    ip link set rai0 down 2>/dev/null || true
}
