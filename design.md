\# Baton — Design Document



\*\*Version:\*\* 0.1 (draft)

\*\*Platforms:\*\* Android (Flutter), Windows, Linux

\*\*Tagline:\*\* Pass your audio from one device to the next.



\---



\## 1. Overview



Baton is a cross-device companion in the spirit of Huawei Super Device and Xiaomi HyperOS Connect. Its first and main feature is \*\*Bluetooth audio device handoff\*\*: your headphones disconnect from the phone and the computer connects automatically, with no trip into Bluetooth settings.



Later versions add audio streaming between devices (using one device's speakers or mic from another) and other cross-device features.



\### 1.1 Goals



\- One-tap or fully automatic headphone handoff between phone and computer.

\- Handoff finishes in a few seconds and never leaves headphones silent or stuck.

\- Local-first: no account, no cloud, everything runs on the LAN.

\- Protocol is simple and modular so new features ship as plugins.



\### 1.2 Non-goals (v1)



\- Transferring Bluetooth pairing keys between devices.

\- Screen mirroring, remote input, app/window handoff.

\- iOS or macOS support.

\- Supporting devices off the local network.



\---



\## 2. Core Feature: Headphone Handoff



\### 2.1 Prerequisite



The headphones must already be \*\*paired with both devices\*\*. Baton only tells each side when to connect or disconnect; it never moves pairing keys.



\### 2.2 Flow (phone to computer)



1\. User triggers handoff on the phone (tile, notification, or automatic rule).

2\. Phone sends `HANDOFF\_REQUEST` with the headphone's Bluetooth address.

3\. Computer replies `HANDOFF\_READY` (it is awake, in range, and has the device paired).

4\. Phone disconnects the headphones.

5\. Phone sends `RELEASED`.

6\. Computer connects to the headphones and sets them as the default audio output.

7\. Computer sends `HANDOFF\_DONE`. If nothing arrives within the timeout (default 10 s), the phone reconnects the headphones (rollback).



The reverse direction (computer to phone) uses the same messages with roles swapped.



\### 2.3 State machine



```

IDLE -> REQUESTED -> PEER\_READY -> RELEASING -> RELEASED -> CONNECTING -> ACTIVE

&#x20;               \\          \\            \\            \\           \\

&#x20;                +----------+------------+------------+-----------+--> ROLLBACK -> IDLE

```



\- Only one handoff may be in flight per headphone.

\- Any timeout or error moves to ROLLBACK, where the original owner reconnects.

\- A new request during an active handoff is rejected with `BUSY`.



\### 2.4 Triggers



| Trigger | Direction | Notes |

|---|---|---|

| Quick Settings tile / notification button | Phone to computer | Manual |

| Tray menu item | Computer to phone | Manual |

| Computer starts audio playback | Phone to computer | Automatic, opt-in |

| Computer unlocked and phone idle | Phone to computer | Automatic, opt-in |

| Incoming call on phone | Computer to phone | Automatic, default on |



Automatic rules have a cooldown (default 30 s) to avoid ping-ponging.



\### 2.5 Multipoint shortcut



If the headphones support Bluetooth multipoint and are already connected to both devices, no disconnect is needed. Baton just pauses playback on the old device and switches the default output on the new one. This is faster and more reliable, so Baton detects it and prefers it.



\---



\## 3. Architecture



```

+-------------------+        TLS (pinned certs)        +-----------------------+

|  Android app      | <------------------------------> |  Desktop client       |

|  Flutter UI       |    JSON messages over TCP        |  Windows / Linux      |

|  Kotlin plugin    |                                  |  Dart + native bridge |

+-------------------+                                  +-----------------------+

&#x20;       |                                                        |

&#x20;  Bluetooth stack                                      Bluetooth stack

&#x20;  (Android)                                            (BlueZ / Windows)

```



\### 3.1 Repository layout



```

baton/

&#x20; protocol/     shared Dart package: messages, state machine, pairing

&#x20; android/      Flutter app + Kotlin platform plugin

&#x20; desktop/      Go desktop client (Windows + Linux), tray + settings

&#x20; docs/         this file and protocol notes

```



The `protocol/` package is shared by Android and desktop, so the handoff logic is written once.



\### 3.2 Discovery



\- mDNS service `\_baton.\_tcp` on the local network.

\- Fallback: manual IP entry or QR code at pairing time.



\### 3.3 Transport



\- One persistent TCP connection per paired device pair.

\- TLS using self-signed per-device certificates, pinned at pairing.

\- Length-prefixed JSON messages; every message has `type`, `id`, and `ts`.

\- Heartbeat every 10 s; reconnect with backoff.



\### 3.4 Message types (v1)



| Type | Purpose |

|---|---|

| `HELLO` | Version and capability exchange |

| `HANDOFF\_REQUEST` | Ask peer to take a device (carries Bluetooth address and name) |

| `HANDOFF\_READY` | Peer accepts and is prepared |

| `RELEASED` | Sender has disconnected the device |

| `HANDOFF\_DONE` | Receiver is connected and active |

| `HANDOFF\_FAILED` | Error with reason code |

| `BUSY` | A handoff is already in progress |

| `DEVICE\_LIST` | Sync of known headphones and their state |



\---



\## 4. Pairing and Security



1\. Computer shows a QR code with its address, certificate fingerprint, and a one-time secret.

2\. Phone scans the QR code and connects.

3\. Both sides exchange certificates and store the other's fingerprint.

4\. A short numeric code is shown on both screens for confirmation.



Rules:



\- All later connections must present the pinned certificate; unknown peers are rejected.

\- Pairing is local-network only and times out after 2 minutes.

\- Handoff requests are only accepted from paired devices.

\- The user can revoke a pairing from either side.



\---



\## 5. Platform Details



\### 5.1 Android (Flutter + Kotlin)



\- Flutter handles UI, settings, and the shared protocol package.

\- Kotlin platform channel handles:

&#x20; - Foreground service to keep the connection alive.

&#x20; - Bluetooth state and connected-device detection.

&#x20; - Quick Settings tile.

&#x20; - Disconnecting headphones.

\- \*\*Disconnect limitation:\*\* normal apps cannot programmatically disconnect an A2DP or headset device. Options in order of preference:

&#x20; 1. Multipoint shortcut (no disconnect needed).

&#x20; 2. Hidden API access via Shizuku or an ADB-granted permission.

&#x20; 3. Stop playback on the phone and let the computer connect (works when headphones drop the older link on a new connection).

&#x20; 4. Root (optional, not a target).



\### 5.2 Windows



\- Dart FFI to Win32 Bluetooth APIs.

\- Connect by toggling the A2DP sink service state for the device (`BluetoothSetServiceState`), which forces a reconnect.

\- Set default output through the Windows audio policy API after the device appears.

\- Tray icon with start-at-login.

\- \*To validate:\* behaviour across Windows 10 and 11, and with LE Audio devices.



\### 5.3 Linux



\- BlueZ over D-Bus: `org.bluez.Device1.Connect` and `Disconnect`.

\- Set default sink through PipeWire (`wpctl set-default`) or PulseAudio compatibility.

\- Tray via StatusNotifier where available; also runs headless with a small CLI.



\---



\## 6. UX



\### 6.1 Android



\- Home screen: paired computers, the current owner of each headphone, one big "Send to computer" button.

\- Settings: automatic rules, cooldown, timeout, notification behaviour.

\- Quick Settings tile: toggles the owner of the selected headphones.



\### 6.2 Desktop



\- Tray icon showing the current owner; click to pull the headphones here.

\- Small window for pairing, device list, and rules.



\### 6.3 Feedback



\- Short toast or notification on success.

\- Clear error with the reason on failure, and automatic rollback message.



\---



\## 7. Failure Handling



| Case | Behaviour |

|---|---|

| Computer asleep or unreachable | Request fails fast; headphones stay on the phone |

| Disconnect fails on Android | Report failure; stay on phone |

| Computer connect fails | Phone reconnects after timeout |

| Headphones out of range | Fail with `DEVICE\_UNAVAILABLE` |

| Both sides request at once | Lower device ID loses; `BUSY` returned |

| Connection drops mid-handoff | Timeout, then rollback |



\---



\## 8. Roadmap



| Milestone | Scope |

|---|---|

| M1 | Protocol package, pairing, discovery, handshake between phone and one computer |

| M2 | Handoff state machine tested with fake connect/disconnect |

| M3 | Linux: real BlueZ connect, default sink, tray |

| M4 | Windows: FFI connect, default output, tray |

| M5 | Android: Kotlin plugin, foreground service, tile, disconnect strategy |

| M6 | Automatic triggers, cooldown, call handling |

| M7 | Polish, installers (MSIX/AppImage or deb), release |

| Later | Audio streaming crossover (Opus over UDP), Windows virtual audio driver, other cross-device features |



\---



\## 9. Future: Audio Crossover



After handoff works, Baton can stream audio between devices:



\- PC audio to phone speakers or earbuds, and phone mic as PC microphone.

\- Opus-encoded audio over UDP/RTP with a jitter buffer and drift correction, aiming for 20 to 60 ms over Wi-Fi.

\- Linux: PipeWire virtual sinks and sources.

\- Windows: WASAPI loopback first; a signed virtual audio driver later.

\- Android: playback capture and mic via platform APIs (system-wide virtual output is not possible without root).



\---



\## 10. Open Questions



\- Which Android disconnect strategy is acceptable as the default (Shizuku vs. fallback only)?

\- Desktop client is Go (decided).

\- How should multiple computers be prioritized when more than one is eligible?

\- Package name and namespace availability for "Baton" on pub.dev, GitHub, and the Play Store.

