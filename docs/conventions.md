# Shared Conventions

## Package naming
- Dart: `baton_protocol` (shared protocol), `baton_core` (future).
- Go module: `github.com/ngbcoder/baton/desktop`.

## Logging
- Levels: debug, info, warn, error.
- Redact Bluetooth addresses and pairing secrets in logs.

## Error codes
See `HandoffFailureReason` (Dart) / `HandoffFailureReason` (Go): deviceUnavailable, busy, timeout, disconnectFailed, connectFailed, outOfRange, rejected, unknown.

## Versioning
- Protocol messages carry `protocolVersion` in `HELLO`. Breaking changes bump major version.