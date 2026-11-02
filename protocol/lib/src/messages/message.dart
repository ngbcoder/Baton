import 'dart:convert';
import 'dart:typed_data';

import 'types.dart';

/// Base class for all Baton protocol messages.
///
/// Every message carries a [type], a unique [id] and a timestamp [ts]
/// (epoch milliseconds). Subclasses add type-specific fields.
abstract class Message {
  /// Wire type discriminator.
  String get type;

  /// Unique message id (UUID v4 recommended).
  final String id;

  /// Epoch milliseconds when the message was created.
  final int ts;

  Message({required this.id, required this.ts});

  /// Serializes this message to a JSON-compatible map.
  Map<String, dynamic> toJson();

  /// Envelope helper: merges type/id/ts into a payload map.
  Map<String, dynamic> envelope(Map<String, dynamic> payload) {
    return <String, dynamic>{
      'type': type,
      'id': id,
      'ts': ts,
      ...payload,
    };
  }

  /// Decodes a [Message] from a JSON map using the registered factory.
  static Message fromJson(Map<String, dynamic> json) {
    final type = json['type'] as String?;
    if (type == null) {
      throw FormatException('Missing "type" field in message');
    }
    if (_messageFactories.isEmpty) {
      registerAllMessageTypes();
    }
    final factory = _messageFactories[type];
    if (factory == null) {
      throw FormatException('Unknown message type: $type');
    }
    return factory(json);
  }

  /// Registry of type -> factory. Populated by message type files.
  static final Map<String, Message Function(Map<String, dynamic>)>
      _messageFactories = <String, Message Function(Map<String, dynamic>)>{};

  static void register(String type, Message Function(Map<String, dynamic>) factory) {
    _messageFactories[type] = factory;
  }
}

/// Length-prefixed JSON framing helpers.
class FrameCodec {
  /// Encodes a [Message] into a length-prefixed byte buffer.
  static Uint8List encode(Message message) {
    final json = jsonEncode(message.toJson());
    final bytes = utf8.encode(json);
    final header = ByteData(4)..setUint32(0, bytes.length, Endian.big);
    final out = BytesBuilder();
    out.add(header.buffer.asUint8List());
    out.add(bytes);
    return out.toBytes();
  }

  /// Decodes a single frame from [buffer] starting at [offset].
  /// Returns the message and the number of bytes consumed, or null if
  /// the buffer does not yet contain a complete frame.
  static ({Message message, int consumed})? tryDecode(Uint8List buffer, [int offset = 0]) {
    if (buffer.length - offset < 4) return null;
    final view = ByteData.sublistView(buffer, offset, offset + 4);
    final length = view.getUint32(0, Endian.big);
    if (buffer.length - offset - 4 < length) return null;
    final jsonBytes = buffer.sublist(offset + 4, offset + 4 + length);
    final json = jsonDecode(utf8.decode(jsonBytes)) as Map<String, dynamic>;
    final message = Message.fromJson(json);
    return (message: message, consumed: 4 + length);
  }
}