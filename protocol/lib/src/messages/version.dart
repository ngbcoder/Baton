/// Protocol version constants and negotiation helpers.

const String protocolVersion = '1.0';
const int protocolMajorVersion = 1;
const int protocolMinorVersion = 0;

/// Capability flags exchanged in HELLO.
class Capability {
  static const String multipoint = 'multipoint';
  static const String audioCrossover = 'audioCrossover';
  static const String callHandling = 'callHandling';
}

/// Returns true if [other] is compatible with this implementation.
bool isCompatibleVersion(String other) {
  final parts = other.split('.');
  if (parts.isEmpty) return false;
  final otherMajor = int.tryParse(parts.first);
  if (otherMajor == null) return false;
  return otherMajor == protocolMajorVersion;
}