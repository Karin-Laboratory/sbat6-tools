# SBAT6 tools

Recovery and diagnostics helpers for user-owned SoftBank Air Terminal 6 research devices.

## Safety model

- Never write the active slot, shared/NV partitions, `misc`, `boot_para`, or eMMC boot areas.
- Complete and verify a whole-device backup before any slot repair.
- The recovery helper only runs while slot A is active and copies the documented A-only partition family to B.
- It verifies every copied partition with SHA-256 before reporting success.

## Included tool

`scripts/rebuild-b-from-live-a.sh` rebuilds the inactive B slot from a currently booted, known-good A slot. It is deliberately device-local and must be run as root on the terminal. It does not select B or reboot; validate the evidence and use the vendor slot-control interface separately.

## Not included

Credentials, device-specific backups, SSH private keys, cookies, LAN configuration, and authenticated RCE code are intentionally excluded.

## Status

Validated against SBAT6 firmware 1.00.22 on a user-owned laboratory unit. Test only on equipment you own or are authorized to administer.
