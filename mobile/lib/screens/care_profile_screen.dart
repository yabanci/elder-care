import "package:flutter/material.dart";
import "package:flutter_riverpod/flutter_riverpod.dart";
import "package:go_router/go_router.dart";

import "../l10n/strings.dart";
import "../state/providers.dart";
import "care_home_screen.dart";

class CareProfileScreen extends ConsumerWidget {
  const CareProfileScreen({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final lang = ref.watch(langProvider);
    final user = ref.watch(authProvider).user;
    if (user == null) return const SizedBox();
    return Scaffold(
      appBar: AppBar(title: Text(tr("profile_title", lang))),
      body: ListView(
        padding: const EdgeInsets.all(16),
        children: [
          Card(
            child: Padding(
              padding: const EdgeInsets.all(16),
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Text(user.fullName,
                      style: const TextStyle(
                          fontWeight: FontWeight.bold, fontSize: 18)),
                  const SizedBox(height: 4),
                  Text(user.email,
                      style: Theme.of(context).textTheme.bodySmall),
                  const SizedBox(height: 8),
                  Text(user.role == "doctor"
                      ? tr("role_doctor", lang)
                      : tr("role_family", lang)),
                ],
              ),
            ),
          ),
          Card(
            child: Padding(
              padding: const EdgeInsets.all(16),
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Text(tr("profile_language", lang),
                      style: const TextStyle(fontWeight: FontWeight.bold)),
                  const SizedBox(height: 8),
                  SegmentedButton<String>(
                    segments: supportedLangs
                        .map((l) => ButtonSegment(
                            value: l, label: Text(langLabel[l] ?? l)))
                        .toList(),
                    selected: {lang},
                    onSelectionChanged: (s) async {
                      ref.read(langProvider.notifier).set(s.first);
                      try {
                        await ref
                            .read(authProvider.notifier)
                            .updateProfile({"lang": s.first});
                      } catch (_) {}
                    },
                  ),
                ],
              ),
            ),
          ),
          const SizedBox(height: 16),
          OutlinedButton.icon(
            icon: const Icon(Icons.logout),
            onPressed: () async {
              await ref.read(authProvider.notifier).logout();
              if (context.mounted) context.go("/login");
            },
            label: Text(tr("profile_logout", lang)),
          ),
        ],
      ),
      bottomNavigationBar: const CareNav(currentIndex: 2),
    );
  }
}
