import "package:flutter/material.dart";
import "package:flutter_riverpod/flutter_riverpod.dart";
import "package:go_router/go_router.dart";

import "../l10n/strings.dart";
import "../state/providers.dart";

class PatientProfileScreen extends ConsumerWidget {
  const PatientProfileScreen({super.key});

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
                  if (user.inviteCode != null) ...[
                    const SizedBox(height: 16),
                    Text("${tr("profile_invite", lang)}:",
                        style: Theme.of(context).textTheme.bodySmall),
                    Text(user.inviteCode!,
                        style: const TextStyle(
                            fontSize: 22,
                            fontWeight: FontWeight.bold,
                            fontFamily: "monospace",
                            letterSpacing: 2)),
                  ],
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
                      final newLang = s.first;
                      ref.read(langProvider.notifier).set(newLang);
                      try {
                        await ref
                            .read(authProvider.notifier)
                            .updateProfile({"lang": newLang});
                      } catch (_) {}
                    },
                  ),
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
                  Text(tr("profile_tz", lang),
                      style: const TextStyle(fontWeight: FontWeight.bold)),
                  const SizedBox(height: 8),
                  Text(user.tz,
                      style: const TextStyle(
                          fontSize: 16, fontFamily: "monospace")),
                  const SizedBox(height: 8),
                  Text(
                    "Asia/Almaty, Europe/Moscow, UTC, …",
                    style: Theme.of(context).textTheme.bodySmall,
                  ),
                  const SizedBox(height: 8),
                  OutlinedButton(
                    onPressed: () => _editTz(context, ref, user.tz),
                    child: Text(tr("edit", lang)),
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
    );
  }

  Future<void> _editTz(
      BuildContext context, WidgetRef ref, String current) async {
    final controller = TextEditingController(text: current);
    final result = await showDialog<String>(
      context: context,
      builder: (ctx) => AlertDialog(
        title: const Text("Timezone (IANA)"),
        content: TextField(
          controller: controller,
          decoration: const InputDecoration(hintText: "Asia/Almaty"),
        ),
        actions: [
          TextButton(
              onPressed: () => Navigator.pop(ctx),
              child: const Text("Cancel")),
          ElevatedButton(
            onPressed: () => Navigator.pop(ctx, controller.text.trim()),
            child: const Text("Save"),
          ),
        ],
      ),
    );
    if (result == null || result.isEmpty || result == current) return;
    try {
      await ref.read(authProvider.notifier).updateProfile({"tz": result});
    } catch (_) {}
  }
}
