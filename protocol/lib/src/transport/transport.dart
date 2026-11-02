import 'dart:async';
import 'dart:io';
import 'dart:typed_data';

import '../messages/message.dart';

/// A single framed message connection over TCP.
class TransportConnection {
  TransportConnection(this._socket) {
    _socket.listen(
      _onData,
      onError: (Object e, StackTrace st) {
        if (!_controller.isClosed) _controller.addError(e, st);
      },
      onDone: () {
        if (!_controller.isClosed) _controller.close();
      },
      cancelOnError: false,
    );
  }

  final Socket _socket;
  final _controller = StreamController<Message>.broadcast();
  final BytesBuilder _buffer = BytesBuilder();

  Stream<Message> get messages => _controller.stream;

  void _onData(Uint8List data) {
    _buffer.add(data);
    _drain();
  }

  void _drain() {
    final buf = _buffer.toBytes();
    int offset = 0;
    bool progressed = true;
    while (progressed) {
      progressed = false;
      if (buf.length - offset < 4) break;
      final view = ByteData.sublistView(buf, offset, offset + 4);
      final length = view.getUint32(0, Endian.big);
      if (buf.length - offset - 4 < length) break;
      try {
        final frame = FrameCodec.tryDecode(buf, offset);
        if (frame == null) break;
        _controller.add(frame.message);
        offset += frame.consumed;
        progressed = true;
      } catch (e) {
        if (!_controller.isClosed) _controller.addError(e);
        // drop buffer to avoid infinite loop
        offset = buf.length;
        progressed = false;
      }
    }
    // retain unconsumed tail
    if (offset > 0) {
      final tail = buf.sublist(offset);
      _buffer.clear();
      _buffer.add(tail);
    }
  }

  Future<void> send(Message message) async {
    _socket.add(FrameCodec.encode(message));
    await _socket.flush();
  }

  Future<void> close() async {
    await _socket.close();
    if (!_controller.isClosed) await _controller.close();
  }
}

/// Server that accepts Baton connections.
class BatonTransportServer {
  BatonTransportServer({required this.port});

  final int port;
  ServerSocket? _server;

  Future<void> start(void Function(TransportConnection conn) onConnect) async {
    _server = await ServerSocket.bind(InternetAddress.anyIPv4, port);
    _server!.listen((socket) {
      onConnect(TransportConnection(socket));
    });
  }

  Future<void> stop() async {
    await _server?.close();
    _server = null;
  }
}

/// Client helper.
class BatonTransportClient {
  static Future<TransportConnection> connect(String host, int port) async {
    final socket = await Socket.connect(host, port);
    return TransportConnection(socket);
  }
}

/// Reconnecting client with exponential backoff.
class ReconnectingTransport {
  ReconnectingTransport({
    required this.host,
    required this.port,
    required this.onConnection,
    this.initialDelay = const Duration(seconds: 1),
    this.maxDelay = const Duration(seconds: 30),
  });

  final String host;
  final int port;
  final void Function(TransportConnection conn) onConnection;
  final Duration initialDelay;
  final Duration maxDelay;

  bool _running = false;
  TransportConnection? _conn;

  Future<void> start() async {
    _running = true;
    var delay = initialDelay;
    while (_running) {
      try {
        _conn = await BatonTransportClient.connect(host, port);
        onConnection(_conn!);
        // wait for disconnect
        await _conn!.messages.drain();
        _conn = null;
      } catch (_) {
        _conn = null;
      }
      if (!_running) break;
      await Future.delayed(delay);
      delay = Duration(
        milliseconds: (delay.inMilliseconds * 2).clamp(0, maxDelay.inMilliseconds),
      );
      if (delay < initialDelay) delay = initialDelay;
    }
  }

  Future<void> stop() async {
    _running = false;
    await _conn?.close();
    _conn = null;
  }
}