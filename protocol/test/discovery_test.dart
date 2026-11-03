import 'package:baton_protocol/baton_protocol.dart';
import 'package:test/test.dart';

void main() {
  test('manual discovery emits peer', () async {
    final d = ManualDiscovery();
    final future = d.peers.first;
    d.addPeer(DiscoveredPeer(name: 'pc', address: '127.0.0.1', port: 4040));
    final p = await future;
    expect(p.name, 'pc');
    expect(p.port, 4040);
    await d.stop();
  });
}
