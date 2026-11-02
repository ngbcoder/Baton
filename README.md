# Baton

![CI](https://github.com/ngbcoder/Baton/actions/workflows/ci.yml/badge.svg)

Pass your audio from one device to the next.

Baton is a cross-device companion (in the spirit of Huawei Super Device and Xiaomi HyperOS Connect). Its first and main feature is **Bluetooth audio device handoff**: your headphones disconnect from the phone and the computer connects automatically, with no trip into Bluetooth settings.

## Platforms
- Android (Flutter + Kotlin)
- Windows (Go)
- Linux (Go)

## Status
Early development (v0.1). See `design.md` and `todo.md`.

## Quick start
- Protocol package (Dart): `cd protocol && dart test`
- Desktop (Go): `cd desktop && go test ./...`
- Android: `cd android && flutter pub get && flutter build apk --debug`

## License
MIT (see `LICENSE`).