/// Abstraction over device certificate provisioning.
abstract class CertificateProvider {
  /// SHA-256 fingerprint of this device's certificate.
  String get fingerprint;

  /// Short numeric-ish confirmation code derived from both fingerprints.
  /// Implementations may compute a 6-digit code from fingerprints.
  String confirmationCode(String peerFingerprint);
}

/// Development provider returning a dummy fingerprint.
class DevCertificateProvider implements CertificateProvider {
  DevCertificateProvider(this.fingerprint);

  @override
  final String fingerprint;

  @override
  String confirmationCode(String peerFingerprint) {
    // Symmetric: order of fingerprints must not matter.
    final parts = [fingerprint, peerFingerprint]..sort();
    final combined = parts.join();
    var hash = 0;
    for (final c in combined.codeUnits) {
      hash = (hash * 31 + c) & 0x7fffffff;
    }
    return hash.toString().padLeft(6, '0').substring(0, 6);
  }
}