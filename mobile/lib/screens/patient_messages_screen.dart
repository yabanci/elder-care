import "package:flutter/material.dart";
import "package:flutter_riverpod/flutter_riverpod.dart";

import "messages_screen.dart";
import "patient_home_screen.dart";

class PatientMessagesScreen extends ConsumerWidget {
  const PatientMessagesScreen({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    return const ThreadsScreen(bottomNav: PatientNav(currentIndex: 4));
  }
}
