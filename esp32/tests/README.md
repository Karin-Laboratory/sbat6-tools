# ESP32 wrapper validation

The initial wrapper is intentionally too small rather than too clever.

Before adding a command to `esp32ctl`, all of the following should be true:

1. the command is observed on the target firmware or proven by a live query;
2. it is query-only and does not alter ESP32 or SBAT6 state;
3. its response is demonstrably short and bounded well below the vendor
   `esp32_uart` defect region;
4. concurrent access is serialized;
5. timeout behavior is known;
6. the command cannot expose stored credentials or unique identifiers by
   default;
7. failure does not require restarting `esp32_uart`.

Do not add Wi-Fi scan, BLE scan, GATT discovery, reset, flash, OTA, GPIO, or
configuration commands to the shell wrapper.

Those features belong behind the future single-owner broker after the response
handling defect has been addressed or safely contained.

## Static checks

A host-side test should at least verify that:

- `raw-short` rejects an unknown AT command;
- `ping` maps only to `AT`;
- `info` maps only to `AT+GMR`;
- the wrapper refuses to run without a timeout implementation;
- a second client cannot acquire the lock.
