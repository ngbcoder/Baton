import 'dart:async';

import 'package:baton_protocol/baton_protocol.dart';
import 'package:test/test.dart';

class FakeBluetooth implements BluetoothController {
  final Set<String> paired = {};
  final Set<String> connected = {};
  bool multipoint = false;
  bool failDisconnect = false;
  bool failConnect = false;

  void addPaired(String address) => paired.add(address);

  @override
  Future<List<BtDevice>> pairedDevices() async =>
      paired.map((a) => BtDevice(address: a, name: a)).toList();

  @override
  Future<bool> isConnected(String address) async => connected.contains(address);

  @override
  Future<void> connect(String address) async {
    if (failConnect) throw StateError('connect failed');
    connected.add(address);
  }

  @override
  Future<void> disconnect(String address) async {
    if (failDisconnect) throw StateError('disconnect failed');
    connected.remove(address);
  }

  @override
  Future<bool> isMultipointConnected(String address) async => multipoint;
}

class FakeAudio implements AudioController {
  String? defaultOutput;
  int pauseCount = 0;
  @override
  Future<void> setDefaultOutput(String address) async => defaultOutput = address;
  @override
  Future<void> pausePlayback() async => pauseCount++;
}

void main() {
  test('phone to computer handoff succeeds', () async {
    const headphone = 'AA:BB:CC:DD:EE:FF';
    final phoneBt = FakeBluetooth();
    final pcBt = FakeBluetooth();
    phoneBt.addPaired(headphone);
    pcBt.addPaired(headphone);
    phoneBt.connected.add(headphone);

    final phoneAudio = FakeAudio();
    final pcAudio = FakeAudio();

    late HandoffController phone;
    late HandoffController pc;

    phone = HandoffController(
      localDeviceId: 'phone',
      bluetooth: phoneBt,
      audio: phoneAudio,
      send: (peerId, msg) async {
        await pc.handleMessage('phone', msg);
      },
    );
    pc = HandoffController(
      localDeviceId: 'pc',
      bluetooth: pcBt,
      audio: pcAudio,
      send: (peerId, msg) async {
        await phone.handleMessage('pc', msg);
      },
    );

    final result = await phone.initiate(
      peerId: 'pc',
      btAddress: headphone,
      deviceName: 'Headphones',
      direction: HandoffDirection.phoneToComputer,
    );

    expect(result.success, isTrue);
    expect(pcBt.connected.contains(headphone), isTrue);
    expect(phoneBt.connected.contains(headphone), isFalse);
    expect(pcAudio.defaultOutput, headphone);
  });

  test('rollback on pc connect failure', () async {
    const headphone = 'AA:BB:CC:DD:EE:FF';
    final phoneBt = FakeBluetooth();
    final pcBt = FakeBluetooth();
    phoneBt.addPaired(headphone);
    pcBt.addPaired(headphone);
    phoneBt.connected.add(headphone);
    pcBt.failConnect = true;

    final phoneAudio = FakeAudio();
    final pcAudio = FakeAudio();

    late HandoffController phone;
    late HandoffController pc;

    phone = HandoffController(
      localDeviceId: 'phone',
      bluetooth: phoneBt,
      audio: phoneAudio,
      send: (peerId, msg) async => pc.handleMessage('phone', msg),
    );
    pc = HandoffController(
      localDeviceId: 'pc',
      bluetooth: pcBt,
      audio: pcAudio,
      send: (peerId, msg) async => phone.handleMessage('pc', msg),
    );

    final result = await phone.initiate(
      peerId: 'pc',
      btAddress: headphone,
      deviceName: 'Headphones',
      direction: HandoffDirection.phoneToComputer,
    );

    expect(result.success, isFalse);
    expect(phoneBt.connected.contains(headphone), isTrue);
  });

  test('multipoint shortcut', () async {
    const headphone = 'AA:BB:CC:DD:EE:FF';
    final phoneBt = FakeBluetooth();
    final pcBt = FakeBluetooth();
    phoneBt.addPaired(headphone);
    pcBt.addPaired(headphone);
    phoneBt.multipoint = true;
    phoneBt.connected.add(headphone);

    final phoneAudio = FakeAudio();
    final pcAudio = FakeAudio();

    late HandoffController phone;
    late HandoffController pc;

    phone = HandoffController(
      localDeviceId: 'phone',
      bluetooth: phoneBt,
      audio: phoneAudio,
      send: (peerId, msg) async => pc.handleMessage('phone', msg),
    );
    pc = HandoffController(
      localDeviceId: 'pc',
      bluetooth: pcBt,
      audio: pcAudio,
      send: (peerId, msg) async => phone.handleMessage('pc', msg),
    );

    final result = await phone.initiate(
      peerId: 'pc',
      btAddress: headphone,
      deviceName: 'Headphones',
      direction: HandoffDirection.phoneToComputer,
    );

    expect(result.success, isTrue);
    expect(phoneAudio.pauseCount, 1);
    expect(phoneBt.connected.contains(headphone), isTrue);
  });
}