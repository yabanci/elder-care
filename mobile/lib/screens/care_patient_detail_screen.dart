import "package:flutter/material.dart";
import "package:flutter_riverpod/flutter_riverpod.dart";
import "package:go_router/go_router.dart";
import "package:intl/intl.dart";

import "../api/api_client.dart";
import "../l10n/strings.dart";
import "../models/models.dart";
import "../state/providers.dart";
import "../widgets/metric_meta.dart";

class CarePatientDetailScreen extends ConsumerStatefulWidget {
  const CarePatientDetailScreen({super.key, required this.patientId});
  final String patientId;
  @override
  ConsumerState<CarePatientDetailScreen> createState() =>
      _CarePatientDetailScreenState();
}

class _CarePatientDetailScreenState
    extends ConsumerState<CarePatientDetailScreen>
    with SingleTickerProviderStateMixin {
  late TabController _tab;
  bool _loading = true;
  String? _patientName;
  List<Metric> _summary = [];
  List<Alert> _alerts = [];
  List<Medication> _meds = [];
  List<CareNote> _notes = [];

  @override
  void initState() {
    super.initState();
    _tab = TabController(length: 4, vsync: this);
    _refresh();
  }

  @override
  void dispose() {
    _tab.dispose();
    super.dispose();
  }

  Future<void> _refresh() async {
    final api = ref.read(apiClientProvider);
    setState(() => _loading = true);
    try {
      final patientsRaw = await api.get("/api/patients");
      for (final p in (patientsRaw as List)) {
        if (p["patient_id"] == widget.patientId) {
          _patientName = p["full_name"] as String;
        }
      }
      final s = await api.get("/api/patients/${widget.patientId}/metrics/summary");
      final a = await api.get("/api/patients/${widget.patientId}/alerts");
      final m = await api.get("/api/patients/${widget.patientId}/medications");
      final n = await api.get("/api/patients/${widget.patientId}/notes");
      setState(() {
        _summary = (s as List).map((e) => Metric.fromJson(e)).toList();
        _alerts = (a as List).map((e) => Alert.fromJson(e)).toList();
        _meds = (m as List).map((e) => Medication.fromJson(e)).toList();
        _notes = (n as List).map((e) => CareNote.fromJson(e)).toList();
        _loading = false;
      });
    } on ApiException {
      setState(() => _loading = false);
    }
  }

  Future<void> _addNote() async {
    final lang = ref.read(langProvider);
    final controller = TextEditingController();
    final body = await showModalBottomSheet<String>(
      context: context,
      isScrollControlled: true,
      builder: (ctx) => Padding(
        padding: EdgeInsets.fromLTRB(
            16, 16, 16, MediaQuery.of(ctx).viewInsets.bottom + 16),
        child: Column(mainAxisSize: MainAxisSize.min, children: [
          Text(tr("care_add_note", lang),
              style: Theme.of(ctx).textTheme.titleLarge),
          const SizedBox(height: 12),
          TextField(
              controller: controller,
              autofocus: true,
              maxLines: 4,
              decoration: InputDecoration(
                  labelText: tr("care_note_body", lang),
                  border: const OutlineInputBorder())),
          const SizedBox(height: 16),
          ElevatedButton(
            onPressed: () => Navigator.pop(ctx, controller.text.trim()),
            child: Text(tr("save", lang)),
          ),
        ]),
      ),
    );
    if (body == null || body.isEmpty) return;
    try {
      await ref.read(apiClientProvider).post(
        "/api/patients/${widget.patientId}/notes",
        data: {"body": body},
      );
      await _refresh();
    } catch (_) {}
  }

  Future<void> _prescribe() async {
    final lang = ref.read(langProvider);
    final name = TextEditingController();
    final dosage = TextEditingController();
    final times = TextEditingController(text: "08:00");
    final result = await showModalBottomSheet<Map<String, dynamic>>(
      context: context,
      isScrollControlled: true,
      builder: (ctx) => Padding(
        padding: EdgeInsets.fromLTRB(
            16, 16, 16, MediaQuery.of(ctx).viewInsets.bottom + 16),
        child: Column(
            mainAxisSize: MainAxisSize.min,
            crossAxisAlignment: CrossAxisAlignment.stretch,
            children: [
              Text(tr("care_prescribe", lang),
                  style: Theme.of(ctx).textTheme.titleLarge),
              const SizedBox(height: 12),
              TextField(
                  controller: name,
                  autofocus: true,
                  decoration:
                      InputDecoration(labelText: tr("meds_name", lang))),
              const SizedBox(height: 12),
              TextField(
                  controller: dosage,
                  decoration:
                      InputDecoration(labelText: tr("meds_dosage", lang))),
              const SizedBox(height: 12),
              TextField(
                  controller: times,
                  decoration:
                      InputDecoration(labelText: tr("meds_times", lang))),
              const SizedBox(height: 16),
              ElevatedButton(
                onPressed: () {
                  if (name.text.trim().isEmpty) return;
                  Navigator.pop(ctx, {
                    "name": name.text.trim(),
                    if (dosage.text.trim().isNotEmpty)
                      "dosage": dosage.text.trim(),
                    "times_of_day": times.text
                        .split(",")
                        .map((s) => s.trim())
                        .where((s) => s.isNotEmpty)
                        .toList(),
                  });
                },
                child: Text(tr("save", lang)),
              ),
            ]),
      ),
    );
    if (result == null) return;
    try {
      await ref.read(apiClientProvider).post(
            "/api/patients/${widget.patientId}/medications",
            data: result,
          );
      await _refresh();
    } catch (_) {}
  }

  @override
  Widget build(BuildContext context) {
    final lang = ref.watch(langProvider);
    return Scaffold(
      appBar: AppBar(
        title: Text(_patientName ?? "..."),
        leading: BackButton(onPressed: () => context.pop()),
        bottom: TabBar(
          controller: _tab,
          isScrollable: true,
          tabs: [
            Tab(text: tr("care_patient_metrics", lang)),
            Tab(text: tr("care_patient_alerts", lang)),
            Tab(text: tr("care_patient_meds", lang)),
            Tab(text: tr("care_patient_notes", lang)),
          ],
        ),
      ),
      body: _loading
          ? const Center(child: CircularProgressIndicator())
          : TabBarView(
              controller: _tab,
              children: [
                _metricsTab(lang),
                _alertsTab(lang),
                _medsTab(lang),
                _notesTab(lang),
              ],
            ),
    );
  }

  Widget _metricsTab(String lang) {
    if (_summary.isEmpty) {
      return Center(child: Text(tr("no_data", lang)));
    }
    return RefreshIndicator(
      onRefresh: _refresh,
      child: ListView(
        padding: const EdgeInsets.all(16),
        children: _summary.map((m) {
          final meta = metricMeta[m.kind];
          return Card(
            child: ListTile(
              leading: Text(meta?.emoji ?? "—",
                  style: const TextStyle(fontSize: 28)),
              title: Text(meta != null
                  ? tr("metric_kind_${m.kind}", lang)
                  : m.kind),
              subtitle: Text(DateFormat("d MMM, HH:mm").format(m.measuredAt)),
              trailing: Text(
                  meta != null ? "${meta.fmt(m.value)} ${meta.unit}" : "${m.value}",
                  style: const TextStyle(
                      fontSize: 18, fontWeight: FontWeight.bold)),
            ),
          );
        }).toList(),
      ),
    );
  }

  Widget _alertsTab(String lang) {
    if (_alerts.isEmpty) return Center(child: Text(tr("alerts_empty", lang)));
    return RefreshIndicator(
      onRefresh: _refresh,
      child: ListView.builder(
        padding: const EdgeInsets.all(16),
        itemCount: _alerts.length,
        itemBuilder: (ctx, i) {
          final a = _alerts[i];
          final isCritical = a.severity == "critical";
          return Card(
            color: isCritical
                ? const Color(0xFFFEF2F2)
                : const Color(0xFFFFFBEB),
            child: ListTile(
              title: Text(_localizedReason(a, lang)),
              subtitle: Text(DateFormat("d MMM, HH:mm").format(a.createdAt)),
              trailing: a.acknowledged
                  ? const Icon(Icons.check, color: Colors.green)
                  : null,
            ),
          );
        },
      ),
    );
  }

  Widget _medsTab(String lang) {
    return Stack(children: [
      RefreshIndicator(
        onRefresh: _refresh,
        child: _meds.isEmpty
            ? ListView(children: [
                const SizedBox(height: 80),
                Center(child: Text(tr("meds_empty", lang))),
              ])
            : ListView.builder(
                padding: const EdgeInsets.fromLTRB(16, 16, 16, 96),
                itemCount: _meds.length,
                itemBuilder: (ctx, i) {
                  final m = _meds[i];
                  return Card(
                    child: ListTile(
                      leading: const Icon(Icons.medication, size: 30),
                      title: Text(m.name),
                      subtitle: Text(
                          "${m.dosage ?? ""}${m.timesOfDay.isNotEmpty ? "  ⏰ ${m.timesOfDay.join(", ")}" : ""}"),
                      trailing: m.prescribedBy != null
                          ? const Icon(Icons.medical_services,
                              color: Colors.teal)
                          : null,
                    ),
                  );
                }),
      ),
      Positioned(
        right: 16,
        bottom: 16,
        child: FloatingActionButton.extended(
          onPressed: _prescribe,
          icon: const Icon(Icons.add),
          label: Text(tr("care_prescribe", lang)),
        ),
      ),
    ]);
  }

  Widget _notesTab(String lang) {
    return Stack(children: [
      RefreshIndicator(
        onRefresh: _refresh,
        child: _notes.isEmpty
            ? ListView(children: [
                const SizedBox(height: 80),
                Center(child: Text(tr("no_data", lang))),
              ])
            : ListView.builder(
                padding: const EdgeInsets.fromLTRB(16, 16, 16, 96),
                itemCount: _notes.length,
                itemBuilder: (ctx, i) {
                  final n = _notes[i];
                  return Card(
                    child: Padding(
                      padding: const EdgeInsets.all(12),
                      child: Column(
                        crossAxisAlignment: CrossAxisAlignment.start,
                        children: [
                          Row(children: [
                            Text(n.authorName,
                                style: const TextStyle(
                                    fontWeight: FontWeight.bold)),
                            const Spacer(),
                            Text(DateFormat("d MMM, HH:mm").format(n.createdAt),
                                style:
                                    Theme.of(context).textTheme.bodySmall),
                          ]),
                          const SizedBox(height: 4),
                          Text(n.body),
                        ],
                      ),
                    ),
                  );
                }),
      ),
      Positioned(
        right: 16,
        bottom: 16,
        child: FloatingActionButton.extended(
          onPressed: _addNote,
          icon: const Icon(Icons.note_add),
          label: Text(tr("care_add_note", lang)),
        ),
      ),
    ]);
  }
}

String _localizedReason(Alert a, String lang) {
  if (a.reasonCode == "legacy" || a.reasonCode.isEmpty) {
    return a.reason.isNotEmpty ? a.reason : tr("reason_legacy", lang);
  }
  final key = "reason_${a.reasonCode}";
  final v = tr(key, lang);
  return v == key
      ? (a.reason.isNotEmpty ? a.reason : a.reasonCode)
      : v;
}
