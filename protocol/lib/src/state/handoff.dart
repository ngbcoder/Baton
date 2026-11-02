import 'dart:async';

import 'package:uuid/uuid.dart';

import '../messages/message.dart';
import '../messages/types.dart';
import '../platform/bluetooth.dart';

/// High-level progress update emitted during a handoff.
class HandoffUpdate {
  final String btAddress;
  final HandoffState state;
  final HandoffFailureReason? reason;
  final String? detail;

  HandoffUpdate(this.btAddress, this.state, {this.reason, this.detail});
}

enum HandoffState {
  idle,
  requested,
  peerReady,
  releasing,
  released,
  connecting,
  active,
  rollback,
}

/// Result of an initiated handoff.
class HandoffResult {
  final bool success;
  final HandoffFailureReason? reason;
  final String? detail;
  HandoffResult(this.success, {this.reason, this.detail});
}

/// Coordinator for headphone handoffs.
class HandoffController {
  HandoffController({
    required this.localDeviceId,
    required this.bluetooth,
    required this.audio,
    required this.send,
    this.handoffTimeout = const Duration(seconds: 10),
    this.cooldown = const Duration(seconds: 30),
  });

  final String localDeviceId;
  final BluetoothController bluetooth;
  final AudioController audio;

  /// Sends a message to a peer. Returns when queued/sent.
  final Future<void> Function(String peerId, Message message) send;

  final Duration handoffTimeout;
  final Duration cooldown;

  final _uuid = const Uuid();


  // Active sessions keyed by btAddress.
  final Map<String, _Session> _sessions = {};

  // Cooldown tracking: last handoff completion time per btAddress.
  final Map<String, DateTime> _lastHandoff = {};

  final StreamController<HandoffUpdate> _updateController =
      StreamController<HandoffUpdate>.broadcast();

  Stream<HandoffUpdate> get updates => _updateController.stream;

  bool _isCoolingDown(String btAddress) {
    final last = _lastHandoff[btAddress];
    if (last == null) return false;
    return DateTime.now().difference(last) < cooldown;
  }

  void _emit(HandoffUpdate u) => _updateController.add(u);

  Future<HandoffResult> initiate({
    required String peerId,
    required String btAddress,
    required String deviceName,
    required HandoffDirection direction,
    bool force = false,
  }) async {
    // Cooldown only for automatic; manual passes force=true.
    if (!force && _isCoolingDown(btAddress)) {
      return HandoffResult(false,
          reason: HandoffFailureReason.busy, detail: 'cooldown');
    }

    final existing = _sessions[btAddress];
    if (existing != null && existing.state != HandoffState.idle) {
      return HandoffResult(false,
          reason: HandoffFailureReason.busy, detail: 'in-flight');
    }

    final session = _Session(btAddress: btAddress, peerId: peerId, direction: direction);
    _sessions[btAddress] = session;

    try {
      // REQUESTED
      session.state = HandoffState.requested;
      _emit(HandoffUpdate(btAddress, HandoffState.requested));


      session.readyCompleter = Completer<HandoffReady>();

      final req = HandoffRequest(
        id: _uuid.v4(),
        ts: DateTime.now().millisecondsSinceEpoch,
        btAddress: btAddress,
        deviceName: deviceName,
        direction: direction == HandoffDirection.phoneToComputer
            ? HandoffDirection.phoneToComputer
            : HandoffDirection.computerToPhone,
      );
      // Note: HandoffDirection enum in types.dart is named HandoffDirection. Use it.
      await send(peerId, req);

      // Wait for HANDOFF_READY
      try {
        await session.readyCompleter!.future.timeout(handoffTimeout);
      } catch (_) {
        await _rollback(session, HandoffFailureReason.timeout);
        return HandoffResult(false, reason: HandoffFailureReason.timeout);
      }
      session.state = HandoffState.peerReady;
      _emit(HandoffUpdate(btAddress, HandoffState.peerReady));

      // Multipoint shortcut
      final multipoint = await bluetooth.isMultipointConnected(btAddress);
      if (multipoint) {
        await audio.pausePlayback();
        // peer handles default output switch; we just mark active
        session.state = HandoffState.active;
        _emit(HandoffUpdate(btAddress, HandoffState.active));
        _lastHandoff[btAddress] = DateTime.now();
        return HandoffResult(true);
      }

      // RELEASING: disconnect locally
      session.state = HandoffState.releasing;
      _emit(HandoffUpdate(btAddress, HandoffState.releasing));
      try {
        await bluetooth.disconnect(btAddress);
      } catch (_) {
        await _rollback(session, HandoffFailureReason.disconnectFailed);
        await send(peerId, HandoffFailed(
          id: _uuid.v4(),
          ts: DateTime.now().millisecondsSinceEpoch,
          btAddress: btAddress,
          reason: HandoffFailureReason.disconnectFailed,
        ));
        return HandoffResult(false, reason: HandoffFailureReason.disconnectFailed);
      }

      // RELEASED
      session.state = HandoffState.released;
      _emit(HandoffUpdate(btAddress, HandoffState.released));
      session.doneCompleter = Completer<HandoffDone?>();
      await send(peerId, Released(
        id: _uuid.v4(),
        ts: DateTime.now().millisecondsSinceEpoch,
        btAddress: btAddress,
      ));

      // Wait for HANDOFF_DONE
      try {
        final done = await session.doneCompleter!.future.timeout(handoffTimeout);
        if (done == null) {
          final reason = session.failureReason ?? HandoffFailureReason.unknown;
          return HandoffResult(false, reason: reason);
        }
      } catch (_) {
        final reason = session.failureReason ?? HandoffFailureReason.timeout;
        await _rollback(session, reason);
        return HandoffResult(false, reason: reason);
      }

      session.state = HandoffState.active;
      _emit(HandoffUpdate(btAddress, HandoffState.active));
      _lastHandoff[btAddress] = DateTime.now();
      return HandoffResult(true);
    } catch (e) {
      await _rollback(session, HandoffFailureReason.unknown);
      return HandoffResult(false, reason: HandoffFailureReason.unknown, detail: e.toString());
    } finally {
      // Keep session for a bit? Remove on completion.
      // Remove only if active or rolled back.
      if (session.state == HandoffState.active || session.state == HandoffState.idle) {
        // keep active? remove to allow future
        _sessions.remove(btAddress);
      }
    }
  }

  Future<void> handleRequest(String peerId, HandoffRequest req) async {
    final btAddress = req.btAddress;

    // Busy check
    final existing = _sessions[btAddress];
    if (existing != null && existing.state != HandoffState.idle) {
      await send(peerId, Busy(
        id: _uuid.v4(),
        ts: DateTime.now().millisecondsSinceEpoch,
        btAddress: btAddress,
      ));
      return;
    }

    // Availability check
    final paired = await bluetooth.pairedDevices();
    final known = paired.any((d) => d.address == btAddress);
    // Consider available if paired (known) ; range check not possible here.
    if (!known) {
      await send(peerId, HandoffFailed(
        id: _uuid.v4(),
        ts: DateTime.now().millisecondsSinceEpoch,
        btAddress: btAddress,
        reason: HandoffFailureReason.deviceUnavailable,
      ));
      return;
    }

    final session = _Session(btAddress: btAddress, peerId: peerId, direction: req.direction == HandoffDirection.phoneToComputer ? HandoffDirection.phoneToComputer : HandoffDirection.computerToPhone);
    // store direction from request
    session.direction = req.direction == HandoffDirection.phoneToComputer
        ? HandoffDirection.phoneToComputer
        : HandoffDirection.computerToPhone;
    _sessions[btAddress] = session;

    session.state = HandoffState.peerReady;
    _emit(HandoffUpdate(btAddress, HandoffState.peerReady));

    await send(peerId, HandoffReady(
      id: _uuid.v4(),
      ts: DateTime.now().millisecondsSinceEpoch,
    ));
  }

  Future<void> handleMessage(String peerId, Message message) async {
    if (message is HandoffRequest) {
      await handleRequest(peerId, message);
      return;
    }
    if (message is HandoffReady) {
      // find session for this peer? We don't have btAddress in ready. Need track by peer.
      // Find the in-flight session for this peer.
      final session = _sessions.values.firstWhere(
        (s) => s.peerId == peerId && s.state == HandoffState.requested,
        orElse: () => throw StateError('no session'),
      );
      session.readyCompleter?.complete(message);
      return;
    }
    if (message is Released) {
      final btAddress = message.btAddress;

      final session = _sessions[btAddress];
      if (session == null) return;
      // Connect
      session.state = HandoffState.connecting;
      _emit(HandoffUpdate(btAddress, HandoffState.connecting));
      try {
        await bluetooth.connect(btAddress);
        await audio.setDefaultOutput(btAddress);
        session.state = HandoffState.active;
        _emit(HandoffUpdate(btAddress, HandoffState.active));
        await send(peerId, HandoffDone(
          id: _uuid.v4(),
          ts: DateTime.now().millisecondsSinceEpoch,
          btAddress: btAddress,
        ));
        _lastHandoff[btAddress] = DateTime.now();
      } catch (_) {
        session.state = HandoffState.idle;
        await send(peerId, HandoffFailed(
          id: _uuid.v4(),
          ts: DateTime.now().millisecondsSinceEpoch,
          btAddress: btAddress,
          reason: HandoffFailureReason.connectFailed,
        ));
      } finally {
        _sessions.remove(btAddress);
      }
      return;
    }
    if (message is HandoffDone) {

      final btAddress = message.btAddress;
      final session = _sessions[btAddress];
      if (session == null) return;

      session.doneCompleter?.complete(message);
      return;
    }
    if (message is HandoffFailed) {
      final btAddress = message.btAddress;
      final session = _sessions[btAddress];
      if (session == null) return;
      // initiator rollback handled by timeout? Here notify via update
      _emit(HandoffUpdate(btAddress, HandoffState.rollback, reason: message.reason, detail: message.detail));
      session.failureReason = message.reason;
      session.doneCompleter?.complete(null);
      await _rollback(session, message.reason);
      _sessions.remove(btAddress);
      return;
    }
    if (message is Busy) {
      // find session
      final session = (message.btAddress != null) ? _sessions[message.btAddress!] : _sessions.values.firstOrNull;
      if (session == null) return;
      _emit(HandoffUpdate(session.btAddress, HandoffState.rollback, reason: HandoffFailureReason.busy));
      await _rollback(session, HandoffFailureReason.busy);
      _sessions.remove(session.btAddress);
      return;
    }
  }

  Future<void> _rollback(_Session session, HandoffFailureReason reason) async {
    session.state = HandoffState.rollback;
    _emit(HandoffUpdate(session.btAddress, HandoffState.rollback, reason: reason));
    // Reconnect locally
    try {
      final connected = await bluetooth.isConnected(session.btAddress);
      if (!connected) {
        await bluetooth.connect(session.btAddress);
      }
      await audio.setDefaultOutput(session.btAddress);
    } catch (_) {
      // best effort
    }
    session.state = HandoffState.idle;
    _emit(HandoffUpdate(session.btAddress, HandoffState.idle));
  }

  void dispose() {
    _updateController.close();
  }
}

class _Session {
  final String btAddress;
  final String peerId;
  HandoffDirection direction;
  HandoffState state = HandoffState.idle;
  HandoffFailureReason? failureReason;
  Completer<HandoffReady>? readyCompleter;
  Completer<HandoffDone?>? doneCompleter;

  _Session({required this.btAddress, required this.peerId, required this.direction});
}

// Need HandoffDirection import alias from types. Use types.HandoffDirection.
// But we used HandoffDirection without import. Add import.
extension _FirstOrNull<T> on Iterable<T> {
  T? get firstOrNull => isEmpty ? null : first;
}

