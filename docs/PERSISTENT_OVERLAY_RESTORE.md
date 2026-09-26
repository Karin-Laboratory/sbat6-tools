# SBAT6 persistent overlay restore path

This note describes the verified persistence mechanism on the laboratory
SBAT6A. It is not a vendor process that periodically resets live configuration.
There are two separate stages: a vendor one-shot restore at boot and a locally
installed cron job that prepares the archive for the next boot.

## Data flow

```text
/data/sbat6-ssh-overlay.tgz       canonical local seed
              |
              | root cron, once per minute
              v
/data/sysupgrade.tgz             next-boot working copy
              |
              | vendor S001restore, once during boot
              v
/                               extracted into the live root overlay
              |
              | vendor boot-completion cleanup
              v
/data/sysupgrade.tgz removed; cron recreates it for the next boot
```

The local root crontab entry is:

```cron
* * * * * cp /data/sbat6-ssh-overlay.tgz /data/sysupgrade.tgz
```

The vendor side is implemented by `/etc/init.d/1restore`, linked as
`/etc/rc.d/S001restore`. It treats `/data/sysupgrade.tgz` as the configuration
archive and extracts it during early userspace startup. Later vendor
boot-completion code removes the consumed working copy. The cron entry then
recreates it from the canonical seed.

## Operational consequence

Editing `/etc/config/*`, an init script, or another file in only the live root
overlay is insufficient. The edit can work until reboot and then appear to
"roll back" because the older canonical archive is restored. This is not a
periodic vendor reset: it happens at boot because the locally maintained seed
still contains the previous version.

Any intended persistent change must therefore be applied to both:

1. the live overlay, for the running system; and
2. `/data/sbat6-ssh-overlay.tgz`, followed by an identical
   `/data/sysupgrade.tgz`, for the next boot.

This includes network and wireless configuration, DHCP-disable settings,
late-start services, and sysctl policy. For example, a 6 GHz AP bridged to a
raw Ethernet LAN needs its `wireless`, `network`, and DHCP configuration in the
seed. If vendor startup later overrides a bridge sysctl, the late-start helper
that restores the desired value must also be present in the seed archive.

## Safe update procedure

Never modify the canonical archive in place. Work in a dedicated directory on
`/data`, retain the previous archive, and publish the replacement only after it
passes validation.

```sh
work=/data/overlay-update-YYYYMMDD-HHMMSS
mkdir -p "$work/root"

cp -p /data/sbat6-ssh-overlay.tgz "$work/seed.before.tgz"
tar -xzf /data/sbat6-ssh-overlay.tgz -C "$work/root"

# Copy only reviewed live files into "$work/root" here.

tar -czf "$work/seed.new.tgz" -C "$work/root" .
gzip -t "$work/seed.new.tgz"
tar -tzf "$work/seed.new.tgz"

mv "$work/seed.new.tgz" /data/sbat6-ssh-overlay.tgz
cp -p /data/sbat6-ssh-overlay.tgz /data/sysupgrade.tgz
sha256sum /data/sbat6-ssh-overlay.tgz /data/sysupgrade.tgz
```

The two SHA-256 values must match. Inspect the member list and, where useful,
extract individual files and compare them with the reviewed live versions.
Do not place passwords, private keys, device backups, or other secrets in this
public repository; the archive itself remains device-local.

## Reboot verification

After a controlled reboot, verify all of the following rather than treating a
successful ping as sufficient:

- vendor bring-up reaches `System bring-up completed.`;
- management paths return without a reboot loop;
- the expected UCI values survived;
- required interfaces and bridges exist and are forwarding;
- authentication daemons own the expected wireless interfaces;
- the device-local DHCP server remains disabled when an upstream server is
  intended; and
- any late-start sysctl helper ran after vendor initialization.

The working copy may be absent immediately after boot because it was consumed.
That is expected; the cron job should recreate it within one minute with the
same digest as the canonical seed.
