import 'dart:convert';
import 'dart:io';

import 'peer.dart';
import 'store.dart';

/// File-backed JSON store for paired peers.
class FilePairingStore implements PairingStore {
  FilePairingStore(this.file);

  final File file;

  Future<Map<String, dynamic>> _read() async {
    if (!await file.exists()) return <String, dynamic>{};
    final raw = await file.readAsString();
    if (raw.trim().isEmpty) return <String, dynamic>{};
    return jsonDecode(raw) as Map<String, dynamic>;
  }

  Future<void> _write(Map<String, dynamic> data) async {
    await file.parent.create(recursive: true);
    await file.writeAsString(jsonEncode(data));
  }

  @override
  Future<List<Peer>> list() async {
    final data = await _read();
    final peers = data['peers'] as List<dynamic>? ?? <dynamic>[];
    return peers
        .cast<Map<String, dynamic>>()
        .map(Peer.fromJson)
        .toList();
  }

  @override
  Future<Peer?> get(String id) async {
    final peers = await list();
    for (final p in peers) {
      if (p.id == id) return p;
    }
    return null;
  }

  @override
  Future<void> add(Peer peer) async {
    final data = await _read();
    final peers = (data['peers'] as List<dynamic>? ?? <dynamic>[]).cast<Map<String, dynamic>>().toList();
    peers.removeWhere((p) => p['id'] == peer.id);
    peers.add(peer.toJson());
    data['peers'] = peers;
    await _write(data);
  }

  @override
  Future<void> remove(String id) async {
    final data = await _read();
    final peers = (data['peers'] as List<dynamic>? ?? <dynamic>[]).cast<Map<String, dynamic>>().toList();
    peers.removeWhere((p) => p['id'] == id);
    data['peers'] = peers;
    await _write(data);
  }

  @override
  Future<void> clear() async {
    await _write(<String, dynamic>{'peers': <dynamic>[]});
  }
}