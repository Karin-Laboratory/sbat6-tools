# SBAT6 ESP32 host-control protocol notes

## Ownership

The ESP32 UART is not a free serial port. The vendor process
`/usr/sbin/esp32_uart 6` owns `/dev/ttyS1` during normal operation.

Applications should therefore use the vendor frontend
`/usr/sbin/esp32_controller --atcmd ...` rather than opening the UART.

## Transport facts

- UART1
- `/dev/ttyS1`
- 115200 baud
- 8 data bits
- no parity
- 1 stop bit
- DT node `/serial@11003000`
- MediaTek `mt6577-uart`
- MMIO `0x11003000`

## Existing IPC path

Static and live inspection confirmed a vendor FIFO/IPC path involving
`/tmp/esp32at_fifo` between the controller frontend and UART owner.

The existing IPC path is suitable for **short query responses only** at this
stage. It must not be treated as a generic unbounded AT transport.

## Known defect

The vendor `esp32_uart` implementation contains a response-buffer over-read
condition at approximately 256 bytes. This makes long responses unsafe to
promote through a generic wrapper.

Affected/high-risk classes include, at minimum:

- Wi-Fi scan/list responses;
- BLE scan/list responses;
- long GATT/service discovery results;
- any future command whose response length has not been bounded experimentally.

## Initial API contract

`esp32ctl` is intentionally narrow.

It must:

1. acquire an exclusive lock;
2. accept only explicitly allowlisted short queries;
3. invoke the vendor frontend;
4. use a timeout;
5. reject direct UART access and state-changing AT commands;
6. return the vendor output without attempting to reinterpret unknown fields.

## Future broker

Longer operations need a single-owner broker rather than more shell wrappers.

Proposed shape:

```text
shell / LuCI / MQTT / n8n
           |
           v
         esp32d
           |
   bounded request queue
   transaction state
   allowlist and policy
   response-length guards
           |
           v
 vendor ESP32 IPC path
           |
           v
      esp32_uart
           |
           v
       ESP32-C3
```

Do not replace `esp32_uart` until the vendor IPC behavior and boot-time
dependencies are fully understood.
