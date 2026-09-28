# SBAT6A cellular results: povo success and Japan Communications failure

This document records the cellular tests performed on the laboratory SBAT6A
on 2026-09-26 and 2026-09-27. It separates observations from hypotheses and
omits complete ICCID, IMSI, IMEI, telephone number, credentials, and exact
installation address.

## Executive summary

| Item | povo / KDDI | Japan Communications / docomo MVNO |
|---|---|---|
| SIM recognized | Yes | Yes |
| PLMN | `44054` | Home PLMN `44010` |
| LTE cell detected | Yes | Yes |
| LTE/EPS registration | Successful | Rejected (`CEREG stat=3`) |
| PDP/data session | Successful | Never reached |
| DNS, ping, HTTPS | Successful | Not possible on SBAT6A |
| Same SIM in Pixel 10a | Not used as a failure control | Data communication successful |
| APN | `povo.jp` | `dm.jplat.net` |
| SIM personalization locks | All seven categories unlocked | Same modem state; all unlocked |
| Main conclusion | KDDI LTE is usable; NR was intermittent and location-dependent | Failure occurs before APN/PDP setup, during EPS registration |

The comparison disproves a general radio failure and a general foreign-SIM
lock: the modified unit registers and passes data on KDDI. The docomo failure
is specific to the combination of the Japan Communications SIM, docomo EPS
registration, and the SoftBank-customized RG620T-SBK modem firmware/profile.

## Common test platform

- Device: SoftBank Air Terminal 6 (laboratory unit)
- Modem: Quectel `RG620T-SBK`
- Modem firmware reported during the test:
  `RG620TSBK00AAR03A01G4G_OCPU`
- MediaTek modem build:
  `MOLY.NR16.R2.MD800.MP1P5MR1.MP.V26.4.P54`
- Host firmware: SBAT6 firmware `1.00.22`
- SIM mapping: physical tray
- SIM state in both tests: `READY`
- Carrier selection: automatic unless a manual registration attempt is
  explicitly described below
- The previously configured fixed KDDI PLMN was removed before the docomo
  test. The persistent restore archives were also updated, so it was not
  silently restored after reboot.

## povo / KDDI: successful baseline

### Registration and data path

The povo SIM registered on KDDI PLMN `44054` and established an IPv4 PDP
session. Device-local tests confirmed:

- LTE home registration;
- WAN interface up;
- IPv4 address assigned;
- DNS resolution successful;
- ICMP successful; and
- HTTPS transfer successful.

The network-reported session AMBR was 4 Mbit/s downlink and 4 Mbit/s uplink.
This was a control-plane value and must not be interpreted as a measured
throughput limit. A later unlimited-data topping was activated for load tests.

### LTE serving cell observed at the main installation area

| Field | Value |
|---|---|
| RAT | LTE / 4G |
| LTE band | Band 1 |
| EARFCN | `100` |
| PCI | `260` |
| TAC | `0x9B4F` / 39759 |
| ECI | `0x0A1E8C17` / 169774103 |
| Derived eNodeB ID | 663180 |
| Derived sector/cell number | 23 |

Position changes did not move the modem to another LTE ECI. Representative
RSRP readings were:

- first position: approximately -93 dBm;
- weaker intermediate position: approximately -106 dBm; and
- later position: approximately -94 dBm.

CSQ improved from 8 at the weaker position to 11 after repositioning. The
-93/-94 dBm positions therefore provided a usable LTE signal even though NR
availability remained inconsistent.

### NR/5G observations

Most idle checks reported LTE registration with no 5G SA registration
(`C5GREG: 0,0`). Moving the unit through several indoor positions did not
produce a stable NR registration. During a later detailed check with traffic
load, NR activity was observed transiently; this was the reason a throughput
test was started. It did not remain reproducible across the tested locations.

The safe conclusion is therefore:

- KDDI LTE registration and data are repeatably successful;
- NR can be visible or activated transiently under some radio/load
  conditions; but
- continuous 5G service was not demonstrated at the installation area.

Do not treat an idle LTE anchor alone as proof that NSA is unavailable. Also
do not treat the historical transient NR observation as proof of persistent
5G coverage.

### Band state used for the later povo tests

Initial SoftBank-oriented state:

- LTE supported: B1, B8, B41, B42;
- LTE enabled: B1, B41, B42;
- NR supported: n3, n28, n77, n79;
- NR enabled initially: n3, n77.

The supported n28 band was then added while retaining n3 and n77. The change
was made persistent and was rechecked after reboot. n78 cannot be enabled
because this RG620T-SBK variant does not report n78 support. n79 was not part
of the confirmed povo configuration.

## Japan Communications / docomo: failed registration

### SIM and APN

The physical Japan Communications SIM was detected correctly:

- SIM state: `READY`;
- IMSI prefix: `44010` (docomo);
- ICCID prefix: `898110`;
- APN: `dm.jplat.net`;
- authentication: PAP/CHAP;
- PDP type: IPv4; and
- carrier selection: automatic.

The configured username and password are intentionally omitted. The same SIM
was moved to a Pixel 10a and completed real data communication, then returned
to the SBAT6A. This establishes that the SIM and subscription were active.

### Radio visibility

The SBAT6A detected an available docomo LTE cell:

| Field | Value |
|---|---|
| PLMN | `44010` |
| LTE band | Band 1 |
| EARFCN | `324` |
| PCI | `398` |
| TAC | `0x124A` |
| ECI | `0x02ED0202` |
| RSRP | approximately -98 to -101 dBm |
| RSRQ | approximately -11.5 to -12.5 dB |

An operator scan listed `44010` as available. Therefore the failure was not
equivalent to no docomo RF coverage, and Band 1 support was sufficient to
exchange registration signalling with the cell.

### Reproducible failure

Automatic registration and an explicit detach/reattach produced:

```text
+CEREG: 3,3,"124A","02ED0202",7,0,0
+CEER: 0,NONE
```

Interpretation:

- `CEREG stat=3`: registration denied;
- access technology `7`: E-UTRAN/LTE;
- the modem reached the docomo cell identified above;
- the vendor AT layer returned cause type 0 and reject cause 0; and
- `CEER` did not expose a more specific failure reason.

Manual selection with `AT+COPS=1,2,"44010",7` also failed (generic/CME 100).
The following did not change the result:

- returning to automatic operator selection;
- modem `CFUN` reset;
- full SBAT6A reboot;
- reinserting the SIM after a successful Pixel 10a session; and
- testing the generic RAT setting in place of the vendor setting, then
  restoring the original setting.

WAN remained pending because EPS registration never completed. APN
authentication and PDP establishment occur after this stage, so an APN error
does not explain the observed `CEREG stat=3` failure.

## Conditions ruled out

### SIM lock and personalization

All MediaTek SIM-lock categories reported unlocked, with zero rule entries:

1. network;
2. network subset;
3. service provider;
4. corporate;
5. SIM/USIM;
6. MediaTek network-subset/operator extension; and
7. MediaTek SIM/corporate extension.

Standard checks for `PN`, `PU`, `PP`, and `PC` also returned unlocked. The
SIM was `READY`. The vendor lock implementation found in the firmware covers
SoftBank-oriented network/service-provider rules, and those rules had been
removed. There is no remaining docomo-MVNO personalization lock indicated by
the modem.

### Stale povo configuration

The old fixed PLMN `44054` was deleted from the live configuration and from
both persistent restore archives. The canonical archive contained the Japan
Communications APN and no fixed PLMN. Registration tests were repeated after
a complete reboot.

### Forbidden PLMN list

Read-only SIM-file inspection returned these entries from `EF_FPLMN`:

- `44020`;
- `44000`;
- `44050`; and
- `44051`.

docomo `44010` was not present, including after another reproduced rejection.
The failure is therefore not explained by the SIM treating `44010` as a
forbidden PLMN. The read-only operation did not alter the SIM.

### SIM location state

`EF_LOCI` and `EF_PSLOCI` contained `44010` but no usable current temporary
identity, consistent with a previous docomo session followed by an incomplete
registration on this modem. `EF_EPSLOCI` was not exposed by this SIM/application
through either CRSM or a logical USIM channel. No SIM file was modified.

## Remaining modem-specific findings

The modem is a SoftBank-customized `RG620T-SBK`, not a generic Japanese
multi-carrier module. Read-only MediaTek configuration checks showed:

```text
AT+EDSBP?  -> +EDSBP: 2
SBP_DISABLE_LTE_BAND_BY_PLMN -> 1
```

Firmware diagnostics also contain the SoftBank dynamic carrier profile
identity `SBP_ID 50`, MCC/MNC `441-00`. SBP/DSBP is independent of SIM
personalization locking and can affect band policy, PLMN selection, IMS, and
NAS behavior. Band 1 was nevertheless active and docomo signalling was
exchanged, so the active SBP flag is evidence of a carrier-specific modem
configuration, not proof of the exact reject cause.

The module's published LTE support is B1/B8/B41/B42. Lack of docomo B19
reduces coverage, but it cannot by itself explain this test because a usable
B1 cell was detected and the network returned an explicit registration
denial.

## Reject-cause and device-identity limits

MediaTek MDLogger captured the reproduction in a proprietary `.muxz` log.
Decoding NAS messages requires a compatible MediaTek ELT build and the exact
modem database. No usable open-source decoder for this generation was found.
The exposed AT interfaces suppress the actual EPS Attach Reject cause.

The IMEI TAC was checked without publishing the full IMEI. Public results did
not identify the unit's TAC, while one public RG620T-SBK module record used a
different TAC. This does not establish an IMEI block: a completed SoftBank Air
product may legitimately use a TAC different from the bare module. No explicit
IMEI/ICCID whitelist was found in the static firmware audit, and there is no
evidence proving that docomo rejected this particular IMEI.

## Final assessment

### Proven

- The modem and modified router work on povo/KDDI LTE and pass user data.
- The Japan Communications SIM is active and passes data in a Pixel 10a.
- The SBAT6A sees and exchanges LTE signalling with docomo Band 1.
- docomo EPS registration is explicitly denied before APN/PDP setup.
- SIM personalization, fixed-KDDI PLMN restoration, and a forbidden `44010`
  entry are not the cause.
- SoftBank-specific MediaTek SBP/DSBP configuration remains active separately
  from SIM unlocking.

### Not proven

- The exact EPS Attach Reject cause value.
- An IMEI/TAC blacklist or whitelist decision by docomo.
- That the active SoftBank SBP setting directly causes the rejection.
- General incompatibility of every RG620T-SBK with every docomo SIM.

### Best remaining hypothesis

The strongest remaining explanation is a docomo interoperability issue caused
by the SoftBank-specific modem firmware/carrier profile, potentially involving
the NAS capability/profile presented during registration. A network-side
device-identity policy remains possible but unsupported by the available
evidence. Further work would require a compatible proprietary MediaTek log
decoder, Quectel/SoftBank modem documentation, or a controlled comparison with
another docomo SIM and/or a generic Japanese RG620T firmware. No further modem
configuration changes were made during the closing investigation.
