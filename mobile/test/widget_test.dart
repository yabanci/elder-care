// Smoke tests covering pure widgets and the i18n bundle, plus a
// regression guard for the SSE retry-storm bug.
import "dart:async";

import "package:eldercare/api/api_client.dart";
import "package:eldercare/l10n/strings.dart";
import "package:eldercare/screens/login_screen.dart";
import "package:eldercare/state/live_alerts.dart";
import "package:flutter/material.dart";
import "package:flutter_riverpod/flutter_riverpod.dart";
import "package:flutter_test/flutter_test.dart";

class _CountingApi extends ApiClient {
  _CountingApi() : super(baseUrl: "http://127.0.0.1:1");
  int sseCalls = 0;
  String? _token;
  @override
  Future<String?> readToken() async => _token;
  void setTestToken(String? t) => _token = t;
  @override
  Stream<Map<String, dynamic>> sseStream(String path) async* {
    sseCalls++;
    throw ApiException(401, "stub");
  }
}

void main() {
  testWidgets("Login screen renders fields and submit button", (tester) async {
    await tester.pumpWidget(
      const ProviderScope(
        child: MaterialApp(
          home: LoginScreen(),
          locale: Locale("ru"),
        ),
      ),
    );
    await tester.pump();

    expect(find.text("Вход"), findsOneWidget);
    expect(find.text("Email"), findsOneWidget);
    expect(find.text("Пароль"), findsOneWidget);
    expect(find.text("Войти"), findsOneWidget);
  });

  test("i18n bundle resolves canonical keys in every language", () {
    const ruKey = "alerts_title";
    expect(tr(ruKey, "ru"), isNot(equals(ruKey)),
        reason: "ru is canonical and must define $ruKey");
    expect(tr(ruKey, "kk"), isNot(equals(ruKey)),
        reason: "kk should translate $ruKey");
    expect(tr(ruKey, "en"), isNot(equals(ruKey)),
        reason: "en should translate $ruKey");
  });

  test("Reason-code keys map to localised text in all languages", () {
    const codes = [
      "safety_below_min",
      "safety_above_max",
      "safety_warn_low",
      "safety_warn_high",
      "baseline_warn_z2",
      "baseline_crit_z3",
      "condition_warn",
      "condition_crit",
      "cold_start",
      "legacy",
    ];
    for (final c in codes) {
      for (final lang in supportedLangs) {
        final key = "reason_$c";
        expect(tr(key, lang), isNot(equals(key)),
            reason: "missing $key for $lang");
      }
    }
  });

  test("Unknown lang falls back to ru, unknown key returns the key", () {
    expect(tr("login_title", "fr"), equals("Вход"));
    expect(tr("totally_unknown", "ru"), equals("totally_unknown"));
  });

  test("LiveAlertsNotifier skips SSE entirely when no token (no flood)",
      () async {
    final api = _CountingApi(); // token stays null
    final n = LiveAlertsNotifier(api);
    // Wait long enough for the first _start() to run + finish.
    await Future<void>.delayed(const Duration(milliseconds: 50));
    expect(api.sseCalls, 0,
        reason:
            "Without a token we must NOT open the SSE connection — was the "
            "source of the 10+ /api/events/sec storm during splash.");
    n.dispose();
  });

  test("LiveAlertsNotifier dispose stops in-flight reconnects", () async {
    final api = _CountingApi()..setTestToken("dummy");
    final n = LiveAlertsNotifier(api);
    // Let the first sseStream throw → _scheduleReconnect arms a 5s timer.
    await Future<void>.delayed(const Duration(milliseconds: 50));
    final before = api.sseCalls;
    n.dispose();
    // Wait past the reconnect window; sseCalls must NOT increase.
    await Future<void>.delayed(const Duration(seconds: 6));
    expect(api.sseCalls, equals(before),
        reason: "Disposed notifier must not fire a queued reconnect.");
  });
}
