# Baton — Milestones & TODO

Granular task breakdown derived from `design.md`. Check off items as they are completed. Each milestone is ordered; later milestones depend on earlier ones unless noted.

---

## M0 — Project Bootstrap & Tooling

### M0.1 Repository & workspace
- [x] Create repository `baton/` with top-level folders: `protocol/`, `android/`, `desktop/`, `docs/`
- [x] Add root `README.md` with project summary, platforms, and quick start
- [x] Add `.gitignore` for Dart/Flutter, Android, Windows, Linux build artifacts
- [x] Add `LICENSE` (choose MIT or Apache-2.0) and record choice in docs
- [x] Add `CONTRIBUTING.md` with branch/PR conventions
- [x] Add `docs/` with `design.md` (existing) and `protocol.md` placeholder

### M0.2 Flutter / Dart toolchain
- [x] Pin Flutter SDK version in `fvm`/`.tool-versions` or `pubspec` environment
- [x] Verify `flutter doctor` clean on Windows dev machine (minor SDK warnings)
- [x] Verify `flutter doctor` clean on Linux dev machine (or VM) (CI ubuntu)
- [x] Verify Android SDK + emulator available (SDK present)
- [x] Add `analysis_options.yaml` with strict lint rules at repo root and per-package

### M0.3 Continuous integration
- [x] Create GitHub Actions workflow: `flutter analyze` + `flutter test` for `protocol/`
- [x] Create workflow for desktop build (Windows + Linux) on push to main
- [x] Create workflow for Android APK build
- [x] Add artifact upload for builds
- [x] Add status badge to README

### M0.4 Shared conventions
- [x] Define package naming (`baton_protocol`, `baton_core`, etc.) and document
- [x] Define logging convention (levels, tags, redaction of addresses/secrets)
- [x] Define error code enum shared across platforms
- [x] Define versioning scheme (semver) for protocol messages

---

## M1 — Protocol Package (`protocol/`)

### M1.1 Package scaffold
- [x] Create Dart package `protocol/` with `pubspec.yaml` (name: `baton_protocol`)
- [ ] Set Dart SDK constraint and dependencies (none external initially)
- [ ] Add `lib/src/` structure: `messages/`, `state/`, `pairing/`, `transport/`
- [ ] Add `lib/baton_protocol.dart` barrel export
- [ ] Add example/ smoke test

### M1.2 Message envelope
- [x] Define `Message` base class with `type`, `id` (UUID), `ts` (epoch ms)
- [x] Implement JSON serialization/deserialization with strict field validation
- [x] Implement length-prefixed framing helpers (encode/decode)
- [x] Add unit tests for envelope round-trip
- [ ] Add unit tests for malformed/unknown message handling

### M1.3 Message types (v1)
- [x] Implement `Hello` (version, capabilities)
- [x] Implement `HandoffRequest` (bt address, device name, direction)
- [x] Implement `HandoffReady`
- [x] Implement `Released`
- [x] Implement `HandoffDone`
- [x] Implement `HandoffFailed` (reason code)
- [x] Implement `Busy`
- [x] Implement `DeviceList` (list of headphones + owner state)
- [x] Add `type` string constants and a registry/factory for decoding
- [x] Add round-trip tests for each message type

### M1.4 Version & capability negotiation
- [ ] Define protocol version constant (e.g. `1.0`)
- [ ] Implement `Hello` exchange on connect
- [ ] Reject peers with incompatible major version
- [ ] Expose capability flags (multipoint, audio crossover, etc.)
- [ ] Tests for version mismatch handling

### M1.5 Error codes
- [x] Define `HandoffFailureReason` enum: `DEVICE_UNAVAILABLE`, `BUSY`, `TIMEOUT`, `DISCONNECT_FAILED`, `CONNECT_FAILED`, `OUT_OF_RANGE`, `REJECTED`, `UNKNOWN`
- [ ] Map reason codes to user-facing strings (i18n-ready)
- [ ] Tests for enum serialization

---

## M2 — Pairing & Security

### M2.1 Certificate generation
- [ ] Generate per-device self-signed cert (RSA/EC) on first run
- [ ] Store private key in OS secure storage (Android Keystore / Windows DPAPI / Linux libsecret)
- [ ] Expose cert fingerprint (SHA-256)
- [ ] Unit tests for cert generation determinism/format

### M2.2 Pairing data model
- [x] Define `Peer` model: id, name, address, cert fingerprint, pairedAt
- [x] Define `PairingStore` interface (load/save/list/remove)
- [x] Implement file-backed store for desktop
- [ ] Implement shared-prefs/secure-storage-backed store for Android
- [ ] Tests for store CRUD

### M2.3 QR code pairing flow
- [x] Generate QR payload: address, port, cert fingerprint, one-time secret
- [ ] Desktop: render QR code in pairing window (use `qr_flutter` or native)
- [ ] Phone: scan QR via camera plugin
- [x] Validate QR payload schema and expiry (2 min)
- [x] Tests for payload encode/decode + expiry

### M2.4 Pairing handshake
- [ ] Implement TCP connect + TLS with cert pinning
- [ ] Exchange certificates; verify against pinned fingerprint
- [ ] Exchange one-time secret; reject on mismatch
- [ ] Show short numeric confirmation code on both screens
- [ ] Require user confirmation on both sides before finalizing
- [ ] Persist peer on success
- [ ] Timeout pairing after 2 minutes
- [ ] Tests with loopback TLS

### M2.5 Revocation
- [ ] `revokePeer(id)` on either side removes pairing
- [ ] Drop active connection on revoke
- [ ] Reject future connections from revoked cert
- [ ] UI entry points (M9) wired to this
- [ ] Tests for revoke + reconnect rejection

### M2.6 TLS transport security
- [ ] Pin cert at pairing; reject unknown peers always
- [ ] Disable insecure cipher suites; require TLS 1.2+
- [ ] Certificate rotation policy documented (regenerate on revoke)
- [ ] Security review checklist doc

---

## M3 — Discovery

### M3.1 mDNS service
- [ ] Choose mDNS library for Dart/Flutter (e.g. `multicast_dns` or native)
- [ ] Advertise `_baton._tcp` with device name + port + version TXT
- [ ] Browse for `_baton._tcp` services on LAN
- [ ] Resolve service to IP + port
- [ ] Handle service add/remove/update events
- [ ] Tests with two processes on loopback (if feasible)

### M3.2 Manual fallback
- [ ] Manual IP + port entry UI and validation
- [ ] QR code fallback at pairing time (already in M2.3)
- [ ] Persist last-known address per peer for reconnect
- [ ] Reconnect with exponential backoff (1s,2s,4s,... capped 30s)

### M3.3 Discovery service abstraction
- [x] Define `DiscoveryService` interface in protocol/core
- [x] Provide mDNS impl + manual impl
- [x] Emit unified `PeerDiscovered` events
- [ ] Tests for event stream

---

## M4 — Transport Layer

### M4.1 TCP connection manager
- [x] Implement connection establishment (client side)
- [x] Implement accept loop (server side)
- [ ] Wrap socket in TLS
- [x] Length-prefixed JSON framing read/write
- [ ] Message id dedup / correlation
- [ ] Heartbeat every 10 s
- [ ] Detect dead connection (missed heartbeats) and reconnect
- [x] Graceful close

### M4.2 Connection state machine
- [ ] States: `disconnected`, `connecting`, `handshaking`, `connected`, `reconnecting`
- [ ] Emit state changes via stream
- [ ] Handle `Hello` exchange on connect
- [ ] Tests for transitions

### M4.3 Multi-peer support
- [ ] Support one connection per paired peer
- [ ] Connection pool keyed by peer id
- [ ] Broadcast `DEVICE_LIST` sync
- [ ] Tests for two peers

### M4.4 Backpressure & ordering
- [ ] Queue outbound messages when disconnected; flush on reconnect (with TTL)
- [ ] Ensure in-order delivery per connection
- [ ] Drop stale queued messages older than N seconds

---

## M5 — Handoff State Machine

### M5.1 Core state machine
- [x] Define states: IDLE, REQUESTED, PEER_READY, RELEASING, RELEASED, CONNECTING, ACTIVE, ROLLBACK
- [x] Implement transitions per design §2.3
- [x] Enforce single in-flight handoff per headphone (lock)
- [x] Reject new request during active handoff with `BUSY`
- [ ] Resolve conflicts: lower device ID loses; return `BUSY`
- [x] Timeout handling (default 10 s) -> ROLLBACK
- [x] Tests for happy path both directions

### M5.2 Handoff orchestrator (initiator side)
- [x] Send `HANDOFF_REQUEST`
- [x] Await `HANDOFF_READY` with timeout
- [x] Disconnect headphones locally
- [x] Send `RELEASED`
- [x] Await `HANDOFF_DONE` with timeout
- [x] On timeout/failure -> rollback (reconnect locally)
- [x] Emit progress events for UI

### M5.3 Handoff handler (receiver side)
- [ ] On `HANDOFF_REQUEST`: validate peer paired, device known
- [ ] Check availability (awake, in range, paired)
- [ ] Reply `HANDOFF_READY` or `HANDOFF_FAILED`/`BUSY`
- [ ] On `RELEASED`: connect to headphones, set default output
- [ ] Send `HANDOFF_DONE` on success or `HANDOFF_FAILED` on error
- [ ] Tests

### M5.4 Rollback logic
- [ ] Reconnect original owner to headphones
- [ ] Set default output back
- [ ] Emit rollback event + user notification
- [ ] Handle rollback failure (retry + alert)
- [ ] Tests for each failure case in §7

### M5.5 Multipoint shortcut detection
- [ ] Detect headphones connected to both devices simultaneously
- [ ] If multipoint: skip disconnect; pause playback on old device; switch default output on new
- [ ] Prefer multipoint path when available
- [ ] Tests for multipoint path

### M5.6 Cooldown & automatic rule gating
- [ ] Per-headphone cooldown timer (default 30 s)
- [ ] Gate automatic triggers behind cooldown
- [ ] Manual triggers bypass cooldown? (decide; document)
- [ ] Tests

---

## M6 — Mock Platform Layer & Tests

### M6.1 Bluetooth abstraction interface
- [x] Define `BluetoothController` interface: listDevices, isConnected, connect, disconnect, setDefaultOutput, isMultipoint
- [x] Define `AudioController` interface: pausePlayback, currentOutput, setDefaultOutput
- [ ] Define `DeviceIdentity` (id, name, os)

### M6.2 Fake implementations
- [x] `FakeBluetoothController` with in-memory device state
- [x] `FakeAudioController`
- [x] Simulate connect/disconnect delays and failures
- [x] Tests for handoff state machine using fakes (M2 milestone in design)

### M6.3 Integration tests
- [x] End-to-end handoff test phone->computer with fakes
- [ ] End-to-end reverse direction
- [x] Failure injection tests (disconnect fail, connect fail, timeout)
- [ ] Concurrency test (simultaneous requests)

---

## M7 — Android App

### M9.1 Project scaffold
- [ ] Create `android/` Flutter app
- [ ] Add `baton_protocol` dependency
- [ ] Configure Android permissions (Bluetooth, notifications, foreground service)
- [ ] Build & run on emulator/device

### M9.2 Kotlin platform channel
- [ ] Set up MethodChannel + EventChannel
- [ ] Implement foreground service to keep connection alive
- [ ] Bluetooth state & connected-device detection
- [ ] Disconnect headphones (strategy selection)
- [ ] Quick Settings tile service
- [ ] Expose all to Dart via channel

### M9.3 Disconnect strategy
- [ ] Implement multipoint detection (no disconnect needed)
- [ ] Implement Shizuku-based disconnect (hidden APIs) — optional
- [ ] Implement ADB-granted permission path
- [ ] Implement fallback: stop playback, let computer connect
- [ ] User setting to choose strategy
- [ ] Document trade-offs

### M9.4 Quick Settings tile
- [ ] Tile service implementation
- [ ] Toggle owner of selected headphones
- [ ] Reflect state in tile icon/label
- [ ] Handle locked device / permission prompts

### M9.5 Android UX (Flutter)
- [ ] Home screen: paired computers, current owner per headphone, "Send to computer" button
- [ ] Settings screen: automatic rules, cooldown, timeout, notification behaviour
- [ ] Pairing flow (scan QR)
- [ ] Device list sync

### M9.6 Notifications
- [ ] Foreground service notification
- [ ] Success/failure toasts
- [ ] Incoming call handling hook (M10)

### M9.7 Android packaging
- [ ] Release build (signed APK/AAB)
- [ ] Play Store listing prep (optional v1)
- [ ] Permission rationale screens

---

## M8 — Windows Desktop Client

### M8.1 Project scaffold
- [x] Go module builds on Windows
- [ ] Add Win32 FFI bindings
- [ ] Build runs on Windows dev machine

### M8.2 Bluetooth FFI
- [ ] Bind `BluetoothSetServiceState` and related APIs
- [ ] Enumerate paired devices
- [ ] Connect by toggling A2DP sink service state
- [ ] Disconnect
- [ ] Detect connection state
- [ ] Map errors to reason codes

### M8.3 Default audio output
- [ ] Use Windows audio policy API (MMDevice/EndpointVolume) to set default output
- [ ] Wait for device to appear after connect
- [ ] Retry with backoff

### M8.4 Tray & startup
- [ ] Tray icon with owner status and "pull here" action
- [ ] Start-at-login via registry/Task Scheduler
- [ ] Pairing window

### M8.5 Validation
- [ ] Test on Windows 10 and 11
- [ ] Test with LE Audio devices
- [ ] Document any device-specific quirks

### M8.6 Windows packaging
- [ ] MSIX build
- [ ] Installer (MSI or Inno Setup)
- [ ] Code signing plan (optional for v1)

---

## M9 — Linux Desktop Client

### M7.1 Project scaffold
- [x] Create `desktop/` Go module (Windows + Linux)
- [ ] Add `baton_protocol` dependency
- [ ] Configure window/tray plugins
- [ ] Build runs on Linux dev machine

### M7.2 BlueZ integration (D-Bus)
- [ ] Add D-Bus client library (e.g. `dbus` package)
- [ ] Enumerate paired Bluetooth devices
- [ ] Implement `connect` via `org.bluez.Device1.Connect`
- [ ] Implement `disconnect` via `org.bluez.Device1.Disconnect`
- [ ] Detect connection state changes via PropertiesChanged signals
- [ ] Map BlueZ errors to `HandoffFailureReason`

### M7.3 Default sink (PipeWire/PulseAudio)
- [ ] Detect audio server (PipeWire vs PulseAudio)
- [ ] Set default sink via `wpctl set-default` (PipeWire)
- [ ] Fallback to `pactl set-default-sink` (PulseAudio)
- [ ] Verify device appears before setting default
- [ ] Retry with backoff if device not yet ready

### M7.4 Tray & UI
- [ ] System tray icon (StatusNotifier) showing current owner
- [ ] Click tray to pull headphones here
- [ ] Small window for pairing, device list, rules
- [ ] Headless CLI mode (`baton --headless`)
- [ ] Start-at-login option (systemd user unit or .desktop autostart)

### M7.5 Linux packaging
- [ ] AppImage build script
- [ ] .deb build script
- [ ] Install/uninstall scripts
- [ ] Test on Ubuntu 22.04+ and Fedora

---

## M10 — Automatic Triggers

### M10.1 Trigger framework
- [ ] Define `Trigger` interface (evaluate, direction, enabled)
- [ ] Trigger manager with cooldown integration
- [ ] User opt-in toggles per trigger
- [ ] Tests

### M10.2 Computer starts audio playback
- [ ] Detect playback start on desktop (Windows/Linux audio session events)
- [ ] Trigger phone->computer handoff (opt-in)
- [ ] Avoid triggering during active call

### M10.3 Computer unlocked & phone idle
- [ ] Detect unlock event on desktop
- [ ] Detect phone idle (screen off / no interaction)
- [ ] Trigger phone->computer (opt-in)

### M10.4 Incoming call on phone
- [ ] Detect incoming call on Android (TelephonyManager / CallScreening)
- [ ] Trigger computer->phone (default on)
- [ ] Pause PC playback
- [ ] Return audio after call ends (optional)

### M10.5 Cooldown & ping-pong prevention
- [ ] Per-headphone cooldown (default 30s)
- [ ] Suppress reverse trigger within cooldown
- [ ] User-configurable cooldown
- [ ] Tests

---

## M11 — Failure Handling & Rollback

### M11.1 Failure cases (§7)
- [ ] Computer asleep/unreachable: fail fast, headphones stay on phone
- [ ] Disconnect fails on Android: report failure, stay on phone
- [ ] Computer connect fails: phone reconnects after timeout
- [ ] Headphones out of range: fail with `DEVICE_UNAVAILABLE`
- [ ] Both sides request at once: lower device ID loses; `BUSY`
- [ ] Connection drops mid-handoff: timeout then rollback

### M11.2 User feedback
- [ ] Toast/notification on success
- [ ] Clear error with reason on failure
- [ ] Automatic rollback message
- [ ] Retry affordance in UI

### M11.3 Resilience
- [ ] Reconnect with backoff for dropped transport
- [ ] Idempotent connect/disconnect calls
- [ ] State recovery after process restart (best-effort)

---

## M12 — UX Polish

### M12.1 Onboarding
- [ ] First-run wizard: name device, permissions, pairing
- [ ] Explain prerequisite (headphones paired with both devices)
- [ ] Show multipoint tip

### M12.2 Settings
- [ ] All toggles wired (rules, cooldown, timeout, notifications, disconnect strategy)
- [ ] Persist settings (shared prefs / config file)
- [ ] Export/import settings (optional)

### M12.3 Accessibility & localization
- [ ] i18n scaffolding (arb files)
- [ ] At least English + one more language
- [ ] Screen reader labels on key controls

### M12.4 Visual design
- [ ] App icon + tray icon assets
- [ ] Consistent theming (light/dark)
- [ ] Empty states and loading indicators

---

## M13 — Release

### M13.1 Versioning & changelog
- [ ] Finalize v1 version number
- [ ] Write CHANGELOG.md
- [ ] Tag release

### M13.2 Installers
- [ ] Windows MSIX/MSI published
- [ ] Linux AppImage/.deb published
- [ ] Android APK/AAB published

### M13.3 Documentation
- [ ] User guide (pairing, handoff, troubleshooting)
- [ ] FAQ for disconnect strategies
- [ ] Developer docs for protocol

### M13.4 Validation
- [ ] End-to-end manual test matrix across devices
- [ ] Performance: handoff completes in a few seconds
- [ ] No silent/stuck headphones in test runs

---

## M14 — Future: Audio Crossover (Later)

### M14.1 Streaming core
- [ ] Opus encode/decode pipeline
- [ ] UDP/RTP transport with jitter buffer
- [ ] Drift correction
- [ ] Target 20–60 ms latency over Wi-Fi
- [ ] Tests

### M14.2 Linux audio
- [ ] PipeWire virtual sinks/sources
- [ ] Route PC audio to phone speakers
- [ ] Route phone mic to PC

### M14.3 Windows audio
- [ ] WASAPI loopback capture first
- [ ] Signed virtual audio driver (later)

### M14.4 Android audio
- [ ] Playback capture
- [ ] Mic via platform APIs
- [ ] Document system-wide virtual output limitation (no root)

### M14.5 UX for crossover
- [ ] Mode toggle in UI
- [ ] Latency/quality settings
- [ ] Device selection

---

## Open Questions (from design §10)

- [ ] Decide default Android disconnect strategy (Shizuku vs fallback only)
- [ ] Decide desktop tech (Flutter vs native daemon + UI)
- [ ] Define multi-computer prioritization
- [ ] Verify "Baton" name availability (pub.dev, GitHub, Play Store)

---

## Suggested Execution Order

M0 → M1 → M2 → M3 → M4 → M5 → M6 → (M7 ∥ M8 ∥ M9) → M10 → M11 → M12 → M13 → M14

Linux/Windows/Android platform work (M7/M8/M9) can proceed in parallel once M6 (mocks + state machine) is green.
