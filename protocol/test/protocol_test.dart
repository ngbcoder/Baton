import 'package:baton_protocol/baton_protocol.dart';
import 'package:test/test.dart';

void main() {
  test('message round-trip', () {
    final msg = Hello(
      id: '1',
      ts: 123,
      protocolVersion: '1.0',
      capabilities: const ['multipoint'],
    );
    final decoded = Message.fromJson(msg.toJson()) as Hello;
    expect(decoded.protocolVersion, '1.0');
    expect(decoded.capabilities, ['multipoint']);
  });
}