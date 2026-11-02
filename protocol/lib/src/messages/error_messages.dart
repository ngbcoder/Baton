import 'types.dart';

extension HandoffFailureReasonMessage on HandoffFailureReason {
  String get userMessage {
    switch (this) {
      case HandoffFailureReason.deviceUnavailable:
        return 'Headphones are not available on the other device.';
      case HandoffFailureReason.busy:
        return 'A handoff is already in progress.';
      case HandoffFailureReason.timeout:
        return 'Handoff timed out; rolled back.';
      case HandoffFailureReason.disconnectFailed:
        return 'Could not disconnect headphones.';
      case HandoffFailureReason.connectFailed:
        return 'Could not connect headphones.';
      case HandoffFailureReason.outOfRange:
        return 'Headphones are out of range.';
      case HandoffFailureReason.rejected:
        return 'Handoff was rejected.';
      case HandoffFailureReason.unknown:
        return 'Unknown error.';
    }
  }
}