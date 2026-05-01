import "package:flutter/material.dart";
import "package:flutter_riverpod/flutter_riverpod.dart";
import "package:go_router/go_router.dart";

import "../api/api_client.dart";
import "../l10n/strings.dart";
import "../models/models.dart";
import "../state/providers.dart";

class CareHomeScreen extends ConsumerStatefulWidget {
  const CareHomeScreen({super.key});
  @override
  ConsumerState<CareHomeScreen> createState() => _CareHomeScreenState();
}

class _CareHomeScreenState extends ConsumerState<CareHomeScreen> {
  bool _loading = true;
  List<LinkedPatient> _patients = [];

  @override
  void initState() {
    super.initState();
    _refresh();
  }

  Future<void> _refresh() async {
    setState(() => _loading = true);
    try {
      final raw = await ref.read(apiClientProvider).get("/api/patients");
      setState(() {
        _patients = (raw as List).map((e) => LinkedPatient.fromJson(e)).toList();
        _loading = false;
      });
    } on ApiException {
      setState(() => _loading = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    final lang = ref.watch(langProvider);
    return Scaffold(
      appBar: AppBar(title: Text(tr("care_title", lang))),
      floatingActionButton: FloatingActionButton.extended(
        onPressed: () async {
          await context.push("/care/link");
          await _refresh();
        },
        icon: const Icon(Icons.add),
        label: Text(tr("add", lang)),
      ),
      body: RefreshIndicator(
        onRefresh: _refresh,
        child: _loading
            ? const Center(child: CircularProgressIndicator())
            : _patients.isEmpty
                ? ListView(children: [
                    const SizedBox(height: 80),
                    Center(child: Text(tr("care_no_patients", lang))),
                  ])
                : ListView.builder(
                    padding: const EdgeInsets.all(16),
                    itemCount: _patients.length,
                    itemBuilder: (ctx, i) {
                      final p = _patients[i];
                      return Card(
                        child: ListTile(
                          leading: CircleAvatar(child: Text(_initials(p.fullName))),
                          title: Text(p.fullName),
                          subtitle: Text(
                              "${p.email}${p.phone != null ? "  ·  ${p.phone}" : ""}"),
                          trailing: const Icon(Icons.chevron_right),
                          onTap: () => context.push("/care/patient/${p.patientId}"),
                        ),
                      );
                    }),
      ),
      bottomNavigationBar: const CareNav(currentIndex: 0),
    );
  }
}

String _initials(String name) {
  final parts = name.split(" ").where((p) => p.isNotEmpty).toList();
  return parts.take(2).map((p) => p[0].toUpperCase()).join();
}

class CareNav extends ConsumerWidget {
  const CareNav({super.key, required this.currentIndex});
  final int currentIndex;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final lang = ref.watch(langProvider);
    return NavigationBar(
      selectedIndex: currentIndex,
      onDestinationSelected: (i) {
        switch (i) {
          case 0:
            context.go("/care");
            break;
          case 1:
            context.go("/care/messages");
            break;
          case 2:
            context.go("/care/profile");
            break;
        }
      },
      destinations: [
        NavigationDestination(
            icon: const Icon(Icons.group_outlined),
            selectedIcon: const Icon(Icons.group),
            label: tr("care_title", lang)),
        NavigationDestination(
            icon: const Icon(Icons.chat_bubble_outline),
            selectedIcon: const Icon(Icons.chat_bubble),
            label: tr("messages_title", lang)),
        NavigationDestination(
            icon: const Icon(Icons.account_circle_outlined),
            selectedIcon: const Icon(Icons.account_circle),
            label: tr("profile_title", lang)),
      ],
    );
  }
}
