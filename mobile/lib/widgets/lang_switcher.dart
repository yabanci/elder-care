import "package:flutter/material.dart";
import "package:flutter_riverpod/flutter_riverpod.dart";

import "../l10n/strings.dart";
import "../state/providers.dart";

/// Top-right language picker. Persists selection via [LangNotifier].
class LangSwitcher extends ConsumerWidget {
  const LangSwitcher({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final current = ref.watch(langProvider);
    return PopupMenuButton<String>(
      icon: const Icon(Icons.language),
      tooltip: tr("profile_language", current),
      onSelected: (l) => ref.read(langProvider.notifier).set(l),
      itemBuilder: (context) => supportedLangs
          .map((l) => PopupMenuItem(
                value: l,
                child: Row(
                  children: [
                    if (l == current) const Icon(Icons.check, size: 18),
                    if (l == current) const SizedBox(width: 8),
                    Text(langLabel[l] ?? l),
                  ],
                ),
              ))
          .toList(),
    );
  }
}
