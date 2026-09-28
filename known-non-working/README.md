# Known non-working configurations

These configurations caused reproducible failures on the laboratory SBAT6A.
They are retained as warnings and must not be deployed unchanged.

## `home24fix` synchronous Wi-Fi STA startup — DO NOT REPEAT SYNCHRONOUSLY

The original `home24fix` station conversion was successful when run manually,
but its boot-integrated form performed long waits synchronously. The device
entered an approximately 106–107 second reset loop before normal bring-up could
complete.

Do not place the original implementation, its 60-second wait, its subsequent
roughly 30-second wait, or equivalent association/DHCP retry loops directly in
an init `start()` path. Use the non-blocking worker design documented in
[`docs/WIFI_STA_BOOT_TIMEOUT_FIX.md`](../docs/WIFI_STA_BOOT_TIMEOUT_FIX.md).

The watchdog/vendor-supervisor explanation is the best fit for the measured
timing. The exact component that issued the final reset was not proven. Full
evidence and recovery details are in
[`docs/BOOT-LOOP-107S-FAILURE.md`](../docs/BOOT-LOOP-107S-FAILURE.md).
