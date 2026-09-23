#!/bin/sh
# SBAT6: rebuild only the inactive B partition family from live slot A.
set -eu
pairs='7:9 8:10 14:27 15:28 16:29 17:30 18:31 19:32 20:33 21:34 22:35 23:36 24:37 25:38 26:39'
case "$(cat /proc/cmdline)" in *bootslot=a*) ;; *) echo 'REFUSE: slot A is not active' >&2; exit 1;; esac
for pair in $pairs; do
  a=${pair%%:*}; b=${pair##*:}; src=/dev/mmcblk0p$a; dst=/dev/mmcblk0p$b
  [ "$(wc -c < "$src")" = "$(wc -c < "$dst")" ] || { echo "REFUSE: size mismatch p$a/p$b" >&2; exit 1; }
  printf 'COPY p%s -> p%s\n' "$a" "$b"
  dd if="$src" of="$dst" bs=4M conv=fsync
done
sync
for pair in $pairs; do
  a=${pair%%:*}; b=${pair##*:}; src=$(sha256sum /dev/mmcblk0p$a | awk '{print $1}'); dst=$(sha256sum /dev/mmcblk0p$b | awk '{print $1}')
  [ "$src" = "$dst" ] || { echo "VERIFY_FAIL p$a->p$b" >&2; exit 1; }
  echo "VERIFY_OK p$a->p$b sha256=$src"
done
echo FINAL_STATUS=PASS
