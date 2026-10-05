# Changelog

## 2026-10-05

- Documented the internal ESP32-C3 MINI-1, its UART1/115200/8N1 host link and
  ESP-AT 3.1.0.0-dev firmware identification.
- Added machine-readable ESP32 device and command metadata plus IPC design
  notes for AI-assisted driver/SDK work.
- Added a deliberately minimal `esp32ctl` query-only wrapper with an exclusive
  lock, timeout and short-command allowlist.
- Recorded the vendor `esp32_uart` response-buffer over-read near 256 bytes and
  kept Wi-Fi/BLE scans and other long responses out of the initial API.
- Added the phase-1 `esp32d` single-owner broker seed and Go companion client.
  The broker serializes clients over a Unix socket, reuses the same inter-process
  lock as the shell wrapper, and still permits only `AT` and `AT+GMR`.
- Added policy tests and an arm64 static build helper for the broker.
- Kept scan/GATT/LuCI expansion blocked until the vendor long-response defect is
  fixed or safely bypassed.

## 2026-09-29

- Documented the non-working synchronous `home24fix` boot integration and its
  approximately 106–107 second reset loop.
- Added an explicit known-non-working warning against synchronous Wi-Fi STA
  association and DHCP waits in the boot-critical path.
- Added the verified non-blocking management-STA helper and its management-only
  DHCP hook, preserving the cellular default route and DNS.
- Added persistent 24-hour network diagnostics for link, bridge, routing,
  firewall, Wi-Fi, and modem state.
- Documented and mitigated management loss caused by vendor 5 GHz dynamic
  channel selection and DFS reconfiguration.
- Added bridge-only wired management addressing and source-specific management
  routes so wired replies do not depend on the Wi-Fi STA.
- Recorded the povo/KDDI success baseline and Japan Communications/docomo EPS
  registration failure without subscriber credentials or complete device
  identifiers.
