// Per-metric display metadata: emoji, unit, value formatter, normal range
// hint. Mirrors `frontend/lib/metric-meta.ts` so the UI stays consistent
// with what the backend reasons about.
import "package:flutter/material.dart";

class MetricMeta {
  const MetricMeta({
    required this.kind,
    required this.emoji,
    required this.unit,
    required this.color,
    required this.fmt,
    required this.suggestedRange,
  });
  final String kind;
  final String emoji;
  final String unit;
  final Color color;
  final String Function(double) fmt;
  final (double, double) suggestedRange;
}

const _teal = Color(0xFF0F766E);
const _amber = Color(0xFFD97706);
const _rose = Color(0xFFE11D48);
const _indigo = Color(0xFF4F46E5);

final Map<String, MetricMeta> metricMeta = {
  "pulse": MetricMeta(
    kind: "pulse",
    emoji: "❤️",
    unit: "уд/мин",
    color: _rose,
    fmt: (v) => v.round().toString(),
    suggestedRange: (40, 140),
  ),
  "bp_sys": MetricMeta(
    kind: "bp_sys",
    emoji: "🩸",
    unit: "мм рт.ст.",
    color: _rose,
    fmt: (v) => v.round().toString(),
    suggestedRange: (80, 200),
  ),
  "bp_dia": MetricMeta(
    kind: "bp_dia",
    emoji: "🩸",
    unit: "мм рт.ст.",
    color: _rose,
    fmt: (v) => v.round().toString(),
    suggestedRange: (50, 120),
  ),
  "glucose": MetricMeta(
    kind: "glucose",
    emoji: "🩸",
    unit: "ммоль/л",
    color: _amber,
    fmt: (v) => v.toStringAsFixed(1),
    suggestedRange: (3, 18),
  ),
  "temperature": MetricMeta(
    kind: "temperature",
    emoji: "🌡",
    unit: "°C",
    color: _indigo,
    fmt: (v) => v.toStringAsFixed(1),
    suggestedRange: (35, 41),
  ),
  "weight": MetricMeta(
    kind: "weight",
    emoji: "⚖️",
    unit: "кг",
    color: _teal,
    fmt: (v) => v.toStringAsFixed(1),
    suggestedRange: (30, 200),
  ),
  "spo2": MetricMeta(
    kind: "spo2",
    emoji: "💨",
    unit: "%",
    color: _teal,
    fmt: (v) => v.round().toString(),
    suggestedRange: (80, 100),
  ),
};
