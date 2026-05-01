// Live-alert subscription. Auth-aware: spins up when a user is signed in,
// tears down on logout. UI listens via [liveAlertsProvider] for the
// latest event and shows a SnackBar / refreshes the alerts list.
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

/// Emits each alert as it arrives; null means "no event yet this session".
class LiveAlertsNotifier extends StateNotifier<LiveAlert?> {
  LiveAlertsNotifier(this._api) : super(null) {
    _start();
  }
  final ApiClient _api;
  StreamSubscription? _sub;

  Future<void> _start() async {
    try {
      final stream = _api.sseStream("/api/events");
      _sub = stream.listen((event) {
        if (event["type"] != "alert") return;
        state = LiveAlert(
          patientId: event["patient_id"] as String? ?? "",
          alertId: event["alert_id"] as String? ?? "",
          severity: event["severity"] as String? ?? "",
          kind: event["kind"] as String? ?? "",
          receivedAt: DateTime.now(),
        );
      }, onError: (e) {
        debugPrint("SSE error: $e");
        // Reconnect after a backoff so a network blip doesn't orphan us.
        Timer(const Duration(seconds: 5), _start);
      }, onDone: () {
        // Server closed the stream; reconnect.
        Timer(const Duration(seconds: 5), _start);
      });
    } catch (e) {
      debugPrint("SSE connect failed: $e");
      Timer(const Duration(seconds: 5), _start);
    }
  }

  @override
  void dispose() {
    _sub?.cancel();
    super.dispose();
  }
}

/// Created lazily when a user is authenticated. The [authProvider]
/// triggers re-creation on login/logout transitions.
final liveAlertsProvider =
    StateNotifierProvider<LiveAlertsNotifier, LiveAlert?>((ref) {
  // Tie our lifetime to the auth state — when the user logs out the
  // notifier is torn down (subscription cancelled) and a fresh one
  // spins up when they log back in.
  ref.watch(authProvider.select((s) => s.user?.id));
  final api = ref.watch(apiClientProvider);
  return LiveAlertsNotifier(api);
});
