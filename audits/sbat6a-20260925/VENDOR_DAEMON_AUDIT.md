# SBAT6A vendor daemon / reboot / rollback static audit

## Scope and baseline

This audit uses the USB backup under `sbat6a-20260925` as immutable input.  The
root filesystems are `mmcblk0p26` (Bank1/A) and `mmcblk0p39` (Bank2/B).  The
`misc` boot-control data is in `mmcblk0p1`; persistent runtime data is the later
`post-management-wwan-20260926/mmcblk0p43.img` copy.

Bank1 reports KnOS `1.01.44`; Bank2 reports `1.01.52`.  The captured boot-control
bytes select Bank2 and are:

| slot | priority | retry | successful | update type |
|---|---:|---:|---:|---:|
| A / Bank1 | `0x0e` | `0x00` | `0x00` | `0x00` |
| B / Bank2 | `0x0f` | `0x00` | `0x01` | `0x00` |

The complete machine-readable inventories are in `generated/services.tsv`,
`generated/executables.tsv`, and `generated/power_evidence.tsv`.  They cover 194
init-script instances (both slots), 384 ELF executables, and all matching
reboot/reset/rollback/watchdog strings.  Bank2 contains 94 enabled init services;
34 are classified as KeyONet, Quectel, MediaTek, CCCI/MIPC, or device-specific
vendor services.  Stock OpenWrt services were retained in the full inventory so
that no startup path was silently excluded.

The copied latest `user_data` image was verified against its USB-side SHA-256:
`5be710d278c291ff23b1e40ded541d0f7dc2905af6d6141f7d5b50cc69d7d72c`.

## Highest-confidence findings

### 1. The approximately 130-second reboot loop is the vendor `procd` bring-up deadline

Both banks contain a modified `/sbin/procd`.  Its AArch64 code arms a uloop timer
with the literal value `120000` milliseconds.  If its internal
`quec_procd_init_done_flag` is still false when that timer fires, the callback:

1. logs `REARCH MAXIMUM TIMEOUT!!!ERROR!!!REBOOT NOW!!!`;
2. logs `Restarting the system due to incomplete bring-up process.`; and
3. invokes the Linux reboot operation.

The 120-second deadline plus shutdown and boot time matches the observed roughly
130-second cycle.  The success path logs `PROCD INIT SUCCESS!!!`, creates
`/var/knos/knos_done`, and records `System bring-up completed.`  Selecting the
newer, already-successful Bank2 explains why power cycling recovered stable SSH.
This conclusion is based on code, not only on suggestive strings.

The latest `user_data` image independently confirms this path at runtime.  Its
two retained emergency logs contain **125** `Restarting the system due to
incomplete bring-up process.` events and 45 successful `System bring-up
completed.` events.  Recent examples are timeout resets at 2026-09-26 03:59:08
and 09:10:39, followed by successful bring-up at 04:01:31 and 09:12:22.  This
rules out the hardware watchdog, thermal protection, modem watchdog, and the
SSH process itself as the cause of that particular reboot loop.

### 2. The A/B update path is “write inactive slot, try it three times, then commit”

`/lib/upgrade/bootctrl.sh` stores boot metadata at byte 2048 of `misc`.  Each slot
has priority, retry, successful, and update-type bytes.  During an upgrade it:

1. writes images to the inactive bank;
2. clones non-present image types from the active bank;
3. gives the target priority `0x0f`, the old bank `0x0e`;
4. gives the target retry count `0x03` and successful=`0`; and
5. reboots forcibly after the stage-2 writer completes.

LK contains and uses `slot-retry-count`, `slot-successful`, `slot-unbootable`,
`bootctrl_get_current_slot`, and `bootctrl_switch_slot`.  Therefore fallback for
failures before userspace declares success is implemented in the bootloader.

### 3. The userspace “restore the other bank” code is broken in both shipped banks

`/etc/init.d/zmtk_boot_done` calls `ab_image_sync "_a"` or `"_b"` to clone a
working bank over the failed/opposite bank.  No definition or executable named
`ab_image_sync` exists anywhere in either root filesystem.  The script does not
check this failure before proceeding to update boot-control bytes.  As a result:

- bootloader retry/fallback remains available before the S99 boot-done stage;
- the advertised userspace bank restoration/cloning does **not** run as written;
- metadata may be marked successful even though the intended bank copy failed.

There is a second defect in `ql_ab_status_check`: `current_slot` is declared
`local` in its caller but referenced as though global inside the callee.  That
secondary recovery check consequently takes its `current_slot error` branch.

`/sbin/quec_bootctrl -s` also tests the slot-A argument before writing slot-B
successful/update-type values.  Its `-F` force-rollback mode is simpler and only
swaps priorities; it does not reboot by itself.

### 4. Persistent-data mount failure can erase all `user_data`

`/etc/init.d/0mount_all` runs ext4 checks at every boot.  If `user_data` is not
recognized as ext4, or if recovery mounting still fails, it executes
`mkfs.ext4 -F /dev/block/user_data` and remounts the new filesystem.  There is no
backup, operator prompt, or preservation attempt after that decision.  The same
script can format several smaller vendor partitions when their filesystem type
is unexpected.

## Reboot, reset, recovery, and power-control paths

| Trigger / owner | Default state and condition | Action | Evidence confidence |
|---|---|---|---|
| vendor `procd` | Always present; bring-up not complete in 120,000 ms | kernel reboot; explains observed loop | confirmed by disassembly |
| `kn_htchkd`: CPU | enabled; 5 s samples, configured thresholds 70/85/90 and over-count 60 | log `REASON_WATCHDOG_CPU`, delay about 3 s, restart | direct binary path/strings |
| `kn_htchkd`: memory | enabled; available-memory floor 64 MiB and utilization thresholds 55/75/85 | drop caches first; persistent breach logs `REASON_WATCHDOG_RAM` and restarts | direct binary path/strings |
| `kn_htchkd`: Wi-Fi | starts after 60 s; 5 s checks, 180 s threshold, retry 1, reboot=1 | restart Wi-Fi; if unresolved restart with `REASON_WATCHDOG_WIFI` | direct binary path/strings |
| `kn_htchkd`: DNS/network | enabled; 30 s checks, retry 2, max 600 s, restart=1, recovery=0 | restart with `REASON_RST_N1_81`; optional CFUN 0/1 recovery exists but is disabled by default | direct binary path/strings |
| `kn_htchkd`: modem | starts after 120 s; 30 s checks; defect threshold 120; recovery=1 | Bank2 first performs WAN `ifup` recovery (60 s threshold), then restart with `REASON_WATCHDOG_MODEM` | direct binary path/strings |
| `kn_sysmond` | IPC/FIFO requests from UI, CLI, button, other daemons | normal restart reasons UI/CLI/button; long reset invokes defaults/factory-reset path | direct imports and strings |
| `kn_sysmond` random reboot | enabled, but start/end time empty by default | schedules randomized reboot only when a window is provisioned; `REASON_RST_RANDOM` | direct binary/config |
| `kn_fotad` | enabled; provisioning cycle 21,600 s; profile-driven | download, validate size/board/CRC via `upgrade`, flash inactive bank, save config, restart with `REASON_RST_UPGRADE` | direct binary/script chain |
| `kn_modeswitchd` | enabled; 5G/LTE/backup state machine | contains reboot decisions after attach/search/weak-signal flows; exact actuator is indirect, so it is not attributed as the 120 s loop | binary control strings; actuator unresolved |
| `scd` | enabled with `modem_reboot_ap=1` | accepts modem MIPC normal-reboot, poweroff, and preshutdown commands; runs `reboot`/`poweroff` | direct binary strings/imports |
| `mtk_monitor_hang` | enabled; `/usr/bin/mtk_monitor_hang 30000 120`; procd watchdog disabled while it owns hang monitoring | kicks kernel hang driver; on failure places driver in reboot state | direct init/binary path |
| procd hardware watchdog | normally available; explicitly stopped while `mtk_monitor_hang` runs | `/dev/watchdog` reset if userspace stops servicing it | direct procd/init path |
| thermal framework | enabled | CPU/device cooling; critical thermal shutdown; SoC hardware reboot configured at 117 C | direct config/binary path |
| Web UI normal restart | authenticated workflow writes `key_ctrl,ui,reset,500,short` to `kn_sysmond` | save then delayed restart | PHP controller source |
| Web UI factory reset | authenticated workflow writes `key_ctrl,ui,reset,5000,long` | defaults/factory restart path | PHP controller source |
| physical reset button | release under 1 s | immediate reboot | `/etc/rc.button/reset` |
| physical reset button | held at least 5 s with overlay mounted | `jffs2reset -y`, then reboot | `/etc/rc.button/reset` |
| reboot button | held at least 5 s | reboot | `/etc/rc.button/reboot` |
| power button event | release | `poweroff` | `/etc/rc.button/power` |
| sysupgrade stage 2 | after image write | unmount, `reboot -f`; after 5 s fallback writes `b` to sysrq-trigger | direct shell source |
| kernel/LK faults | panic, WDT, thermal, fastboot timeout, reset sources | LK classifies and reboots; AEE records exception class | LK/AEE strings and config |

`libknlibc.so` is the common restart actuator for KeyONet daemons.  It records a
reason in `/data/knos/knos-boot-reason`, stops logging, optionally runs
`firstboot`/`jffs2reset` for a factory reset, and executes immediate or delayed
`/sbin/reboot`.  Known reason names include UI, CLI, button, upgrade, save,
random, CPU, RAM, flash, LTE, Ethernet, USB, Wi-Fi, and modem watchdog causes.

## Vendor daemon inventory and role

The following is the exhaustive enabled vendor subset in current Bank2, grouped
by function.  Exact command, package, hash, start order, and stop order are in
`generated/services.tsv`.

| Group | Enabled services | Static role and power relevance |
|---|---|---|
| KeyONet monitor suite | `kn_dcsd`, `kn_fotad`, `kn_htchkd`, `kn_key_controller`, `kn_led_controller`, `kn_ledmond`, `kn_meshd`, `kn_modeswitchd`, `kn_procd`, `kn_syslogd`, `kn_sysmond`, `kn_telnetd` | FOTA, health, reset-button/UI IPC, mode switching, process/log/LED/mesh management. Power-changing members are detailed above. |
| KeyONet boot/config | `knos_config` | migrates UCI/NVRAM defaults, slot-dependent config, FOTA and monitor state at boot; clears stale reboot/FOTA runtime state. |
| MediaTek modem core | `1nvram_daemon`, `2ccci_fsd`, `3ccci_mdinit`, `3ccci_rpcd_com`, `atci_service`, `mdlogger`, `mipc_submonitor.init`, `mipc_wan.init` | modem NVRAM, firmware filesystem/RPC initialization, AT interface, modem logging, WAN and modem-IPC supervision. |
| MediaTek platform | `mtk_agpsd`, `mtk_bridge`, `mtk_hang_detect`, `mtk_mem`, `mtk_netagent`, `mtk_pre_wifi.init` | GNSS, bridge/data-call management, memory setup, Wi-Fi preload, and system-hang watchdog. |
| Quectel platform | `ql_ippt`, `ql_netd`, `ql_powerd.init`, `ql_ril_service`, `scd` | IP pass-through, network/RIL, sleep/USB/Wi-Fi power coordination, and modem-to-AP reboot/poweroff control. `ql_powerd` itself changes sleep/radio/USB state but has no direct reboot string. |
| ESP32 companion | `esp32_controllerd`, `esp32_uart` | companion-controller command and UART transport. No direct AP reboot path was found in these binaries. |

Vendor executables installed but not directly started by an init link are also
listed in `generated/executables.tsv`; this includes `upgrade`, `mipc_wan_cli`,
`nvramedit`, ESP32 flashing/control utilities, MTK data-call tools, and test
clients.  They are helpers reached by daemons or operator commands, not omitted
startup daemons.

## A/B version differences relevant to stability

- Bank1 is KnOS 1.01.44; Bank2 is 1.01.52.  The monitor-suite binaries differ.
- Bank2 adds modem WAN-IP recovery and `monitor.modem.threshold_recovery=60`.
- Bank2 changes `modeswitch.5g.ms_fdd_Band_Capability_Disable` from `1` to `0`
  and adds `weak_sinr_servingcell_check_flag=1`.
- Both banks retain the 120-second `procd` bring-up deadline and both contain the
  broken `ab_image_sync` calls and `current_slot` scope defect.
- The captured boot metadata explicitly marks Bank2 successful and Bank1 not
  successful.  That is stronger evidence than filesystem timestamps when
  determining the intended live bank.

## Operational implications

1. Keep Bank2 selected until Bank1 is repaired or deliberately replaced.  Bank1
   is not marked successful and was associated with the observed loop.
2. Do not rely on `zmtk_boot_done` to clone/repair a bank.  Back up first and use
   an independently verified writer if bank repair is required.
3. Treat `user_data` filesystem errors as potentially destructive: the vendor
   startup script reformats automatically.
4. Preserve `knos-boot-reason`, early `procd` output, kernel logs, and boot-control
   bytes across incidents.  They distinguish the 120-second bring-up reset from
   health, modem, thermal, watchdog, UI, and FOTA resets.
5. The static result establishes reachable code and defaults.  Paths dependent
   on modem firmware, kernel drivers, or provisioned server profiles require a
   controlled runtime fault-injection test to prove timing; no such fault was
   injected during this read-only audit.

## Reproduction

After extracting both SquashFS images, run:

```sh
python3 scripts/analyze-vendor-rootfs.py \
  --rootfs-a /path/to/rootfs-a \
  --rootfs-b /path/to/rootfs-b \
  --output-dir analysis-output
```

The analyzer hashes each init script and ELF, maps package ownership, records
enabled rc links, and extracts power-control evidence. The raw backup images
were never mounted read-write and the device was not modified during this
audit. Firmware images and persistent/runtime data are deliberately excluded
from this repository; only the report and derived inventories are published.
