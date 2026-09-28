# SBAT6A boot and connectivity failure modes

This document collects failure modes confirmed on the laboratory SBAT6A. It
distinguishes an actual system reset from a boot stopped before userspace and
from loss of all remote management paths. Those cases look similar from the
network but require different recovery actions.

## Quick classification

| Symptom | Confirmed cause | Distinguishing evidence |
|---|---|---|
| Reset roughly every 130 seconds | Vendor `procd` 120-second bring-up deadline | `REARCH MAXIMUM TIMEOUT` and `Restarting the system due to incomplete bring-up process.` |
| Kernel is alive but no normal services start | Kernel command line ends with `init=/bin/sh` | PID 1 is `/bin/sh`; `/proc` and `/sys` may initially be unmounted |
| Ethernet carrier remains up but SSH addresses disappear after Wi-Fi work | Whole-device `wifi reload` tears down all radios and dependent network state | Link carrier can remain present while management STA, IP, route, or bridge state is gone |
| A change works until reboot, then old settings return | Canonical seed archive was not updated | Boot-time `S001restore` extracted the older `/data/sysupgrade.tgz` |
| 6 GHz SSID is visible but authentication never completes | AP interface was raised manually without hostapd registration | Beacon is visible, but hostapd has no interface socket and reports no enabled BSS |
| WPA3 succeeds but the client remains at “obtaining IP address” | AP and Ethernet are not bridged, or bridge netfilter drops the DHCP reply | hostapd shows `AUTHORIZED`; DHCP request reaches upstream, but OFFER/ACK does not return |

## Confirmed reboot loop: vendor bring-up deadline

The modified vendor `/sbin/procd` starts a 120,000 ms timer. If its internal
bring-up-complete flag is still false at expiry, it logs the deadline failure
and invokes a kernel reboot. Shutdown and boot time make the observed cycle
approximately 130 seconds.

The detailed disassembly and retained-log evidence are in
[`../audits/sbat6a-20260925/VENDOR_DAEMON_AUDIT.md`](../audits/sbat6a-20260925/VENDOR_DAEMON_AUDIT.md).

One locally introduced trigger was a synchronous management-STA init script.
Association, fixed waits, DHCP, and firewall work consumed the remaining boot
budget. Moving that work into a background worker allowed `procd` bring-up to
complete in about 46 seconds. See
[`WIFI_STA_BOOT_TIMEOUT_FIX.md`](WIFI_STA_BOOT_TIMEOUT_FIX.md).

Do not attribute every reset to this path. The firmware also contains health,
modem, thermal, kernel-hang, hardware-watchdog, UI, and FOTA reset paths. Use
the retained reason and emergency logs to distinguish them.

## Apparent boot freeze: `init=/bin/sh`

A recovery boot can append `init=/bin/sh` to the kernel command line. The
kernel is healthy, but PID 1 becomes an interactive shell instead of
`/etc/preinit` and `/sbin/procd`. Network configuration, Dropbear, Wi-Fi,
modem services, and vendor bring-up therefore never start. From the network it
looks like a frozen unit even though UART presents a BusyBox prompt.

Confirm with:

```sh
mount -t proc proc /proc 2>/dev/null || true
mount -t sysfs sysfs /sys 2>/dev/null || true
cat /proc/cmdline
ps w
```

Expected failure evidence is a final `init=/bin/sh` and PID 1 `/bin/sh`.
For a one-time continuation, PID 1 can hand control to normal early userspace:

```sh
exec /etc/preinit
```

Input over this UART must be paced; sending a whole command at once can lose
characters and execute a malformed recovery command. After a normal reboot,
verify that the effective command line no longer ends in `init=/bin/sh`.

## Management loss after Wi-Fi reload

The vendor `/sbin/wifi` wrapper can operate on one MediaTek device, but its
device identifiers use dots, for example `MT7990.1.3`. UCI section names use
underscores, for example `MT7990_1_3`. Passing the UCI-style name to `wifi`
does not select the intended radio. Escalating to an unqualified `wifi reload`
reloads all MediaTek radios and can remove the management STA together with
dependent IP, route, bridge, and firewall state.

Before a radio operation:

1. obtain the runtime device names from the MediaTek L1 profile/API;
2. retain an independent UART or wired management path;
3. prefer `wifi up <runtime-device>` when only hostapd registration is needed;
4. do not use a whole-device reload merely because a targeted command had no
   visible effect; and
5. verify both interface state and the authentication daemon.

For the verified 6 GHz radio, the runtime identifier was `MT7990.1.3`, the
primary AP interface was `rax0`, and hostapd had to report `state=ENABLED`.
The secondary `rax1` interface could not be created on this driver build
(`useCut 1 > supportNum 0`), so a configured UCI section alone did not prove
that the interface existed.

## Boot-time configuration rollback

The vendor restore service does not periodically reset the running system.
It performs a one-shot extraction at boot. A local cron job prepares the next
boot archive from a canonical seed. If that seed still contains old network or
wireless files, a correct live change is replaced on the next boot.

The complete data flow and atomic update procedure are documented in
[`PERSISTENT_OVERLAY_RESTORE.md`](PERSISTENT_OVERLAY_RESTORE.md).

## Management STA lost by vendor dynamic channel selection

The vendor `kn_dcsd` daemon can control the 5 GHz and 6 GHz radios
independently.  Enabling 5 GHz DCS is unsafe when `apclii0` is used as the
management STA on the same MT7990 radio.  A channel or DFS transition
reinitializes every interface on that radio, deliberately sends a deauth to
the upstream AP, and does not always reconnect the STA.

The captured failure on 2026-09-27 was unambiguous:

```text
DfsDedicatedInBandSetChannel: Channel(104)
ap_phy_rrm_init_byRf: apclii0 Send DeAuth to <upstream-ap-bssid>
cntl_disconnect_request: caller:ap_phy_rrm_init_byRf, reason=8
apclii0: NO-CARRIER
```

The STA remained `DISCONNECTED` until power cycling.  DHCP was not involved;
the link carrier disappeared first.  Preserve 6 GHz channel management while
disabling DCS on the 5 GHz management radio and pinning it to the upstream AP's
channel (48 in this deployment):

```sh
uci set dcsd.cbs.5g_enabled=0
uci set wireless.MT7990_1_2.channel=48
uci set wireless.MT7990_1_2.kn_channel=48
uci commit dcsd
uci commit wireless
/etc/init.d/kn_dcsd restart
```

Keep `dcsd.cbs.6g_enabled=1` when 6 GHz automatic selection is still wanted.
Copy both `/etc/config/dcsd` and `/etc/config/wireless` into the canonical
overlay archive or the boot restore path will undo the fix.

## WPA3 works but DHCP does not

An `AUTHORIZED` station proves authentication, not LAN service. With a raw
Ethernet LAN, adding a wireless interface to the non-bridge device fails with
`Operation not supported`. The AP must share an actual bridge with Ethernet.

Even after both ports show `state forwarding`, bridge netfilter can pass DHCP
frames into the IP firewall. A request may reach the upstream DHCP server while
the reply is rejected on the return path. On the verified deployment, a
late-start helper applied the intended pure-L2 policy after vendor startup:

```text
net.bridge.bridge-nf-call-iptables=0
net.bridge.bridge-nf-call-ip6tables=0
net.bridge.bridge-nf-call-arptables=0
```

The helper itself must be included in the canonical seed archive because a
vendor startup stage can restore the sysctl values later than `/etc/sysctl.conf`
is processed.

## Wired management replies lost with the Wi-Fi STA

Do not assign the management address to both `eth0` and `br-lan`.  An address
on an enslaved bridge port causes duplicate connected routes and ARP ambiguity.
Also ensure the wired bridge prefix covers the actual management LAN.  In the
verified deployment, clients use `192.168.0.0/22`; configuring the bridge as
`192.168.3.10/24` made replies to `192.168.0.x` follow the more-specific route
through `apclii0`.  When DCS disconnected that STA, Ethernet still answered
ARP but ping and SSH replies were sent into the dead wireless interface.

The stable layout is:

```text
br-lan   192.168.3.10/22
eth0     no IP address; bridge port only
apclii0  independent management address
```

Both the UCI `network.lan.netmask` and late recovery scripts must agree with
this layout.  Otherwise a later boot stage can silently recreate the duplicate
address.  Because both management interfaces reach the same `/22`, add
source-specific routes as well: traffic sourced by `192.168.3.10` must use
`br-lan`, while traffic sourced by `192.168.3.13` must use `apclii0`.  Without
those rules, the main table can still return wired replies through a failed
Wi-Fi STA.

## Completion criteria

A recovery is not complete until all relevant checks pass:

- `System bring-up completed.` is present and no deadline reset follows;
- PID 1 is `/sbin/procd`;
- wired and management-Wi-Fi addresses return;
- the expected AP is owned by hostapd and reports `ENABLED`;
- the client reports `AUTH`, `ASSOC`, `AUTHORIZED`, and `MFP` for WPA3;
- bridge ports are forwarding and bidirectional client traffic succeeds;
- the device-local DHCP server remains disabled when an upstream server is
  intended; and
- two controlled boots preserve the configuration without a reboot loop.
