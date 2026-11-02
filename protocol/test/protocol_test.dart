import 'package:baton_protocol/baton_protocol.dart';
import 'package:test/test.dart';

void main() {
  test('message round-trip', () {
    final msg = Hello(
      id: '1',
      ts: 123,
      protocolVersion: protocolVersion,
      capabilities: const [Capability.multipoint],
    );
    final decoded = Message.fromJson(msg.toJson()) as Hello;
    expect(decoded.protocolVersion, protocolVersion);
    expect(decoded.capabilities, [Capability.multipoint]);
  });

  test('unknown message type throws', () {
    expect(
      () => Message.fromJson({'type': 'NOPE', 'id': '1', 'ts': 1}),
      throwsFormatException,
    );
  });

  test('malformed message throws', () {
    expect(() => Message.fromJson({}), throwsFormatException);
    expect(() => Message.fromJson({'type': 'HELLO'}), throwsA(isA<Object>()));
  });

  test('HandoffFailureReason serialization round-trip', () {
    for (final r in HandoffFailureReason.values) {
      final name = r.name;
      final parsed = HandoffFailureReason.values.byName(name);
      expect(parsed, r);
    }
  });

  test('version compatibility', () {
    expect(isCompatibleVersion('1.0'), isTrue);
    expect(isCompatibleVersion('2.0'), isFalse);
    expect(isCompatibleVersion('garbage'), isFalse);
  });
}