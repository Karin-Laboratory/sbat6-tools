#!/usr/bin/env python3
"""Build a reproducible static inventory of SBAT6 startup and power paths."""

import argparse
import csv
import hashlib
import os
import re
import subprocess
from pathlib import Path


POWER_RE = re.compile(
    rb"(?i)(reboot|poweroff|shutdown|factory.?reset|jffs2reset|rollback|recovery|"
    rb"bootctrl|bootpart|current_slot|sysupgrade|upgrade|fota|watchdog|sysrq-trigger|"
    rb"force_ro|mmcblk0boot|boot.?reason|retry_[ab]|success_[ab])"
)
VENDOR_RE = re.compile(
    r"(?i)(^|[-_/])(kn|knos|keyonet|ql|quectel|mtk|mediatek|mipc|esp32|sbk|"
    r"rudolf|ccci|connsys|mdlogger|atci|scd)([-_/]|$)"
)


def parse_args():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--rootfs-a", type=Path, required=True,
                        help="extracted Bank1/A root filesystem")
    parser.add_argument("--rootfs-b", type=Path, required=True,
                        help="extracted Bank2/B root filesystem")
    parser.add_argument("--output-dir", type=Path, required=True,
                        help="directory in which TSV reports are written")
    return parser.parse_args()


def sha256(path):
    digest = hashlib.sha256()
    with path.open("rb") as stream:
        for block in iter(lambda: stream.read(1024 * 1024), b""):
            digest.update(block)
    return digest.hexdigest()


def read(path):
    try:
        return path.read_bytes()
    except (OSError, PermissionError):
        return b""


def package_map(tree):
    result = {}
    info = tree / "usr/lib/opkg/info"
    if not info.is_dir():
        return result
    for listing in info.glob("*.list"):
        package = listing.name[:-5]
        for line in read(listing).decode("utf-8", "replace").splitlines():
            result[line.rstrip("/")] = package
    return result


def commands(text):
    found = []
    patterns = [
        r"(?:^|\n)\s*PROG\s*=\s*['\"]?([^\s'\"]+)",
        r"procd_set_param\s+command\s+([^\n;]+)",
        r"procd_append_param\s+command\s+([^\n;]+)",
        r"service_start\s+([^\n;]+)",
        r"start-stop-daemon[^\n]*?--exec\s+([^\s;]+)",
    ]
    for pattern in patterns:
        found.extend(match.strip() for match in re.findall(pattern, text, re.M))
    return sorted(set(found))


def write_dict_tsv(path, rows, fields):
    with path.open("w", newline="", encoding="utf-8") as stream:
        writer = csv.DictWriter(
            stream, fieldnames=fields, delimiter="\t", lineterminator="\n"
        )
        writer.writeheader()
        writer.writerows(rows)


def analyze(trees, output_dir):
    services = []
    binaries = []
    evidence = []

    for slot, tree in trees.items():
        if not tree.is_dir():
            raise SystemExit(f"rootfs {slot.upper()} is not a directory: {tree}")

        packages = package_map(tree)
        enabled = {}
        rc_dir = tree / "etc/rc.d"
        if rc_dir.is_dir():
            for link in rc_dir.iterdir():
                if link.is_symlink() and link.name[:1] in ("S", "K"):
                    target = os.path.basename(os.readlink(link))
                    enabled.setdefault(target, []).append(link.name)

        init_dir = tree / "etc/init.d"
        for path in sorted(init_dir.glob("*")):
            if not path.is_file():
                continue
            relative = "/" + str(path.relative_to(tree))
            data = read(path)
            text = data.decode("utf-8", "replace")
            command_list = commands(text)
            package = packages.get(relative, "")
            is_vendor = bool(
                VENDOR_RE.search(path.name)
                or VENDOR_RE.search(package)
                or any(VENDOR_RE.search(item) for item in command_list)
            )
            services.append({
                "slot": slot,
                "service": path.name,
                "enabled_links": " ".join(sorted(enabled.get(path.name, []))),
                "start": next(iter(re.findall(r"(?m)^START=(\S+)", text)), ""),
                "stop": next(iter(re.findall(r"(?m)^STOP=(\S+)", text)), ""),
                "commands": " | ".join(command_list),
                "package": package,
                "vendor_related": int(is_vendor),
                "sha256": sha256(path),
                "path": relative,
            })
            for line_number, line in enumerate(text.splitlines(), 1):
                if POWER_RE.search(line.encode("utf-8", "replace")):
                    evidence.append((slot, relative, line_number, "script", line.strip()))

        for base in ("bin", "sbin", "usr/bin", "usr/sbin"):
            folder = tree / base
            if not folder.is_dir():
                continue
            for path in sorted(folder.iterdir()):
                if not path.is_file() or path.is_symlink():
                    continue
                relative = "/" + str(path.relative_to(tree))
                if not read(path).startswith(b"\x7fELF"):
                    continue
                package = packages.get(relative, "")
                referenced = [
                    service["service"] for service in services
                    if service["slot"] == slot and relative in service["commands"]
                ]
                is_vendor = bool(VENDOR_RE.search(relative) or VENDOR_RE.search(package) or referenced)
                try:
                    description = subprocess.run(
                        ["file", "-b", str(path)], capture_output=True, text=True, check=False
                    ).stdout.strip()
                except OSError:
                    description = "ELF"
                try:
                    raw_strings = subprocess.run(
                        ["strings", "-a", "-n", "5", str(path)], capture_output=True, check=False
                    ).stdout
                    hits = sorted({
                        item.decode("utf-8", "replace")[:500].rstrip()
                        for item in raw_strings.splitlines() if POWER_RE.search(item)
                    })
                except OSError:
                    hits = []
                binaries.append({
                    "slot": slot,
                    "path": relative,
                    "package": package,
                    "referenced_by": " ".join(referenced),
                    "vendor_or_daemon": int(is_vendor),
                    "size": path.stat().st_size,
                    "sha256": sha256(path),
                    "power_string_count": len(hits),
                    "file": description,
                })
                evidence.extend((slot, relative, "", "ELF-string", hit) for hit in hits)

    output_dir.mkdir(parents=True, exist_ok=True)
    write_dict_tsv(output_dir / "services.tsv", services, (
        "slot", "service", "enabled_links", "start", "stop", "commands", "package",
        "vendor_related", "sha256", "path",
    ))
    write_dict_tsv(output_dir / "executables.tsv", binaries, (
        "slot", "path", "package", "referenced_by", "vendor_or_daemon", "size",
        "sha256", "power_string_count", "file",
    ))
    with (output_dir / "power_evidence.tsv").open("w", newline="", encoding="utf-8") as stream:
        writer = csv.writer(stream, delimiter="\t", lineterminator="\n")
        writer.writerow(("slot", "path", "line", "kind", "evidence"))
        writer.writerows(evidence)
    print(f"services={len(services)} executables={len(binaries)} evidence={len(evidence)}")


def main():
    args = parse_args()
    analyze({"a": args.rootfs_a.resolve(), "b": args.rootfs_b.resolve()}, args.output_dir.resolve())


if __name__ == "__main__":
    main()
