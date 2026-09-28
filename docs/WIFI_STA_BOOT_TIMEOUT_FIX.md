# Wi-Fi STA boot-timeout mitigation

The SBAT6 vendor `procd` reboots the device when its bring-up completion flag
has not been set before the compiled 120-second deadline. A locally added
`S98t6a_wifi5_sta` service originally performed interface discovery, two fixed
eight-second waits, association, DHCP, and firewall setup synchronously in the
OpenWrt boot sequence. Depending on radio and modem timing, that consumed the
remaining bring-up margin and produced intermittent reboot loops.

[`../scripts/t6a-wifi5-sta-async.sh`](../scripts/t6a-wifi5-sta-async.sh) keeps
the same STA procedure and management-host firewall rules but delegates them to
a background worker. The init `start()` method returns immediately, while a PID
file prevents duplicate workers and `stop()` terminates an unfinished worker.

The management STA uses a dedicated `udhcpc` hook which deliberately ignores
the offered default gateway and DNS servers. The vendor default DHCP hook
replaces every existing default route; when the management STA finishes after
WWAN setup, that behavior removes the cellular default route and sends carrier
DNS traffic toward the management LAN. The dedicated hook installs only the
address and connected management-LAN route, leaving WWAN as the Internet path.

## Verification on the laboratory unit

The original script and recovery archive were backed up under
`/data/t6a-wifi5-async-<timestamp>/`. The replacement was installed both in the
live overlay and in `sbat6-ssh-overlay.tgz`/`sysupgrade.tgz`. The archives were
expanded and compared with the live script before rebooting.

Two consecutive controlled Bank2 boots completed successfully:

| boot | management path | vendor completion | result |
|---|---:|---:|---|
| 1 | 115 seconds | about 46 seconds | `PROCD INIT SUCCESS` |
| 2 | 116 seconds | about 46 seconds | `PROCD INIT SUCCESS` |

In both tests `dhcp.lan.ignore=1` remained persistent, UDP port 67 was absent,
DNS on port 53 remained available, and the device stayed online beyond the old
failure deadline. The vendor PID 1 binary was not patched or disabled.
