# INT-S4 physical touch device readiness

- Date: 2026-09-02
- Host: macOS 26.6.2 / arm64
- Scope: `INT-408` physical-touch preflight only
- Result: **not runnable; device unavailable**

## Observed state

Read-only device discovery used `xcrun devicectl list devices`,
`xcrun devicectl device info details`, `xcrun xctrace list devices`, and the
USB device inventory. A previously paired physical iPhone 17 Pro running iOS
26.5 was present in CoreDevice metadata, but it was reported as unavailable:

```text
state: unavailable
pairingState: paired
tunnelState: unavailable
ddiServicesAvailable: false
xctrace: Devices Offline
USB inventory: no connected iPhone or iPad
```

The committed record deliberately omits the device UDID, ECID and hostnames.
No application was installed, no device setting was changed, and no personal
media or browser data was read.

## Gate consequence

The paired record proves that a physical target exists, but it is not execution
evidence. FolioPath was not opened on the device and no touch interaction was
performed. Simulator, Playwright touch emulation and responsive viewport checks
remain ineligible substitutes.

To resume this check, the paired device must be powered on, unlocked, trusted,
reachable by USB or the configured wireless pairing path, and expose an
available developer tunnel/DDI service. `INT-408` remains incomplete until a
physical-device run is captured, unless the product owner explicitly changes
that acceptance requirement in a separate change record.
