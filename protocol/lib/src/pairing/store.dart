import 'peer.dart';

/// Abstract store for paired peers.
abstract class PairingStore {
  Future<List<Peer>> list();
  Future<Peer?> get(String id);
  Future<void> add(Peer peer);
  Future<void> remove(String id);
  Future<void> clear();
}

/// In-memory store (useful for tests and ephemeral use).
class InMemoryPairingStore implements PairingStore {
  final Map<String, Peer> _peers = {};

  @override
  Future<List<Peer>> list() async => _peers.values.toList();

  @override
  Future<Peer?> get(String id) async => _peers[id];

  @override
  Future<void> add(Peer peer) async => _peers[peer.id] = peer;

  @override
  Future<void> remove(String id) async => _peers.remove(id);

  @override
  Future<void> clear() async => _peers.clear();
}