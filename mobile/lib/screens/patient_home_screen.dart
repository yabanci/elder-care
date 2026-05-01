import "package:flutter/material.dart";
import "package:flutter_riverpod/flutter_riverpod.dart";
import "package:go_router/go_router.dart";
import "package:intl/intl.dart";

import "../api/api_client.dart";
import "../l10n/strings.dart";
import "../models/models.dart";
import "../state/providers.dart";
import "../widgets/metric_meta.dart";

class PatientHomeScreen extends ConsumerStatefulWidget {
  const PatientHomeScreen({super.key});
  @override
  ConsumerState<PatientHomeScreen> createState() => _PatientHomeScreenState();
}

class _PatientHomeScreenState extends ConsumerState<PatientHomeScreen> {
  bool _loading = true;
  String? _error;
  List<Metric> _summary = [];
  List<Alert> _alerts = [];
  List<MedScheduleItem> _schedule = [];

  @override
  void initState() {
    super.initState();
    _refresh();
  }

  Future<void> _refresh() async {
    final api = ref.read(apiClientProvider);
    setState(() {
      _loading = true;
      _error = null;
    });
    try {
      final s = await api.get("/api/metrics/summary");
      final a = await api.get("/api/alerts");
      final sch = await api.get("/api/medications/today");
      setState(() {
        _summary = (s as List).map((e) => Metric.fromJson(e)).toList();
        _alerts = (a as List).map((e) => Alert.fromJson(e)).toList();
        _schedule =
            (sch as List).map((e) => MedScheduleItem.fromJson(e)).toList();
        _loading = false;
      });
    } on ApiException catch (e) {
      setState(() {
        _error = e.message;
        _loading = false;
      });
    }
  }

  Future<void> _takeDose(MedScheduleItem item) async {
    final api = ref.read(apiClientProvider);
    try {
      await api.post(
        "/api/medications/${item.medicationId}/log",
        data: {
          "scheduled_at": item.scheduledAt.toUtc().toIso8601String(),
          "status": "taken",
        },
      );
      await _refresh();
    } catch (_) {/* surface elsewhere */}
  }

  @override
  Widget build(BuildContext context) {
    final lang = ref.watch(langProvider);
    final user = ref.watch(authProvider).user;
    if (user == null) return const SizedBox();

    final hour = DateTime.now().hour;
    final greeting = hour < 12
        ? tr("greeting_morning", lang)
        : hour < 18
            ? tr("greeting_day", lang)
            : tr("greeting_evening", lang);
    final unack = _alerts.where((a) => !a.acknowledged).toList();

    return Scaffold(
      appBar: AppBar(
        title: Text(tr("app_name", lang)),
        actions: [
          IconButton(
            icon: const Icon(Icons.account_circle),
            onPressed: () => context.push("/patient/profile"),
          ),
        ],
      ),
      body: RefreshIndicator(
        onRefresh: _refresh,
        child: _loading
            ? const Center(child: CircularProgressIndicator())
            : ListView(
                padding: const EdgeInsets.all(16),
                children: [
                  if (_error != null)
                    Card(
                      color: Theme.of(context).colorScheme.errorContainer,
                      child: Padding(
                        padding: const EdgeInsets.all(12),
                        child: Text(_error!),
                      ),
                    ),
                  Text("$greeting,",
                      style: Theme.of(context).textTheme.bodyLarge),
                  Text(user.fullName.split(" ").first,
                      style: Theme.of(context).textTheme.headlineMedium),
                  const SizedBox(height: 16),
                  if (unack.isNotEmpty)
                    Card(
                      color: const Color(0xFFFEF2F2),
                      child: ListTile(
                        leading: const Icon(Icons.warning, color: Colors.red),
                        title: Text(
                          "${unack.length} ${tr("alerts_new", lang)}",
                          style: const TextStyle(fontWeight: FontWeight.bold),
                        ),
                        subtitle: Text(tr("reason_${unack.first.reasonCode}", lang)),
                        onTap: () => context.push("/patient/alerts"),
                      ),
                    ),
                  const SizedBox(height: 16),
                  Text(tr("recent_metrics", lang),
                      style: Theme.of(context).textTheme.titleLarge),
                  const SizedBox(height: 8),
                  _SummaryGrid(metrics: _summary, lang: lang),
                  const SizedBox(height: 16),
                  Row(
                    children: [
                      Expanded(
                        child: Text(tr("today_meds", lang),
                            style: Theme.of(context).textTheme.titleLarge),
                      ),
                      TextButton(
                        onPressed: () => context.push("/patient/medications"),
                        child: Text(tr("show_all", lang)),
                      ),
                    ],
                  ),
                  const SizedBox(height: 8),
                  if (_schedule.isEmpty)
                    Card(
                      child: Padding(
                        padding: const EdgeInsets.all(16),
                        child: Text(tr("today_no_meds", lang)),
                      ),
                    )
                  else
                    ..._schedule.map((s) => _DoseCard(
                          item: s,
                          lang: lang,
                          onTake: () => _takeDose(s),
                        )),
                  const SizedBox(height: 16),
                  Text(tr("quick_entry", lang),
                      style: Theme.of(context).textTheme.titleLarge),
                  const SizedBox(height: 8),
                  GridView.count(
                    crossAxisCount: 2,
                    crossAxisSpacing: 12,
                    mainAxisSpacing: 12,
                    childAspectRatio: 1.6,
                    physics: const NeverScrollableScrollPhysics(),
                    shrinkWrap: true,
                    children: const ["pulse", "bp_sys", "glucose", "temperature"]
                        .map((k) => _QuickEntryButton(kind: k))
                        .toList(),
                  ),
                  const SizedBox(height: 8),
                  Center(
                    child: TextButton(
                      onPressed: () => context.push("/patient/metrics"),
                      child: Text(tr("all_metrics", lang)),
                    ),
                  ),
                  if (user.inviteCode != null) ...[
                    const SizedBox(height: 16),
                    Card(
                      color: const Color(0xFFECFDF5),
                      child: Padding(
                        padding: const EdgeInsets.all(16),
                        child: Column(
                          crossAxisAlignment: CrossAxisAlignment.start,
                          children: [
                            Text(tr("invite_code_label", lang),
                                style: const TextStyle(
                                    fontWeight: FontWeight.bold)),
                            const SizedBox(height: 4),
                            Text(user.inviteCode!,
                                style: const TextStyle(
                                    fontSize: 22,
                                    fontFamily: "monospace",
                                    fontWeight: FontWeight.bold,
                                    letterSpacing: 2)),
                            const SizedBox(height: 6),
                            Text(tr("invite_code_hint", lang),
                                style: Theme.of(context).textTheme.bodySmall),
                          ],
                        ),
                      ),
                    ),
                  ],
                ],
              ),
      ),
      bottomNavigationBar: const _PatientNav(currentIndex: 0),
    );
  }
}

class _SummaryGrid extends StatelessWidget {
  const _SummaryGrid({required this.metrics, required this.lang});
  final List<Metric> metrics;
  final String lang;

  @override
  Widget build(BuildContext context) {
    final byKind = {for (final m in metrics) m.kind: m};
    const order = [
      "pulse",
      "bp_sys",
      "bp_dia",
      "glucose",
      "spo2",
      "temperature",
      "weight"
    ];
    return GridView.count(
      crossAxisCount: 2,
      crossAxisSpacing: 12,
      mainAxisSpacing: 12,
      childAspectRatio: 1.4,
      physics: const NeverScrollableScrollPhysics(),
      shrinkWrap: true,
      children: order.map((k) {
        final meta = metricMeta[k]!;
        final m = byKind[k];
        return Card(
          child: Padding(
            padding: const EdgeInsets.all(12),
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Row(children: [
                  Text(meta.emoji,
                      style: const TextStyle(fontSize: 18)),
                  const SizedBox(width: 6),
                  Text(tr("metric_kind_$k", lang),
                      style: Theme.of(context).textTheme.bodySmall),
                ]),
                const Spacer(),
                Text(
                  m == null ? "—" : meta.fmt(m.value),
                  style: const TextStyle(
                      fontSize: 26, fontWeight: FontWeight.bold),
                ),
                Text(meta.unit,
                    style: Theme.of(context).textTheme.bodySmall),
              ],
            ),
          ),
        );
      }).toList(),
    );
  }
}

class _DoseCard extends StatelessWidget {
  const _DoseCard(
      {required this.item, required this.lang, required this.onTake});
  final MedScheduleItem item;
  final String lang;
  final VoidCallback onTake;

  @override
  Widget build(BuildContext context) {
    final taken = item.status == "taken";
    final missed = item.status == "missed";
    return Card(
      child: Padding(
        padding: const EdgeInsets.all(12),
        child: Row(
          children: [
            SizedBox(
              width: 64,
              child: Text(
                DateFormat.Hm().format(item.scheduledAt),
                style: const TextStyle(
                    fontSize: 22, fontWeight: FontWeight.bold),
              ),
            ),
            Expanded(
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Text(item.name,
                      style: const TextStyle(
                          fontSize: 16, fontWeight: FontWeight.w600)),
                  if (item.dosage != null) Text(item.dosage!),
                ],
              ),
            ),
            if (taken)
              const Icon(Icons.check_circle, color: Colors.green, size: 28)
            else
              ElevatedButton(
                onPressed: onTake,
                style: missed
                    ? ElevatedButton.styleFrom(
                        backgroundColor:
                            Theme.of(context).colorScheme.error)
                    : null,
                child: Text(tr("take_dose", lang)),
              ),
          ],
        ),
      ),
    );
  }
}

class _QuickEntryButton extends ConsumerWidget {
  const _QuickEntryButton({required this.kind});
  final String kind;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final lang = ref.watch(langProvider);
    final meta = metricMeta[kind]!;
    return InkWell(
      onTap: () async {
        final v = await _showQuickEntry(context, lang, kind);
        if (v == null) return;
        try {
          await ref
              .read(apiClientProvider)
              .post("/api/metrics", data: {"kind": kind, "value": v});
          if (context.mounted) {
            ScaffoldMessenger.of(context).showSnackBar(
                SnackBar(content: Text("✓ ${meta.fmt(v)} ${meta.unit}")));
          }
        } catch (_) {}
      },
      child: Card(
        child: Padding(
          padding: const EdgeInsets.all(8),
          child: Column(
            mainAxisAlignment: MainAxisAlignment.center,
            children: [
              Text(meta.emoji, style: const TextStyle(fontSize: 26)),
              const SizedBox(height: 4),
              Text(tr("metric_kind_$kind", lang),
                  textAlign: TextAlign.center,
                  style:
                      const TextStyle(fontSize: 13, fontWeight: FontWeight.w600)),
            ],
          ),
        ),
      ),
    );
  }
}

Future<double?> _showQuickEntry(
    BuildContext context, String lang, String kind) {
  final controller = TextEditingController();
  final meta = metricMeta[kind]!;
  return showModalBottomSheet<double>(
    context: context,
    isScrollControlled: true,
    builder: (ctx) {
      return Padding(
        padding: EdgeInsets.fromLTRB(
            16, 16, 16, MediaQuery.of(ctx).viewInsets.bottom + 16),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [
            Text("${meta.emoji} ${tr("metric_kind_$kind", lang)}",
                style: Theme.of(ctx).textTheme.titleLarge),
            const SizedBox(height: 12),
            TextField(
              controller: controller,
              autofocus: true,
              keyboardType:
                  const TextInputType.numberWithOptions(decimal: true),
              decoration: InputDecoration(
                labelText: tr("metrics_value", lang),
                suffixText: meta.unit,
              ),
            ),
            const SizedBox(height: 16),
            ElevatedButton(
              onPressed: () {
                final v = double.tryParse(
                    controller.text.replaceAll(",", "."));
                if (v == null) return;
                Navigator.pop(ctx, v);
              },
              child: Text(tr("save", lang)),
            ),
          ],
        ),
      );
    },
  );
}

class _PatientNav extends ConsumerWidget {
  const _PatientNav({required this.currentIndex});
  final int currentIndex;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final lang = ref.watch(langProvider);
    return NavigationBar(
      selectedIndex: currentIndex,
      onDestinationSelected: (i) {
        switch (i) {
          case 0:
            context.go("/patient");
            break;
          case 1:
            context.go("/patient/medications");
            break;
          case 2:
            context.go("/patient/plans");
            break;
          case 3:
            context.go("/patient/alerts");
            break;
          case 4:
            context.go("/patient/messages");
            break;
        }
      },
      destinations: [
        NavigationDestination(
            icon: const Icon(Icons.home_outlined),
            selectedIcon: const Icon(Icons.home),
            label: tr("app_name", lang)),
        NavigationDestination(
            icon: const Icon(Icons.medication_outlined),
            selectedIcon: const Icon(Icons.medication),
            label: tr("meds_title", lang)),
        NavigationDestination(
            icon: const Icon(Icons.calendar_today_outlined),
            selectedIcon: const Icon(Icons.calendar_today),
            label: tr("plans_title", lang)),
        NavigationDestination(
            icon: const Icon(Icons.warning_amber_outlined),
            selectedIcon: const Icon(Icons.warning_amber),
            label: tr("alerts_title", lang)),
        NavigationDestination(
            icon: const Icon(Icons.chat_bubble_outline),
            selectedIcon: const Icon(Icons.chat_bubble),
            label: tr("messages_title", lang)),
      ],
    );
  }
}

/// Public so other patient screens reuse it.
class PatientNav extends ConsumerWidget {
  const PatientNav({super.key, required this.currentIndex});
  final int currentIndex;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    return _PatientNav(currentIndex: currentIndex);
  }
}
