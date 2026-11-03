import 'package:qr/qr.dart';

/// Renders [data] as a (crude) ASCII QR code.
String renderQrAscii(String data) {
  final qrCode = QrCode.fromData(
    data: data,
    errorCorrectLevel: QrErrorCorrectLevel.M,
  );
  final image = QrImage(qrCode);
  final buffer = StringBuffer();
  for (var row = 0; row < image.moduleCount; row++) {
    for (var col = 0; col < image.moduleCount; col++) {
      buffer.write(image.isDark(row, col) ? '##' : '  ');
    }
    buffer.writeln();
  }
  return buffer.toString();
}