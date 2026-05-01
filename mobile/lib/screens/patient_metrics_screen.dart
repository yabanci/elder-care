import "package:fl_chart/fl_chart.dart";
import "package:flutter/material.dart";
import "package:flutter_riverpod/flutter_riverpod.dart";
import "package:intl/intl.dart";

import "../api/api_client.dart";
import "../l10n/strings.dart";
import "../models/models.dart";
import "../state/providers.dart";
import "../widgets/metric_meta.dart";
import "patient_home_screen.dart";

class PatientMetricsScreen extends ConsumerStatefulWidget {
  const PatientMetricsScreen({super.key});
  @override
  ConsumerState<PatientMetricsScreen> createState() =>
      _PatientMetricsScreenState();
}

class _PatientMetricsScreenState extends ConsumerState<PatientMetricsScreen> {
  bool _loading = true;
  List<Metric> _metrics = [];

  @override
  void initState() {
    super.initState();
    _refresh();
  }

  Future<void> _refresh() async {
    setState(() => _loading = true);
    try {
      final raw = await ref.read(apiClientProvider).get("/api/metrics");
      setState(() {
        _metrics = (raw as List).map((e) => Metric.fromJson(e)).toList();
        _loading = false;
      });
    } on ApiException {
      setState(() => _loading = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    final lang = ref.watch(langProvider);
    final byKind = <String, List<Metric>>{};
    for (final m in _metrics) {
      byKind.putIfAbsent(m.kind, () => []).add(m);
    }
    return Scaffold(
      appBar: AppBar(title: Text(tr("metrics_title", lang))),
      body: RefreshIndicator(
        onRefresh: _refresh,
        child: _loading
            ? const Center(child: CircularProgressIndicator())
            : ListView(
                padding: const EdgeInsets.all(16),
                children: byKind.entries.map((entry) {
                  final meta = metricMeta[entry.key];
                  if (meta == null) return const SizedBox.shrink();
                  return _MetricChartCard(
                    metric: entry.key,
                    label: tr("metric_kind_${entry.key}", lang),
                    meta: meta,
                    samples: entry.value,
                    lang: lang,
                  );
                }).toList(),
              ),
      ),
      bottomNavigationBar: const PatientNav(currentIndex: 0),
    );
  }
}

class _MetricChartCard extends StatelessWidget {
  const _MetricChartCard({
    required this.metric,
    required this.label,
    required this.meta,
    required this.samples,
    required this.lang,
  });
  final String metric;
  final String label;
  final MetricMeta meta;
  final List<Metric> samples;
  final String lang;

  @override
  Widget build(BuildContext context) {
    final sorted = [...samples]
      ..sort((a, b) => a.measuredAt.compareTo(b.measuredAt));
    final spots = [
      for (var i = 0; i < sorted.length; i++)
        FlSpot(i.toDouble(), sorted[i].value)
    ];
    final mean = sorted.isEmpty
        ? 0.0
        : sorted.map((s) => s.value).reduce((a, b) => a + b) / sorted.length;

    return Card(
      child: Padding(
        padding: const EdgeInsets.all(12),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Row(children: [
              Text(meta.emoji, style: const TextStyle(fontSize: 22)),
              const SizedBox(width: 8),
              Text(label,
                  style: const TextStyle(
                      fontWeight: FontWeight.bold, fontSize: 16)),
              const Spacer(),
              Text(meta.unit,
                  style: Theme.of(context).textTheme.bodySmall),
            ]),
            const SizedBox(height: 8),
            SizedBox(
              height: 160,
              child: spots.length < 2
                  ? Center(
                      child: Text(tr("no_data", lang),
                          style: Theme.of(context).textTheme.bodySmall),
                    )
                  : LineChart(
                      LineChartData(
                        gridData: const FlGridData(show: true),
                        titlesData: const FlTitlesData(show: false),
                        borderData: FlBorderData(show: false),
                        extraLinesData: ExtraLinesData(horizontalLines: [
                          HorizontalLine(
                            y: mean,
                            color: Colors.grey,
                            dashArray: [4, 4],
                            strokeWidth: 1,
                            label: HorizontalLineLabel(
                              show: true,
                              alignment: Alignment.topRight,
                              labelResolver: (_) =>
                                  "${tr("alert_baseline", lang)} ${meta.fmt(mean)}",
                              style: const TextStyle(fontSize: 10),
                            ),
                          ),
                        ]),
                        lineBarsData: [
                          LineChartBarData(
                            spots: spots,
                            color: meta.color,
                            barWidth: 3,
                            isCurved: true,
                            dotData: const FlDotData(show: true),
                          ),
                        ],
                      ),
                    ),
            ),
            const SizedBox(height: 4),
            Text(
              sorted.isEmpty
                  ? ""
                  : "${DateFormat("d MMM").format(sorted.first.measuredAt)} → ${DateFormat("d MMM").format(sorted.last.measuredAt)}",
              style: Theme.of(context).textTheme.bodySmall,
            ),
          ],
        ),
      ),
    );
  }
}
