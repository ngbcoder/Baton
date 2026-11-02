import 'dart:convert';

/// A paired peer device.
class Peer {
  final String id;
  final String name;
  final String address; // host
  final int port;
  final String certFingerprint;
  final DateTime pairedAt;

  Peer({
    required this.id,
    required this.name,
    required this.address,
    required this.port,
    required this.certFingerprint,
    required this.pairedAt,
  });

  Map<String, dynamic> toJson() => <String, dynamic>{
        'id': id,
        'name': name,
        'address': address,
        'port': port,
        'certFingerprint': certFingerprint,
        'pairedAt': pairedAt.toIso8601String(),
      };

  factory Peer.fromJson(Map<String, dynamic> json) => Peer(
        id: json['id'] as String,
        name: json['name'] as String,
        address: json['address'] as String,
        port: json['port'] as int,
        certFingerprint: json['certFingerprint'] as String,
        pairedAt: DateTime.parse(json['pairedAt'] as String),
      );

  @override
  String toString() => 'Peer($id, $name, $address:$port)';

  @override
  bool operator ==(Object other) => other is Peer && other.id == id;

  @override
  int get hashCode => id.hashCode;
}

/// QR code payload exchanged during pairing.
class QrPayload {
  final String address;
  final int port;
  final String certFingerprint;
  final String secret;
  final DateTime expiresAt;

  QrPayload({
    required this.address,
    required this.port,
    required this.certFingerprint,
    required this.secret,
    required this.expiresAt,
  });

  bool get isExpired => DateTime.now().isAfter(expiresAt);

  Map<String, dynamic> toJson() => <String, dynamic>{
        'address': address,
        'port': port,
        'certFingerprint': certFingerprint,
        'secret': secret,
        'expiresAt': expiresAt.toIso8601String(),
      };

  factory QrPayload.fromJson(Map<String, dynamic> json) => QrPayload(
        address: json['address'] as String,
        port: json['port'] as int,
        certFingerprint: json['certFingerprint'] as String,
        secret: json['secret'] as String,
        expiresAt: DateTime.parse(json['expiresAt'] as String),
      );

  String encode() => jsonEncode(toJson());

  static QrPayload decode(String raw) =>
      QrPayload.fromJson(jsonDecode(raw) as Map<String, dynamic>);
}