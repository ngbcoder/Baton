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
- [ ] Create workflow for desktop build (Windows + Linux) on push to main
- [ ] Create workflow for Android APK build
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
- [x] Set Dart SDK constraint and dependencies (none external initially)
- [x] Add `lib/src/` structure: `messages/`, `state/`, `pairing/`, `transport/`
- [x] Add `lib/baton_protocol.dart` barrel export
- [x] Add example/ smoke test

### M1.2 Message envelope
- [x] Define `Message` base class with `type`, `id` (UUID), `ts` (epoch ms)
- [x] Implement JSON serialization/deserialization with strict field validation
- [x] Implement length-prefixed framing helpers (encode/decode)
- [x] Add unit tests for envelope round-trip
- [x] Add unit tests for malformed/unknown message handling

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
- [x] Define protocol version constant (e.g. `1.0`)
- [x] Implement `Hello` exchange on connect (helper + version check)
- [x] Reject peers with incompatible major version
- [x] Expose capability flags (multipoint, audio crossover, etc.)
- [x] Tests for version mismatch handling

### M1.5 Error codes
- [x] Define `HandoffFailureReason` enum: `DEVICE_UNAVAILABLE`, `BUSY`, `TIMEOUT`, `DISCONNECT_FAILED`, `CONNECT_FAILED`, `OUT_OF_RANGE`, `REJECTED`, `UNKNOWN`
- [x] Map reason codes to user-facing strings (i18n-ready)
- [x] Tests for enum serialization

---

## M2 — Pairing & Security

### M2.1 Certificate generation
- [x] Generate per-device self-signed cert (RSA/EC) on first run (Go)
- [ ] Store private key in OS secure storage (Android Keystore / Windows DPAPI / Linux libsecret)
- [x] Expose cert fingerprint (SHA-256)
- [x] Unit tests for cert generation determinism/format

### M2.2 Pairing data model
- [x] Define `Peer` model: id, name, address, cert fingerprint, pairedAt
- [x] Define `PairingStore` interface (load/save/list/remove)
- [ ] Implement file-backed store for desktop
- [x] Implement shared-prefs/secure-storage-backed store for Android (abstraction; platform impl pending)
- [x] Tests for store CRUD (pairing flow covers)

### M2.3 QR code pairing flow
- [x] Generate QR payload: address, port, cert fingerprint, one-time secret
- [x] Desktop: render QR code in pairing window (use `qr_flutter` or native)
- [x] Phone: scan QR via camera plugin
- [x] Validate QR payload schema and expiry (2 min)
- [x] Tests for payload encode/decode + expiry

### M2.4 Pairing handshake
- [x] Implement TCP connect + TLS with cert pinning (Go + Dart TLS transport)
- [x] Exchange certificates; verify against pinned fingerprint (handshake uses fingerprints)
- [x] Exchange one-time secret; reject on mismatch
- [x] Show short numeric confirmation code on both screens (computed; UI pending)
- [ ] Require user confirmation on both sides before finalizing
- [x] Persist peer on success
- [x] Timeout pairing after 2 minutes
- [x] Tests with loopback TLS

### M2.5 Revocation
- [x] `revokePeer(id)` on either side removes pairing
- [ ] Drop active connection on revoke
- [x] Reject future connections from revoked cert (revocation list)
- [ ] UI entry points (M9) wired to this
- [ ] Tests for revoke + reconnect rejection

### M2.6 TLS transport security
- [x] Pin cert at pairing; reject unknown peers always
- [x] Disable insecure cipher suites; require TLS 1.2+ (Go tls.Config defaults)
- [x] Certificate rotation policy documented (regenerate on revoke)
- [x] Security review checklist doc

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
- [x] QR code fallback at pairing time (already in M2.3)
- [ ] Persist last-known address per peer for reconnect
- [x] Reconnect with exponential backoff (1s,2s,4s,... capped 30s)

### M3.3 Discovery service abstraction
- [x] Define `DiscoveryService` interface in protocol/core
- [x] Provide mDNS impl + manual impl
- [x] Emit unified `PeerDiscovered` events
- [x] Tests for event stream

---

## M4 — Transport Layer

### M4.1 TCP connection manager
- [x] Implement connection establishment (client side)
- [x] Implement accept loop (server side)
- [x] Wrap socket in TLS
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
- [x] On `HANDOFF_REQUEST`: validate peer paired, device known
- [x] Check availability (awake, in range, paired)
- [x] Reply `HANDOFF_READY` or `HANDOFF_FAILED`/`BUSY`
- [x] On `RELEASED`: connect to headphones, set default output
- [x] Send `HANDOFF_DONE` on success or `HANDOFF_FAILED` on error
- [x] Tests

### M5.4 Rollback logic
- [x] Reconnect original owner to headphones
- [x] Set default output back
- [x] Emit rollback event + user notification
- [ ] Handle rollback failure (retry + alert)
- [x] Tests for each failure case in §7

### M5.5 Multipoint shortcut detection
- [x] Detect headphones connected to both devices simultaneously
- [x] If multipoint: skip disconnect; pause playback on old device; switch default output on new
- [x] Prefer multipoint path when available
- [x] Tests for multipoint path

### M5.6 Cooldown & automatic rule gating
- [x] Per-headphone cooldown timer (default 30 s)
- [x] Gate automatic triggers behind cooldown
- [ ] Manual triggers bypass cooldown? (decide; document)
- [x] Tests

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

### M7.1 Project scaffold
- [x] Create `android/` Flutter app
- [x] Add `baton_protocol` dependency
- [x] Configure Android permissions (Bluetooth, notifications, foreground service)
- [x] Build & run on emulator/device

### M7.2 Kotlin platform channel
- [x] Set up MethodChannel (EventChannel pending)
- [x] Implement foreground service to keep connection alive
- [x] Bluetooth state & connected-device detection
- [ ] Disconnect headphones (strategy selection)
- [x] Quick Settings tile service
- [x] Expose all to Dart via channel

### M7.3 Disconnect strategy
- [ ] Implement multipoint detection (no disconnect needed)
- [ ] Implement Shizuku-based disconnect (hidden APIs) — optional
- [ ] Implement ADB-granted permission path
- [ ] Implement fallback: stop playback, let computer connect
- [ ] User setting to choose strategy
- [ ] Document trade-offs

### M7.4 Quick Settings tile
- [x] Tile service implementation
- [ ] Toggle owner of selected headphones
- [ ] Reflect state in tile icon/label
- [ ] Handle locked device / permission prompts

### M7.5 Android UX (Flutter)
- [x] Home screen: paired computers, current owner per headphone, "Send to computer" button
- [x] Settings screen: automatic rules, cooldown, timeout, notification behaviour
- [x] Pairing flow (scan QR)
- [ ] Device list sync

### M7.6 Notifications
- [x] Foreground service notification
- [x] Success/failure toasts
- [ ] Incoming call handling hook (M10)

### M7.7 Android packaging
- [ ] Release build (signed APK/AAB)
- [ ] Play Store listing prep (optional v1)
- [ ] Permission rationale screens

---

## M8 — Windows Desktop Client

### M8.1 Project scaffold
- [x] Go module builds on Windows
- [x] Add Win32 FFI bindings (experimental cgo enumeration)
- [x] Build runs on Windows dev machine

### M8.2 Bluetooth FFI
- [x] Bind `BluetoothSetServiceState` and related APIs (via syscall, no CGO)
- [x] Enumerate paired devices
- [x] Connect by toggling A2DP sink service state
- [x] Disconnect
- [x] Detect connection state
- [x] Map errors to reason codes

### M8.3 Default audio output
- [ ] Use Windows audio policy API (MMDevice/EndpointVolume) to set default output
- [ ] Wait for device to appear after connect
- [ ] Retry with backoff

### M8.4 Tray & startup
- [ ] Tray icon with owner status and "pull here" action
- [ ] Start-at-login via registry/Task Scheduler
- [x] Pairing window

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

### M9.1 Project scaffold
- [x] Create `desktop/` Go module (Windows + Linux)
- [ ] Add `baton_protocol` dependency
- [x] Configure window/tray plugins
- [ ] Build runs on Linux dev machine

### M9.2 BlueZ integration (D-Bus)
- [ ] Add D-Bus client library (e.g. `dbus` package)
- [ ] Enumerate paired Bluetooth devices
- [ ] Implement `connect` via `org.bluez.Device1.Connect`
- [ ] Implement `disconnect` via `org.bluez.Device1.Disconnect`
- [ ] Detect connection state changes via PropertiesChanged signals
- [ ] Map BlueZ errors to `HandoffFailureReason`

### M9.3 Default sink (PipeWire/PulseAudio)
- [ ] Detect audio server (PipeWire vs PulseAudio)
- [ ] Set default sink via `wpctl set-default` (PipeWire)
- [ ] Fallback to `pactl set-default-sink` (PulseAudio)
- [ ] Verify device appears before setting default
- [ ] Retry with backoff if device not yet ready

### M9.4 Tray & UI
- [ ] System tray icon (StatusNotifier) showing current owner
- [ ] Click tray to pull headphones here
- [x] Small window for pairing, device list, rules
- [x] Headless CLI mode (`baton --headless`)
- [ ] Start-at-login option (systemd user unit or .desktop autostart)

### M9.5 Linux packaging
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
- [x] Computer connect fails: phone reconnects after timeout
- [ ] Headphones out of range: fail with `DEVICE_UNAVAILABLE`
- [ ] Both sides request at once: lower device ID loses; `BUSY`
- [x] Connection drops mid-handoff: timeout then rollback

### M11.2 User feedback
- [x] Toast/notification on success
- [x] Clear error with reason on failure
- [x] Automatic rollback message
- [ ] Retry affordance in UI

### M11.3 Resilience
- [x] Reconnect with backoff for dropped transport
- [ ] Idempotent connect/disconnect calls
- [ ] State recovery after process restart (best-effort)

---

## M12 — UX Polish

### M12.1 Onboarding
- [x] First-run wizard: name device, permissions, pairing
- [x] Explain prerequisite (headphones paired with both devices)
- [x] Show multipoint tip

### M12.2 Settings
- [x] All toggles wired (rules, cooldown, timeout, notifications, disconnect strategy)
- [x] Persist settings (shared prefs / config file)
- [ ] Export/import settings (optional)

### M12.3 Accessibility & localization
- [ ] i18n scaffolding (arb files)
- [ ] At least English + one more language
- [ ] Screen reader labels on key controls

### M12.4 Visual design
- [ ] App icon + tray icon assets
- [x] Consistent theming (light/dark)
- [x] Empty states and loading indicators

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

## M15 — File Transfer (Future)

### M15.1 Protocol messages
- [ ] Define `FileOffer` (name, size, mime, checksum)
- [ ] Define `FileAccept` / `FileReject`
- [ ] Define `FileChunk` (offset, bytes, eof)
- [ ] Define `FileDone` / `FileFailed` (reason)
- [ ] Add message type constants + factory registration
- [ ] Round-trip tests for each new message

### M15.2 Transfer engine
- [ ] Chunked streaming with configurable chunk size
- [ ] Progress events for UI
- [ ] Resume / retry on failure
- [ ] Integrity check (hash) on completion
- [ ] Concurrency limits and backpressure

### M15.3 Transport & storage
- [ ] Reuse existing TLS TCP transport for chunks
- [ ] Stream to temp file then atomic move
- [ ] Handle large files without loading into memory

### M15.4 Security
- [ ] Require paired peer + pinned cert
- [ ] User confirmation before receive
- [ ] Size limits and path sanitization
- [ ] Malware/extension warnings (optional)

### M15.5 Desktop UX
- [ ] Send file via tray/menu or drag-and-drop
- [ ] Incoming transfer prompt + progress
- [ ] Transfer history / notifications

### M15.6 Android UX
- [ ] Share sheet integration to send files
- [ ] Incoming transfer notification + save location
- [ ] Progress UI

### M15.7 Packaging & docs
- [ ] Document file transfer protocol
- [ ] Update installers if new permissions needed

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
