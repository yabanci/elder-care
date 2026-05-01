// Live-alert subscription. Spins up only when a user is signed in,
// tears down on logout. UI listens via [liveAlertsProvider] for the
// latest event and shows a SnackBar / refreshes the alerts list.
//
// Reconnection policy: fixed 5s backoff with a single pending timer.
// onError + onDone both fire on a closed stream — naively scheduling
// in both gave us 2× retries per failure, which compounded to a flood
// of /api/events 401s during the unauthenticated splash window.
import "dart:async";

import "package:flutter/foundation.dart";
import "package:flutter_riverpod/flutter_riverpod.dart";

import "../api/api_client.dart";
import "providers.dart";

class LiveAlert {
  LiveAlert({
    required this.patientId,
    required this.alertId,
    required this.severity,
    required this.kind,
    required this.receivedAt,
  });
  final String patientId;
  final String alertId;
  final String severity;
  final String kind;
  final DateTime receivedAt;
}

class LiveAlertsNotifier extends StateNotifier<LiveAlert?> {
  LiveAlertsNotifier(this._api) : super(null) {
    _start();
  }
  final ApiClient _api;
  StreamSubscription<Map<String, dynamic>>? _sub;
  Timer? _reconnect;
  bool _disposed = false;

  Future<void> _start() async {
    if (_disposed) return;
    // Skip the round-trip entirely when there's no token; otherwise we
    // hammer the backend with 401s during the unauthenticated splash
    // window. The next login bumps authProvider, the provider rebuilds,
    // and a fresh notifier picks up the new token.
    final token = await _api.readToken();
    if (token == null || token.isEmpty) {
      _scheduleReconnect();
      return;
    }
    try {
      _sub = _api.sseStream("/api/events").listen(
        (event) {
          if (event["type"] != "alert") return;
          state = LiveAlert(
            patientId: event["patient_id"] as String? ?? "",
            alertId: event["alert_id"] as String? ?? "",
            severity: event["severity"] as String? ?? "",
            kind: event["kind"] as String? ?? "",
            receivedAt: DateTime.now(),
          );
        },
        onError: (Object e) {
          debugPrint("SSE error: $e");
          _scheduleReconnect();
        },
        onDone: _scheduleReconnect,
        cancelOnError: true,
      );
    } catch (e) {
      debugPrint("SSE connect failed: $e");
      _scheduleReconnect();
    }
  }

  /// Idempotent — only the first call after a failure arms the timer;
  /// subsequent callbacks (onError + onDone fire on the same close
  /// event) are no-ops.
  void _scheduleReconnect() {
    if (_disposed) return;
    if (_reconnect != null && _reconnect!.isActive) return;
    _reconnect = Timer(const Duration(seconds: 5), () {
      _reconnect = null;
      _start();
    });
  }

  @override
  void dispose() {
    _disposed = true;
    _reconnect?.cancel();
    _sub?.cancel();
    super.dispose();
  }
}

/// Tied to auth state — the notifier is recreated on each login/logout
/// transition so the SSE subscription picks up the fresh token (or stops
/// reconnecting once there's no token).
final liveAlertsProvider =
    StateNotifierProvider<LiveAlertsNotifier, LiveAlert?>((ref) {
  ref.watch(authProvider.select((s) => s.user?.id));
  final api = ref.watch(apiClientProvider);
  return LiveAlertsNotifier(api);
});
