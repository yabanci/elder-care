import "package:flutter/material.dart";
import "package:flutter_localizations/flutter_localizations.dart";
import "package:flutter_riverpod/flutter_riverpod.dart";
import "package:go_router/go_router.dart";
import "package:intl/date_symbol_data_local.dart";

import "screens/care_home_screen.dart";
import "screens/care_link_screen.dart";
import "screens/care_messages_screen.dart";
import "screens/care_patient_detail_screen.dart";
import "screens/care_profile_screen.dart";
import "screens/login_screen.dart";
import "screens/messages_screen.dart";
import "screens/patient_alerts_screen.dart";
import "screens/patient_home_screen.dart";
import "screens/patient_meds_screen.dart";
import "screens/patient_messages_screen.dart";
import "screens/patient_metrics_screen.dart";
import "screens/patient_onboarding_screen.dart";
import "screens/patient_plans_screen.dart";
import "screens/patient_profile_screen.dart";
import "screens/register_screen.dart";
import "screens/splash_screen.dart";
import "state/live_alerts.dart";
import "theme.dart";

Future<void> main() async {
  WidgetsFlutterBinding.ensureInitialized();
  // Pre-load locale data for DateFormat in ru/kk/en.
  await initializeDateFormatting("ru_RU");
  await initializeDateFormatting("en_US");
  await initializeDateFormatting("kk_KZ");
  runApp(const ProviderScope(child: ElderCareApp()));
}

final _router = GoRouter(
  initialLocation: "/",
  routes: [
    GoRoute(path: "/", builder: (_, __) => const SplashScreen()),
    GoRoute(path: "/login", builder: (_, __) => const LoginScreen()),
    GoRoute(path: "/register", builder: (_, __) => const RegisterScreen()),
    GoRoute(path: "/patient", builder: (_, __) => const PatientHomeScreen()),
    GoRoute(
        path: "/patient/onboarding",
        builder: (_, __) => const PatientOnboardingScreen()),
    GoRoute(
        path: "/patient/metrics",
        builder: (_, __) => const PatientMetricsScreen()),
    GoRoute(
        path: "/patient/medications",
        builder: (_, __) => const PatientMedsScreen()),
    GoRoute(
        path: "/patient/plans",
        builder: (_, __) => const PatientPlansScreen()),
    GoRoute(
        path: "/patient/alerts",
        builder: (_, __) => const PatientAlertsScreen()),
    GoRoute(
        path: "/patient/messages",
        builder: (_, __) => const PatientMessagesScreen()),
    GoRoute(
      path: "/patient/messages/:otherID",
      builder: (ctx, state) =>
          ThreadScreen(otherId: state.pathParameters["otherID"]!),
    ),
    GoRoute(
        path: "/patient/profile",
        builder: (_, __) => const PatientProfileScreen()),
    GoRoute(path: "/care", builder: (_, __) => const CareHomeScreen()),
    GoRoute(path: "/care/link", builder: (_, __) => const CareLinkScreen()),
    GoRoute(
        path: "/care/messages",
        builder: (_, __) => const CareMessagesScreen()),
    GoRoute(
      path: "/care/messages/:otherID",
      builder: (ctx, state) =>
          ThreadScreen(otherId: state.pathParameters["otherID"]!),
    ),
    GoRoute(
        path: "/care/profile",
        builder: (_, __) => const CareProfileScreen()),
    GoRoute(
      path: "/care/patient/:id",
      builder: (ctx, state) =>
          CarePatientDetailScreen(patientId: state.pathParameters["id"]!),
    ),
  ],
);

class ElderCareApp extends ConsumerWidget {
  const ElderCareApp({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    // Touch the live-alerts provider so the SSE subscription stays active
    // whenever a user is signed in.
    ref.watch(liveAlertsProvider);

    return MaterialApp.router(
      title: "ElderCare",
      theme: buildTheme(),
      debugShowCheckedModeBanner: false,
      routerConfig: _router,
      localizationsDelegates: const [
        GlobalMaterialLocalizations.delegate,
        GlobalCupertinoLocalizations.delegate,
        GlobalWidgetsLocalizations.delegate,
      ],
      supportedLocales: const [
        Locale("ru"),
        Locale("kk"),
        Locale("en"),
      ],
      builder: (context, child) =>
          _LiveAlertListener(child: child ?? const SizedBox()),
    );
  }
}

/// Surfaces live alerts as a SnackBar regardless of which screen is open.
class _LiveAlertListener extends ConsumerWidget {
  const _LiveAlertListener({required this.child});
  final Widget child;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    ref.listen<LiveAlert?>(liveAlertsProvider, (prev, next) {
      if (next == null) return;
      final messenger = ScaffoldMessenger.maybeOf(context);
      if (messenger == null) return;
      messenger.showSnackBar(SnackBar(
        backgroundColor: next.severity == "critical"
            ? Colors.red.shade700
            : Colors.orange.shade700,
        duration: const Duration(seconds: 4),
        content: Text("⚠ ${next.kind} — ${next.severity}"),
      ));
    });
    return child;
  }
}
