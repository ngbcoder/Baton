/// Platform abstraction for Bluetooth operations.
library;

class BtDevice {
  final String address;
  final String name;
  BtDevice({required this.address, required this.name});

  @override
  bool operator ==(Object other) =>
      other is BtDevice && other.address == address;

  @override
  int get hashCode => address.hashCode;
}

abstract class BluetoothController {
  /// Lists paired devices.
  Future<List<BtDevice>> pairedDevices();

  /// Whether [address] is currently connected.
  Future<bool> isConnected(String address);

  /// Connect to [address].
  Future<void> connect(String address);

  /// Disconnect from [address].
  Future<void> disconnect(String address);

  /// Whether the device supports multipoint and is connected to both sides.
  Future<bool> isMultipointConnected(String address);
}

abstract class AudioController {
  /// Set the default audio output to [address].
  Future<void> setDefaultOutput(String address);

  /// Pause current playback (best-effort).
  Future<void> pausePlayback();
}