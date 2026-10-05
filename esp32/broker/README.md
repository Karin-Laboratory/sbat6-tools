# esp32d broker seed

`esp32d` is the phase-1 single-owner broker for the SBAT6 internal ESP32-C3.

It does not replace `esp32_uart` and never opens `/dev/ttyS1`. The path is:

```text
client -> Unix socket -> esp32d -> esp32_controller -> esp32_uart -> ESP32-C3
```

## Current scope

Only two operations are accepted:

- `ping` -> `AT`
- `info` -> `AT+GMR`

The daemon handles clients serially and also acquires the legacy
`/tmp/esp32ctl.lock` around each vendor call, so the shell wrapper and broker
cannot invoke the vendor frontend simultaneously.

## Why commands remain tiny

The vendor `esp32_uart` path has a confirmed response-buffer over-read around
256-byte responses. A broker solves ownership, policy, timeout and concurrency.
It does not repair that vendor-side defect.

Wi-Fi scan, BLE scan, GATT discovery, long diagnostics and state-changing
commands therefore remain disabled.

## Protocol

Default socket: `/var/run/esp32d.sock`.

One newline-delimited JSON request is accepted per connection:

```json
{"operation":"info"}
```

Response:

```json
{"ok":true,"output":"..."}
```

Unknown operations are rejected before `esp32_controller` is invoked.

## Build

Go 1.22 or newer:

```sh
./build.sh
```

The helper defaults to Linux arm64 with CGO disabled and writes:

```text
dist/esp32d
dist/esp32ctl-broker
```

No binaries are committed.

## First live validation

Do not install `esp32d` as a boot service yet. On a recoverable test unit,
run it interactively and validate only:

```sh
esp32ctl-broker ping
esp32ctl-broker info
```

Confirm that `esp32_uart` remains running and that no restart, GPIO, UART or
persistent configuration changes occur.

The next development gate is the vendor long-response defect, not adding more
AT commands.
