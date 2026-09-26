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

## Not included

Credentials, device-specific backups, SSH private keys, cookies, LAN configuration, and authenticated RCE code are intentionally excluded.

## Status

Validated against SBAT6 firmware 1.00.22 on a user-owned laboratory unit. Test only on equipment you own or are authorized to administer.
