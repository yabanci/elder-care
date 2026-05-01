import "package:flutter/material.dart";
import "package:flutter_riverpod/flutter_riverpod.dart";

import "../api/api_client.dart";
import "../l10n/strings.dart";
import "../models/models.dart";
import "../state/providers.dart";
import "patient_home_screen.dart";

class PatientMedsScreen extends ConsumerStatefulWidget {
  const PatientMedsScreen({super.key});
  @override
  ConsumerState<PatientMedsScreen> createState() => _PatientMedsScreenState();
}

class _PatientMedsScreenState extends ConsumerState<PatientMedsScreen> {
  bool _loading = true;
  List<Medication> _meds = [];

  @override
  void initState() {
    super.initState();
    _refresh();
  }

  Future<void> _refresh() async {
    setState(() => _loading = true);
    try {
      final raw = await ref.read(apiClientProvider).get("/api/medications");
      setState(() {
        _meds = (raw as List).map((e) => Medication.fromJson(e)).toList();
        _loading = false;
      });
    } on ApiException {
      setState(() => _loading = false);
    }
  }

  Future<void> _create() async {
    final lang = ref.read(langProvider);
    final result = await showModalBottomSheet<Map<String, dynamic>>(
      context: context,
      isScrollControlled: true,
      builder: (ctx) => _AddMedSheet(lang: lang),
    );
    if (result == null) return;
    try {
      await ref.read(apiClientProvider).post("/api/medications", data: result);
      await _refresh();
    } catch (_) {}
  }

  Future<void> _delete(String id) async {
    try {
      await ref.read(apiClientProvider).delete("/api/medications/$id");
      await _refresh();
    } catch (_) {}
  }

  @override
  Widget build(BuildContext context) {
    final lang = ref.watch(langProvider);
    return Scaffold(
      appBar: AppBar(title: Text(tr("meds_title", lang))),
      floatingActionButton: FloatingActionButton.extended(
        onPressed: _create,
        icon: const Icon(Icons.add),
        label: Text(tr("add", lang)),
      ),
      body: RefreshIndicator(
        onRefresh: _refresh,
        child: _loading
            ? const Center(child: CircularProgressIndicator())
            : _meds.isEmpty
                ? ListView(children: [
                    const SizedBox(height: 80),
                    Center(child: Text(tr("meds_empty", lang))),
                  ])
                : ListView.builder(
                    padding: const EdgeInsets.all(16),
                    itemCount: _meds.length,
                    itemBuilder: (ctx, i) {
                      final m = _meds[i];
                      return Card(
                        child: ListTile(
                          leading: const Icon(Icons.medication, size: 32),
                          title: Text(m.name,
                              style: const TextStyle(
                                  fontWeight: FontWeight.bold)),
                          subtitle: Column(
                            crossAxisAlignment: CrossAxisAlignment.start,
                            children: [
                              if (m.dosage != null) Text(m.dosage!),
                              if (m.timesOfDay.isNotEmpty)
                                Text("⏰ ${m.timesOfDay.join(", ")}"),
                              if (m.prescribedBy != null)
                                Padding(
                                  padding: const EdgeInsets.only(top: 4),
                                  child: Text(
                                    "🩺 ${tr("meds_prescribed_by", lang)}",
                                    style: const TextStyle(
                                        color: Colors.teal,
                                        fontSize: 12),
                                  ),
                                ),
                            ],
                          ),
                          trailing: IconButton(
                            icon: const Icon(Icons.delete_outline),
                            onPressed: () => _delete(m.id),
                          ),
                        ),
                      );
                    }),
      ),
      bottomNavigationBar: const PatientNav(currentIndex: 1),
    );
  }
}

class _AddMedSheet extends StatefulWidget {
  const _AddMedSheet({required this.lang});
  final String lang;
  @override
  State<_AddMedSheet> createState() => _AddMedSheetState();
}

class _AddMedSheetState extends State<_AddMedSheet> {
  final _name = TextEditingController();
  final _dosage = TextEditingController();
  final _times = TextEditingController(text: "08:00,20:00");

  @override
  Widget build(BuildContext context) {
    final lang = widget.lang;
    return Padding(
      padding: EdgeInsets.fromLTRB(
          16, 16, 16, MediaQuery.of(context).viewInsets.bottom + 16),
      child: Column(
        mainAxisSize: MainAxisSize.min,
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          Text(tr("add", lang),
              style: Theme.of(context).textTheme.titleLarge),
          const SizedBox(height: 12),
          TextField(
            controller: _name,
            autofocus: true,
            decoration: InputDecoration(labelText: tr("meds_name", lang)),
          ),
          const SizedBox(height: 12),
          TextField(
            controller: _dosage,
            decoration: InputDecoration(labelText: tr("meds_dosage", lang)),
          ),
          const SizedBox(height: 12),
          TextField(
            controller: _times,
            decoration: InputDecoration(labelText: tr("meds_times", lang)),
          ),
          const SizedBox(height: 16),
          ElevatedButton(
            onPressed: () {
              if (_name.text.trim().isEmpty) return;
              Navigator.pop(context, {
                "name": _name.text.trim(),
                if (_dosage.text.trim().isNotEmpty) "dosage": _dosage.text.trim(),
                "times_of_day": _times.text
                    .split(",")
                    .map((s) => s.trim())
                    .where((s) => s.isNotEmpty)
                    .toList(),
              });
            },
            child: Text(tr("save", lang)),
          ),
        ],
      ),
    );
  }
}
