# SBAT6 internal ESP32-C3

This directory documents and provides a deliberately small access layer for the
ESP32-C3 module fitted inside the SoftBank Air Terminal 6.

## Confirmed platform

The 2026-10-05 audit identified the module as an **ESP32-C3 MINI-1** running:

- ESP-AT: `3.1.0.0-dev`
- ESP-IDF: `v5.0-541-g885e501d99`
- binary version: `2.5.0`
- host link: UART1, `/dev/ttyS1`, 115200 baud, 8N1
- host UART driver: `mt6577-uart`
- DT node: `/serial@11003000`
- MMIO: `0x11003000`

The normal owner of the UART is the vendor process
`/usr/sbin/esp32_uart 6`. Do **not** open `/dev/ttyS1` directly while that
process is active.

The host-side control path is:

```text
client
  -> /usr/sbin/esp32_controller --atcmd ...
  -> vendor FIFO/IPC path
  -> /usr/sbin/esp32_uart
  -> /dev/ttyS1
  -> ESP32-C3
```

## Current capability assessment

| Capability | Status | Notes |
|---|---|---|
| SoftAP | feasible | Vendor startup logic already uses ESP-AT SoftAP commands. |
| STA | partial | Command family is present; not promoted to a production API yet. |
| Wi-Fi scan | partial | Long responses are unsafe through the current vendor FIFO path. |
| BLE gateway | partial | BLE AT support is present, but scan/GATT need a safer broker first. |
| MQTT | present | AT command family was registered on the tested firmware. |
| HTTP | unknown | Expected for the matching upstream build, but not fully recovered from the live command registry. |

## Critical safety finding

The vendor `esp32_uart` process has a **256-byte response buffer over-read**
condition. Long responses, especially scans and other list-style commands,
must not be exposed through the current FIFO path as a general-purpose API.

For that reason the first public tool, `esp32ctl`, is intentionally tiny:

- exclusive single-client lock;
- short-query allowlist only;
- timeout;
- no reset, flash, OTA, GPIO, Wi-Fi reconfiguration, BLE mutation, or direct
  UART access;
- no scan support.

This is a safety boundary, not a feature limitation to work around casually.

## Files

- `device.yaml`: machine-readable platform facts for AI-assisted tooling.
- `protocol.md`: current host-side ownership and IPC design notes.
- `commands.yaml`: conservative command classification.
- `esp32ctl`: minimal query-only wrapper.
- `tests/README.md`: validation requirements before adding commands.

## Roadmap

The next architectural step is a **single-owner broker** between shell/LuCI/MQTT
clients and the vendor IPC path. The broker should serialize requests, enforce
an allowlist, bound responses, maintain transaction state, and reject unsafe
commands. Only after that layer exists should scan, BLE discovery/GATT, or LuCI
integration be enabled.

This directory intentionally does not contain ESP32 firmware images, device
identifiers, wireless credentials, or raw UART captures.
