// Smoke tests covering pure widgets and the i18n bundle. The full app
// can't easily be widget-tested without a backend mock, so we keep these
// targeted: i18n key coverage + login screen renders.
import "package:eldercare/l10n/strings.dart";
import "package:eldercare/screens/login_screen.dart";
import "package:flutter/material.dart";
import "package:flutter_riverpod/flutter_riverpod.dart";
import "package:flutter_test/flutter_test.dart";

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
}
