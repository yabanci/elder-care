import "package:flutter/material.dart";
import "package:flutter_riverpod/flutter_riverpod.dart";

import "care_home_screen.dart";
import "messages_screen.dart";

class CareMessagesScreen extends ConsumerWidget {
  const CareMessagesScreen({super.key});
  @override
  Widget build(BuildContext context, WidgetRef ref) {
    return const ThreadsScreen(bottomNav: CareNav(currentIndex: 1));
  }
}
