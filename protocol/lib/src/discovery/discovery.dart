import 'dart:async';

/// Event emitted when a peer is discovered or lost.
class DiscoveredPeer {
  final String name;
  final String address;
  final int port;
  final String? txtVersion;

  DiscoveredPeer({
    required this.name,
    required this.address,
    required this.port,
    this.txtVersion,
  });
}

/// Abstract discovery service.
abstract class DiscoveryService {
  /// Starts advertising this device (if supported).
  Future<void> advertise({required String name, required int port});

  /// Starts browsing for peers.
  Future<void> browse();

  /// Stream of discovered peers.
  Stream<DiscoveredPeer> get peers;

  /// Stops all discovery.
  Future<void> stop();
}

/// Manual discovery: user supplies IP/port. Useful fallback and tests.
class ManualDiscovery implements DiscoveryService {
  final _controller = StreamController<DiscoveredPeer>.broadcast();

  @override
  Stream<DiscoveredPeer> get peers => _controller.stream;

  @override
  Future<void> advertise({required String name, required int port}) async {}

  @override
  Future<void> browse() async {}

  void addPeer(DiscoveredPeer peer) => _controller.add(peer);

  @override
  Future<void> stop() async => _controller.close();
}