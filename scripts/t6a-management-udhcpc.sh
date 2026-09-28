#!/bin/sh

# DHCP hook for the management-only 5 GHz STA. Ignore offered router and DNS
# options so this path cannot replace the cellular Internet route/resolver.

[ -n "$interface" ] || exit 1

case "$1" in
    deconfig)
        ip rule del priority 101 2>/dev/null || true
        ip route flush table 113 2>/dev/null || true
        ip -4 addr flush dev "$interface"
        ;;
    bound|renew)
        [ -n "$ip" ] || exit 1
        ip -4 addr flush dev "$interface"
        ip addr add "$ip/${subnet:-255.255.255.0}" \
            broadcast "${broadcast:-+}" dev "$interface"
        ip rule del priority 101 2>/dev/null || true
        ip route flush table 113 2>/dev/null || true
        ip route add table 113 192.168.0.0/22 dev "$interface" src "$ip"
        ip rule add priority 101 from "$ip/32" lookup 113
        ;;
esac

exit 0
