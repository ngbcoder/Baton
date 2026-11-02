import 'package:baton_protocol/baton_protocol.dart';

void main() {
  final msg = Hello(
    id: 'example',
    ts: DateTime.now().millisecondsSinceEpoch,
    protocolVersion: '1.0',
    capabilities: const [],
  );
  print(msg.toJson());
}