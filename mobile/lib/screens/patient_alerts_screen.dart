import "package:flutter/material.dart";
import "package:flutter_riverpod/flutter_riverpod.dart";
import "package:intl/intl.dart";

import "../api/api_client.dart";
import "../l10n/strings.dart";
import "../models/models.dart";
import "../state/providers.dart";
import "../widgets/metric_meta.dart";
import "patient_home_screen.dart";

class PatientAlertsScreen extends ConsumerStatefulWidget {
  const PatientAlertsScreen({super.key});
  @override
  ConsumerState<PatientAlertsScreen> createState() =>
      _PatientAlertsScreenState();
}

class _PatientAlertsScreenState extends ConsumerState<PatientAlertsScreen> {
  bool _loading = true;
  List<Alert> _alerts = [];

  @override
  void initState() {
    super.initState();
    _refresh();
  }

  Future<void> _refresh() async {
    setState(() => _loading = true);
    try {
      final raw = await ref.read(apiClientProvider).get("/api/alerts");
      setState(() {
        _alerts = (raw as List).map((e) => Alert.fromJson(e)).toList();
        _loading = false;
      });
    } on ApiException {
      setState(() => _loading = false);
    }
  }

  Future<void> _ack(String id) async {
    try {
      await ref
          .read(apiClientProvider)
          .post("/api/alerts/$id/acknowledge");
      await _refresh();
    } catch (_) {}
  }

  @override
  Widget build(BuildContext context) {
    final lang = ref.watch(langProvider);
    return Scaffold(
      appBar: AppBar(title: Text(tr("alerts_title", lang))),
      body: RefreshIndicator(
        onRefresh: _refresh,
        child: _loading
            ? const Center(child: CircularProgressIndicator())
            : _alerts.isEmpty
                ? ListView(children: [
                    const SizedBox(height: 80),
                    Center(child: Text(tr("alerts_empty", lang))),
                  ])
                : ListView.builder(
                    padding: const EdgeInsets.all(16),
                    itemCount: _alerts.length,
                    itemBuilder: (ctx, i) {
                      final a = _alerts[i];
                      final isCritical = a.severity == "critical";
                      final color = isCritical
                          ? const Color(0xFFFEF2F2)
                          : const Color(0xFFFFFBEB);
                      final meta = metricMeta[a.kind];
                      return Card(
                        color: color,
                        child: ListTile(
                          leading: Text(
                            meta?.emoji ?? "⚠️",
                            style: const TextStyle(fontSize: 28),
                          ),
                          title: Row(children: [
                            Expanded(
                                child: Text(meta != null
                                    ? tr("metric_kind_${a.kind}", lang)
                                    : a.kind)),
                            Container(
                              padding: const EdgeInsets.symmetric(
                                  horizontal: 8, vertical: 2),
                              decoration: BoxDecoration(
                                color: isCritical
                                    ? Colors.red
                                    : Colors.orange,
                                borderRadius: BorderRadius.circular(8),
                              ),
                              child: Text(
                                isCritical
                                    ? tr("severity_critical", lang)
                                    : tr("severity_warning", lang),
                                style: const TextStyle(
                                    color: Colors.white,
                                    fontSize: 12,
                                    fontWeight: FontWeight.bold),
                              ),
                            ),
                          ]),
                          subtitle: Column(
                            crossAxisAlignment: CrossAxisAlignment.start,
                            children: [
                              const SizedBox(height: 4),
                              Text(_localizedReason(a, lang)),
                              if (a.value != null) ...[
                                const SizedBox(height: 2),
                                Text(
                                  "${meta?.fmt(a.value!) ?? a.value}"
                                  "${meta != null ? " ${meta.unit}" : ""}"
                                  "${a.baselineMean != null ? "  ·  ${tr("alert_baseline", lang)} ≈ ${meta?.fmt(a.baselineMean!) ?? a.baselineMean}" : ""}",
                                  style: const TextStyle(
                                      color: Colors.black54),
                                ),
                              ],
                              const SizedBox(height: 2),
                              Text(
                                DateFormat("d MMM, HH:mm",
                                        _localeFor(lang))
                                    .format(a.createdAt),
                                style: Theme.of(context).textTheme.bodySmall,
                              ),
                            ],
                          ),
                          trailing: a.acknowledged
                              ? const Icon(Icons.check, color: Colors.green)
                              : TextButton(
                                  onPressed: () => _ack(a.id),
                                  child: Text(tr("alerts_ack", lang)),
                                ),
                        ),
                      );
                    }),
      ),
      bottomNavigationBar: const PatientNav(currentIndex: 3),
    );
  }
}

String _localizedReason(Alert a, String lang) {
  if (a.reasonCode == "legacy" || a.reasonCode.isEmpty) {
    return a.reason.isNotEmpty ? a.reason : tr("reason_legacy", lang);
  }
  final key = "reason_${a.reasonCode}";
  final localized = tr(key, lang);
  return localized == key
      ? (a.reason.isNotEmpty ? a.reason : a.reasonCode)
      : localized;
}

String _localeFor(String lang) =>
    lang == "kk" ? "kk_KZ" : (lang == "en" ? "en_US" : "ru_RU");
