# Changelog

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
