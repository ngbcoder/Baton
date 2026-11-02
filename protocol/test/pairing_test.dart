import 'package:baton_protocol/baton_protocol.dart';
import 'package:test/test.dart';

void main() {
  test('pairing flow over transport', () async {
    final server = BatonTransportServer(port: 4050);
    final storeA = InMemoryPairingStore();
    final storeB = InMemoryPairingStore();
    final certA = DevCertificateProvider('fpA');
    final certB = DevCertificateProvider('fpB');
    final mgrA = PairingManager(cert: certA, store: storeA, deviceName: 'A');
    final mgrB = PairingManager(cert: certB, store: storeB, deviceName: 'B');

    final qr = mgrA.createQr('127.0.0.1', 4050);

    await server.start((conn) async {
      final req = await conn.messages
          .where((m) => m is PairRequest)
          .cast<PairRequest>()
          .first;
      await mgrB.pairAsResponder(conn, req, qr.secret);
    });

    final client = await BatonTransportClient.connect('127.0.0.1', 4050);
    final peer = await mgrA.pairAsInitiator(client, qr);

    expect(peer.certFingerprint, 'fpB');
    final peersA = await storeA.list();
    final peersB = await storeB.list();
    expect(peersA.length, 1);
    expect(peersB.length, 1);

    await client.close();
    await server.stop();
  });
}