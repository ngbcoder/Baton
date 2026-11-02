import 'package:baton_protocol/baton_protocol.dart';
import 'package:test/test.dart';

void main() {
  test('transport round trip', () async {
    final server = BatonTransportServer(port: 4040);
    final received = <Message>[];
    await server.start((conn) {
      conn.messages.listen(received.add);
    });

    final client = await BatonTransportClient.connect('127.0.0.1', 4040);
    final msg = Heartbeat(id: '1', ts: 1);
    await client.send(msg);

    await Future.delayed(const Duration(milliseconds: 100));
    expect(received.length, 1);
    expect(received.first, isA<Heartbeat>());

    await client.close();
    await server.stop();
  });
}