import 'package:baton_protocol/baton_protocol.dart';
import 'package:flutter/material.dart';
import 'package:mobile_scanner/mobile_scanner.dart';

class PairingPage extends StatefulWidget {
  const PairingPage({super.key});

  @override
  State<PairingPage> createState() => _PairingPageState();
}

class _PairingPageState extends State<PairingPage> {
  String? _payload;
  String? _error;

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: const Text('Scan pairing QR')),
      body: Column(
        children: [
          Expanded(
            child: MobileScanner(
              onDetect: (capture) {
                final barcode = capture.barcodes.isNotEmpty ? capture.barcodes.first : null;
                final raw = barcode?.rawValue;
                if (raw == null) return;
                try {
                  final qr = QrPayload.decode(raw);
                  if (qr.isExpired) {
                    setState(() => _error = 'QR code expired');
                    return;
                  }
                  setState(() {
                    _payload = raw;
                    _error = null;
                  });
                } catch (e) {
                  setState(() => _error = 'Invalid QR: $e');
                }
              },
            ),
          ),
          if (_error != null)
            Padding(
              padding: const EdgeInsets.all(12),
              child: Text(_error!, style: const TextStyle(color: Colors.red)),
            ),
          if (_payload != null)
            Padding(
              padding: const EdgeInsets.all(12),
              child: Text('Scanned: $_payload'),
            ),
        ],
      ),
    );
  }
}