import "package:flutter/material.dart";
import "package:flutter_riverpod/flutter_riverpod.dart";

import "../api/api_client.dart";
import "../l10n/strings.dart";
import "../models/models.dart";
import "../state/providers.dart";
import "patient_home_screen.dart";

class PatientPlansScreen extends ConsumerStatefulWidget {
  const PatientPlansScreen({super.key});
  @override
  ConsumerState<PatientPlansScreen> createState() =>
      _PatientPlansScreenState();
}

class _PatientPlansScreenState extends ConsumerState<PatientPlansScreen> {
  bool _loading = true;
  List<Plan> _plans = [];

  @override
  void initState() {
    super.initState();
    _refresh();
  }

  Future<void> _refresh() async {
    setState(() => _loading = true);
    try {
      final raw = await ref.read(apiClientProvider).get("/api/plans");
      setState(() {
        _plans = (raw as List).map((e) => Plan.fromJson(e)).toList();
        _loading = false;
      });
    } on ApiException {
      setState(() => _loading = false);
    }
  }

  Future<void> _create() async {
    final result = await showModalBottomSheet<Map<String, dynamic>>(
      context: context,
      isScrollControlled: true,
      builder: (ctx) => _AddPlanSheet(lang: ref.read(langProvider)),
    );
    if (result == null) return;
    try {
      await ref.read(apiClientProvider).post("/api/plans", data: result);
      await _refresh();
    } catch (_) {}
  }

  Future<void> _delete(String id) async {
    try {
      await ref.read(apiClientProvider).delete("/api/plans/$id");
      await _refresh();
    } catch (_) {}
  }

  @override
  Widget build(BuildContext context) {
    final lang = ref.watch(langProvider);
    final byDay = <int, List<Plan>>{};
    for (final p in _plans) {
      byDay.putIfAbsent(p.dayOfWeek, () => []).add(p);
    }
    final dayNames = lang == "kk"
        ? const ["Жс", "Дс", "Сс", "Ср", "Бс", "Жм", "Сб"]
        : lang == "en"
            ? const ["Sun", "Mon", "Tue", "Wed", "Thu", "Fri", "Sat"]
            : const ["Вс", "Пн", "Вт", "Ср", "Чт", "Пт", "Сб"];

    return Scaffold(
      appBar: AppBar(title: Text(tr("plans_title", lang))),
      floatingActionButton: FloatingActionButton.extended(
        onPressed: _create,
        icon: const Icon(Icons.add),
        label: Text(tr("add", lang)),
      ),
      body: RefreshIndicator(
        onRefresh: _refresh,
        child: _loading
            ? const Center(child: CircularProgressIndicator())
            : ListView.builder(
                padding: const EdgeInsets.all(16),
                itemCount: 7,
                itemBuilder: (ctx, i) {
                  final plans = byDay[i] ?? [];
                  return Padding(
                    padding: const EdgeInsets.only(bottom: 12),
                    child: Column(
                      crossAxisAlignment: CrossAxisAlignment.start,
                      children: [
                        Text(dayNames[i],
                            style: const TextStyle(
                                fontWeight: FontWeight.bold, fontSize: 18)),
                        const SizedBox(height: 4),
                        if (plans.isEmpty)
                          Text(tr("plan_empty", lang),
                              style: Theme.of(context).textTheme.bodySmall)
                        else
                          ...plans.map((p) => Card(
                                child: ListTile(
                                  leading: const Icon(Icons.event_note),
                                  title: Text(p.title),
                                  subtitle: Text(
                                    "${p.timeOfDay ?? ""}  ·  ${tr("plan_type_${p.planType}", lang)}",
                                  ),
                                  trailing: IconButton(
                                    icon: const Icon(Icons.close),
                                    onPressed: () => _delete(p.id),
                                  ),
                                ),
                              )),
                      ],
                    ),
                  );
                },
              ),
      ),
      bottomNavigationBar: const PatientNav(currentIndex: 2),
    );
  }
}

class _AddPlanSheet extends StatefulWidget {
  const _AddPlanSheet({required this.lang});
  final String lang;
  @override
  State<_AddPlanSheet> createState() => _AddPlanSheetState();
}

class _AddPlanSheetState extends State<_AddPlanSheet> {
  final _title = TextEditingController();
  final _time = TextEditingController(text: "09:00");
  int _day = DateTime.now().weekday % 7;
  String _type = "other";

  @override
  Widget build(BuildContext context) {
    final lang = widget.lang;
    final dayLabels = lang == "kk"
        ? const ["Жс", "Дс", "Сс", "Ср", "Бс", "Жм", "Сб"]
        : lang == "en"
            ? const ["Sun", "Mon", "Tue", "Wed", "Thu", "Fri", "Sat"]
            : const ["Вс", "Пн", "Вт", "Ср", "Чт", "Пт", "Сб"];
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
            controller: _title,
            autofocus: true,
            decoration: InputDecoration(labelText: tr("plan_name", lang)),
          ),
          const SizedBox(height: 12),
          DropdownButtonFormField<int>(
            initialValue: _day,
            decoration: InputDecoration(labelText: tr("plan_day", lang)),
            items: List.generate(7, (i) {
              return DropdownMenuItem(value: i, child: Text(dayLabels[i]));
            }),
            onChanged: (v) => setState(() => _day = v!),
          ),
          const SizedBox(height: 12),
          TextField(
            controller: _time,
            decoration: InputDecoration(labelText: tr("plan_time", lang)),
          ),
          const SizedBox(height: 12),
          DropdownButtonFormField<String>(
            initialValue: _type,
            decoration: InputDecoration(labelText: tr("plan_type", lang)),
            items: const ["doctor_visit", "take_med", "rest", "other"]
                .map((t) => DropdownMenuItem(
                    value: t, child: Text(tr("plan_type_$t", lang))))
                .toList(),
            onChanged: (v) => setState(() => _type = v!),
          ),
          const SizedBox(height: 16),
          ElevatedButton(
            onPressed: () {
              if (_title.text.trim().isEmpty) return;
              Navigator.pop(context, {
                "title": _title.text.trim(),
                "day_of_week": _day,
                "plan_type": _type,
                if (_time.text.trim().isNotEmpty)
                  "time_of_day": _time.text.trim(),
              });
            },
            child: Text(tr("save", lang)),
          ),
        ],
      ),
    );
  }
}
