# SBAT6 tools

Recovery and diagnostics helpers for user-owned SoftBank Air Terminal 6 research devices.

## Safety model

- Never write the active slot, shared/NV partitions, `misc`, `boot_para`, or eMMC boot areas.
- Complete and verify a whole-device backup before any slot repair.
- The recovery helper only runs while slot A is active and copies the documented A-only partition family to B.
- It verifies every copied partition with SHA-256 before reporting success.

## Included tools

`scripts/rebuild-b-from-live-a.sh` rebuilds the inactive B slot from a currently booted, known-good A slot. It is deliberately device-local and must be run as root on the terminal. It does not select B or reboot; validate the evidence and use the vendor slot-control interface separately.

`scripts/analyze-vendor-rootfs.py` performs a read-only static inventory of two
extracted SBAT6 root filesystems. It records init services, ELF executables,
package ownership, hashes, and strings related to reboot, rollback, watchdog,
upgrade, and recovery behavior.

`scripts/t6a-wifi5-sta-async.sh` is the verified non-blocking replacement for
the local 5 GHz management-STA init script. It delegates association and DHCP
work to a background worker so vendor boot completion is not held against the
120-second `procd` deadline.

`scripts/t6a-management-udhcpc.sh` keeps the management STA from replacing the
cellular default route and installs its source-specific reply path.

`scripts/t6recover` and `scripts/rc.local` keep the wired management address on
`br-lan` rather than its enslaved Ethernet port. The checked-in private-address
defaults describe the laboratory topology and must be adapted before use on a
different LAN.

`scripts/t6diag24` records a bounded 24-hour rotation of network, Wi-Fi,
bridge, firewall, modem, and selected system events under `/data/t6diag24`.

Do not put long sleeps, association loops, or DHCP waits directly in a
boot-critical synchronous init path. The original `home24fix` integration did
so and produced a reproducible 106–107 second reset loop. See
[`docs/BOOT-LOOP-107S-FAILURE.md`](docs/BOOT-LOOP-107S-FAILURE.md) and the
[`known-non-working` index](known-non-working/README.md).


## Internal ESP32-C3

The internal ESP32 is documented as a first-class subsystem under
[`esp32/`](esp32/README.md). The audited unit contains an ESP32-C3 MINI-1
running ESP-AT 3.1.0.0-dev over UART1 at 115200 8N1, owned by the vendor
`esp32_uart` process.

The initial shell wrapper remains short-query-only, and
[`esp32/broker/`](esp32/broker/README.md) now contains a phase-1 Go
single-owner broker seed. It serializes `ping`/`info` requests over a Unix
socket while continuing to use the vendor frontend.

Do not open `/dev/ttyS1` directly in normal operation. A 256-byte response
buffer over-read was identified in the vendor UART bridge. The broker fixes
ownership/concurrency, not that defect, so Wi-Fi/BLE scans and other long
responses remain disabled.

## Static audit snapshot

[`audits/sbat6a-20260925/VENDOR_DAEMON_AUDIT.md`](audits/sbat6a-20260925/VENDOR_DAEMON_AUDIT.md)
contains the vendor-daemon, reboot, and A/B rollback audit of the 2026-09-25
backup. The accompanying `generated/*.tsv` files are the complete
machine-readable inventories. Raw firmware, persistent-data images, runtime
logs, and credentials are not included.

## Network reconstruction notes

[`docs/NETWORK_DEBRIDGE_WIFI_STA.md`](docs/NETWORK_DEBRIDGE_WIFI_STA.md) documents a verified, safety-first method to establish a Wi-Fi STA management path and then remove the Linux LAN bridge/VLAN devices in favor of raw `eth0`. The document intentionally uses placeholders and documentation-only addresses; it contains no credentials, keys, device addresses, or RCE implementation.

[`docs/PERSISTENT_OVERLAY_RESTORE.md`](docs/PERSISTENT_OVERLAY_RESTORE.md)
documents the exact next-boot restore path used on the laboratory unit: the
vendor `S001restore` service consumes `/data/sysupgrade.tgz` once during boot,
while a locally installed root cron entry recreates that next-boot archive
every minute from the canonical `/data/sbat6-ssh-overlay.tgz`. It also explains
why changing only the live overlay is not persistent and gives a safe,
atomic update and verification procedure.

[`docs/BOOT_AND_CONNECTIVITY_FAILURES.md`](docs/BOOT_AND_CONNECTIVITY_FAILURES.md)
collects the confirmed causes of reboot loops, apparent boot hangs, management
loss, and configuration rollback. It separates proven reset causes from
connectivity failures that only resemble a frozen or rebooting device.

[`docs/CELLULAR_POVO_AND_JCI_RESULTS_20260927.md`](docs/CELLULAR_POVO_AND_JCI_RESULTS_20260927.md)
records the successful povo/KDDI LTE baseline and the reproducible Japan
Communications/docomo EPS-registration failure. It includes serving-cell and
band conditions, controls already performed, SIM-file results, and the limits
of the remaining modem-firmware/SBP hypothesis without publishing subscriber
credentials or complete device identifiers.

## Not included

Credentials, device-specific backups, SSH private keys, cookies, wireless
secrets, subscriber identifiers, and authenticated RCE code are intentionally
excluded.

## Status

Validated against SBAT6 firmware 1.00.22 on a user-owned laboratory unit. Test only on equipment you own or are authorized to administer.

Release-oriented changes are summarized in [`CHANGELOG.md`](CHANGELOG.md).
