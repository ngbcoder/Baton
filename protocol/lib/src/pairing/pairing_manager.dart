import 'dart:async';

import 'package:uuid/uuid.dart';

import '../messages/types.dart';
import '../transport/transport.dart';
import 'certificate.dart';
import 'peer.dart';
import 'store.dart';

class PairingManager {
  PairingManager({
    required this.cert,
    required this.store,
    required this.deviceName,
    this.pairingTimeout = const Duration(minutes: 2),
  });

  final CertificateProvider cert;
  final PairingStore store;
  final String deviceName;
  final Duration pairingTimeout;

  final _uuid = const Uuid();
  final Set<String> _revoked = <String>{};

  QrPayload createQr(String host, int port) {
    return QrPayload(
      address: host,
      port: port,
      certFingerprint: cert.fingerprint,
      secret: _uuid.v4(),
      expiresAt: DateTime.now().add(const Duration(minutes: 2)),
    );
  }

  Future<Peer> pairAsInitiator(
    TransportConnection conn,
    QrPayload qr,
  ) async {
    await conn.send(PairRequest(
      id: _uuid.v4(),
      ts: DateTime.now().millisecondsSinceEpoch,
      deviceName: deviceName,
      certFingerprint: cert.fingerprint,
      secret: qr.secret,
    ));

    final accept = await conn.messages
        .where((m) => m is PairAccept)
        .cast<PairAccept>()
        .first
        .timeout(pairingTimeout);

    // Verify secret? Initiator trusts QR; responder validated secret.
    final code = cert.confirmationCode(accept.certFingerprint);
    if (code != accept.confirmationCode) {
      throw StateError('confirmation code mismatch');
    }

    await conn.send(PairConfirm(
      id: _uuid.v4(),
      ts: DateTime.now().millisecondsSinceEpoch,
      confirmationCode: code,
    ));

    final peer = Peer(
      id: accept.certFingerprint,
      name: accept.deviceName,
      address: qr.address,
      port: qr.port,
      certFingerprint: accept.certFingerprint,
      pairedAt: DateTime.now(),
    );
    await store.add(peer);
    return peer;
  }

  bool isRevoked(String certFingerprint) => _revoked.contains(certFingerprint);

  Future<void> revokePeer(String peerId) async {
    final peer = await store.get(peerId);
    if (peer != null) {
      _revoked.add(peer.certFingerprint);
    }
    await store.remove(peerId);
  }

  Future<Peer> pairAsResponder(
    TransportConnection conn,
    PairRequest req,
    String expectedSecret,
  ) async {
    if (req.secret != expectedSecret) {
      throw StateError('invalid pairing secret');
    }
    if (req.certFingerprint == cert.fingerprint) {
      throw StateError('cannot pair with self');
    }

    final code = cert.confirmationCode(req.certFingerprint);
    await conn.send(PairAccept(
      id: _uuid.v4(),
      ts: DateTime.now().millisecondsSinceEpoch,
      deviceName: deviceName,
      certFingerprint: cert.fingerprint,
      confirmationCode: code,
    ));

    final confirm = await conn.messages
        .where((m) => m is PairConfirm)
        .cast<PairConfirm>()
        .first
        .timeout(pairingTimeout);

    if (confirm.confirmationCode != code) {
      throw StateError('confirmation code mismatch');
    }

    final peer = Peer(
      id: req.certFingerprint,
      name: req.deviceName,
      address: '', // filled later by discovery/manual
      port: 0,
      certFingerprint: req.certFingerprint,
      pairedAt: DateTime.now(),
    );
    await store.add(peer);
    return peer;
  }
}