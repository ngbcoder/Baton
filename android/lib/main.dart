import 'package:flutter/material.dart';
import 'package:flutter/services.dart';

void main() => runApp(const BatonApp());

class BatonApp extends StatelessWidget {
  const BatonApp({super.key});

  @override
  Widget build(BuildContext context) {
    return MaterialApp(
      title: 'Baton',
      theme: ThemeData(useMaterial3: true),
      home: const HomePage(),
    );
  }
}

class HomePage extends StatefulWidget {
  const HomePage({super.key});

  @override
  State<HomePage> createState() => _HomePageState();
}

class _HomePageState extends State<HomePage> {
  static const _channel = MethodChannel('baton');
  String _status = 'Ready';

  Future<void> _send() async {
    try {
      final ok = await _channel.invokeMethod<bool>('sendHandoff');
      setState(() => _status = ok == true ? 'Handoff requested' : 'Failed');
    } on PlatformException catch (e) {
      setState(() => _status = 'Error: ${e.message}');
    }
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: const Text('Baton')),
      body: Center(
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            Text(_status),
            const SizedBox(height: 16),
            FilledButton(
              onPressed: _send,
              child: const Text('Send headphones to computer'),
            ),
          ],
        ),
      ),
    );
  }
}