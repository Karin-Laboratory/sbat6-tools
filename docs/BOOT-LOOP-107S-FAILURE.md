# 106–107 second boot loop caused by synchronous Wi-Fi STA startup

## Summary

Manual conversion of the 5 GHz radio to a Wi-Fi station succeeded on the
running system. The failure began only after the same procedure was installed
as the synchronous `home24fix` boot path. The unit then reset repeatedly after
approximately 106–107 seconds.

**A long-running synchronous Wi-Fi station-mode startup job most likely
blocked the normal boot sequence long enough for a watchdog/vendor supervisor
to reset the device. The final reset issuer remains unproven.**

This is a failure report, not a recipe. Do not repeat the synchronous boot
configuration.

## What worked

The underlying Wi-Fi operations were valid when performed manually after the
system had completed startup:

- creation and activation of the 5 GHz station interface;
- WPA association with the upstream access point;
- DHCP address acquisition; and
- SSH management through the station interface.

This distinction matters: station mode itself was not disproved. The unsafe
part was placing a long, blocking implementation directly in the boot-critical
path.

## Reproducible failure

The synchronous `home24fix` implementation included waits of roughly 60
seconds followed by another roughly 30-second phase while association and
network setup completed. Once installed in the boot path, the observed reset
period was approximately 106–107 seconds. The timing is consistent with the
boot sequence failing to reach a vendor-defined healthy/completed state before
a deadline.

Evidence supports the following sequence:

1. normal vendor services begin starting;
2. the synchronous Wi-Fi station job occupies the boot path;
3. the expected boot-completion/health transition is delayed;
4. a watchdog or vendor supervisor resets the unit; and
5. the same configuration is restored and the cycle repeats.

The timing correlation is strong, but the retained logs did not identify the
last process or kernel path that issued the reset. Do not claim a specific
watchdog or daemon as proven.

## Recovery

The decisive recovery was the device's A/B layout. The broken side could be
abandoned and the alternate known-good side selected, restoring a bootable
management path. That escape route was safer than continuing to alter the
failing slot under a short reset deadline.

After recovery, the STA startup was rewritten as a non-blocking init service.
Its `start()` method launches a guarded background worker and returns
immediately, allowing vendor bring-up to finish while association continues.
See [`WIFI_STA_BOOT_TIMEOUT_FIX.md`](WIFI_STA_BOOT_TIMEOUT_FIX.md).

## Lessons

- Prove an operation manually before integrating it into boot, but do not
  assume manual success makes a blocking init implementation safe.
- Never put long sleeps, association loops, DHCP waits, or network retries in a
  boot-critical synchronous path.
- Preserve an independent UART or wired recovery route before changing Wi-Fi.
- Keep a verified canonical restore archive; otherwise the failing script can
  be reintroduced on every boot.
- Treat A/B slot selection as a recovery boundary, not as a routine trial-and-
  error mechanism.
- Record reset timing and boot-completion evidence, and distinguish correlation
  from a proven reset issuer.
