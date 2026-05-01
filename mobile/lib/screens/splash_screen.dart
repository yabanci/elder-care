import "package:flutter/material.dart";
import "package:flutter_riverpod/flutter_riverpod.dart";
import "package:go_router/go_router.dart";

import "../l10n/strings.dart";
import "../state/providers.dart";

class SplashScreen extends ConsumerWidget {
  const SplashScreen({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final auth = ref.watch(authProvider);
    final lang = ref.watch(langProvider);

    // Bootstrap completed → redirect.
    if (!auth.loading) {
      WidgetsBinding.instance.addPostFrameCallback((_) {
        if (!context.mounted) return;
        if (auth.user == null) {
          context.go("/login");
        } else if (auth.user!.role == "patient" && !auth.user!.onboarded) {
          context.go("/patient/onboarding");
        } else if (auth.user!.role == "patient") {
          context.go("/patient");
        } else {
          context.go("/care");
        }
      });
    }

    return Scaffold(
      body: Center(
        child: Column(
          mainAxisAlignment: MainAxisAlignment.center,
          children: [
            Container(
              width: 96,
              height: 96,
              decoration: BoxDecoration(
                color: Theme.of(context).colorScheme.primary,
                borderRadius: BorderRadius.circular(24),
              ),
              child: const Center(
                child: Icon(Icons.favorite, color: Colors.white, size: 48),
              ),
            ),
            const SizedBox(height: 24),
            Text(tr("app_name", lang),
                style: Theme.of(context).textTheme.headlineMedium),
            const SizedBox(height: 8),
            Text(tr("app_tagline", lang),
                style: Theme.of(context).textTheme.bodyLarge),
            const SizedBox(height: 32),
            const CircularProgressIndicator(),
          ],
        ),
      ),
    );
  }
}
