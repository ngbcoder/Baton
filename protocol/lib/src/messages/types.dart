import 'message.dart';

// ---------------------------------------------------------------------------
// HELLO
// ---------------------------------------------------------------------------

class Hello extends Message {
  static const String kType = 'HELLO';

  final String protocolVersion;
  final List<String> capabilities;

  Hello({
    required String id,
    required int ts,
    required this.protocolVersion,
    required this.capabilities,
  }) : super(id: id, ts: ts);

  @override
  String get type => kType;

  @override
  Map<String, dynamic> toJson() => envelope(<String, dynamic>{
        'protocolVersion': protocolVersion,
        'capabilities': capabilities,
      });

  factory Hello.fromJson(Map<String, dynamic> json) => Hello(
        id: json['id'] as String,
        ts: json['ts'] as int,
        protocolVersion: json['protocolVersion'] as String,
        capabilities: (json['capabilities'] as List).cast<String>(),
      );
}

// ---------------------------------------------------------------------------
// HANDOFF_REQUEST
// ---------------------------------------------------------------------------

enum HandoffDirection { phoneToComputer, computerToPhone }

class HandoffRequest extends Message {
  static const String kType = 'HANDOFF_REQUEST';

  final String btAddress;
  final String deviceName;
  final HandoffDirection direction;

  HandoffRequest({
    required String id,
    required int ts,
    required this.btAddress,
    required this.deviceName,
    required this.direction,
  }) : super(id: id, ts: ts);

  @override
  String get type => kType;

  @override
  Map<String, dynamic> toJson() => envelope(<String, dynamic>{
        'btAddress': btAddress,
        'deviceName': deviceName,
        'direction': direction.name,
      });

  factory HandoffRequest.fromJson(Map<String, dynamic> json) => HandoffRequest(
        id: json['id'] as String,
        ts: json['ts'] as int,
        btAddress: json['btAddress'] as String,
        deviceName: json['deviceName'] as String,
        direction: HandoffDirection.values.byName(json['direction'] as String),
      );
}

// ---------------------------------------------------------------------------
// HANDOFF_READY
// ---------------------------------------------------------------------------

class HandoffReady extends Message {
  static const String kType = 'HANDOFF_READY';

  HandoffReady({required String id, required int ts}) : super(id: id, ts: ts);

  @override
  String get type => kType;

  @override
  Map<String, dynamic> toJson() => envelope(<String, dynamic>{});

  factory HandoffReady.fromJson(Map<String, dynamic> json) =>
      HandoffReady(id: json['id'] as String, ts: json['ts'] as int);
}

// ---------------------------------------------------------------------------
// RELEASED
// ---------------------------------------------------------------------------

class Released extends Message {
  static const String kType = 'RELEASED';

  final String btAddress;

  Released({required String id, required int ts, required this.btAddress})
      : super(id: id, ts: ts);

  @override
  String get type => kType;

  @override
  Map<String, dynamic> toJson() => envelope(<String, dynamic>{
        'btAddress': btAddress,
      });

  factory Released.fromJson(Map<String, dynamic> json) => Released(
        id: json['id'] as String,
        ts: json['ts'] as int,
        btAddress: json['btAddress'] as String,
      );
}

// ---------------------------------------------------------------------------
// HANDOFF_DONE
// ---------------------------------------------------------------------------

class HandoffDone extends Message {
  static const String kType = 'HANDOFF_DONE';

  final String btAddress;

  HandoffDone({required String id, required int ts, required this.btAddress})
      : super(id: id, ts: ts);

  @override
  String get type => kType;

  @override
  Map<String, dynamic> toJson() => envelope(<String, dynamic>{
        'btAddress': btAddress,
      });

  factory HandoffDone.fromJson(Map<String, dynamic> json) => HandoffDone(
        id: json['id'] as String,
        ts: json['ts'] as int,
        btAddress: json['btAddress'] as String,
      );
}

// ---------------------------------------------------------------------------
// HANDOFF_FAILED
// ---------------------------------------------------------------------------

enum HandoffFailureReason {
  deviceUnavailable,
  busy,
  timeout,
  disconnectFailed,
  connectFailed,
  outOfRange,
  rejected,
  unknown,
}

class HandoffFailed extends Message {
  static const String kType = 'HANDOFF_FAILED';

  final String btAddress;
  final HandoffFailureReason reason;
  final String? detail;

  HandoffFailed({
    required String id,
    required int ts,
    required this.btAddress,
    required this.reason,
    this.detail,
  }) : super(id: id, ts: ts);

  @override
  String get type => kType;

  @override
  Map<String, dynamic> toJson() => envelope(<String, dynamic>{
        'btAddress': btAddress,
        'reason': reason.name,
        if (detail != null) 'detail': detail,
      });

  factory HandoffFailed.fromJson(Map<String, dynamic> json) => HandoffFailed(
        id: json['id'] as String,
        ts: json['ts'] as int,
        btAddress: json['btAddress'] as String,
        reason: HandoffFailureReason.values.byName(json['reason'] as String),
        detail: json['detail'] as String?,
      );
}

// ---------------------------------------------------------------------------
// BUSY
// ---------------------------------------------------------------------------

class Busy extends Message {
  static const String kType = 'BUSY';

  final String? btAddress;

  Busy({required String id, required int ts, this.btAddress}) : super(id: id, ts: ts);

  @override
  String get type => kType;

  @override
  Map<String, dynamic> toJson() => envelope(<String, dynamic>{
        if (btAddress != null) 'btAddress': btAddress,
      });

  factory Busy.fromJson(Map<String, dynamic> json) => Busy(
        id: json['id'] as String,
        ts: json['ts'] as int,
        btAddress: json['btAddress'] as String?,
      );
}

// ---------------------------------------------------------------------------
// DEVICE_LIST
// ---------------------------------------------------------------------------

class DeviceEntry {
  final String btAddress;
  final String name;
  /// 'phone' or 'computer' (owner id) or null if unknown.
  final String? owner;

  DeviceEntry({required this.btAddress, required this.name, this.owner});

  Map<String, dynamic> toJson() => <String, dynamic>{
        'btAddress': btAddress,
        'name': name,
        if (owner != null) 'owner': owner,
      };

  factory DeviceEntry.fromJson(Map<String, dynamic> json) => DeviceEntry(
        btAddress: json['btAddress'] as String,
        name: json['name'] as String,
        owner: json['owner'] as String?,
      );
}

class DeviceList extends Message {
  static const String kType = 'DEVICE_LIST';

  final List<DeviceEntry> devices;

  DeviceList({required String id, required int ts, required this.devices})
      : super(id: id, ts: ts);

  @override
  String get type => kType;

  @override
  Map<String, dynamic> toJson() => envelope(<String, dynamic>{
        'devices': devices.map((d) => d.toJson()).toList(),
      });

  factory DeviceList.fromJson(Map<String, dynamic> json) => DeviceList(
        id: json['id'] as String,
        ts: json['ts'] as int,
        devices: (json['devices'] as List)
            .cast<Map<String, dynamic>>()
            .map(DeviceEntry.fromJson)
            .toList(),
      );
}

// ---------------------------------------------------------------------------
// Registration
// ---------------------------------------------------------------------------

void registerAllMessageTypes() {
  Message.register(Hello.kType, Hello.fromJson);
  Message.register(HandoffRequest.kType, HandoffRequest.fromJson);
  Message.register(HandoffReady.kType, HandoffReady.fromJson);
  Message.register(Released.kType, Released.fromJson);
  Message.register(HandoffDone.kType, HandoffDone.fromJson);
  Message.register(HandoffFailed.kType, HandoffFailed.fromJson);
  Message.register(Busy.kType, Busy.fromJson);
  Message.register(DeviceList.kType, DeviceList.fromJson);
}