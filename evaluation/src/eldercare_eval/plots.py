"""Defense plots — F1 bar charts, false-alarm rate by condition, lead-time."""
from __future__ import annotations

from pathlib import Path
from typing import Iterable

import matplotlib

matplotlib.use("Agg")  # headless safe; CI has no display
import matplotlib.pyplot as plt  # noqa: E402

from .metrics import EvalRow

# Stable algorithm ordering for plots / tables.
DEFAULT_ALGO_ORDER = [
    "static_v1",
    "mean_std_v1",
    "median_mad",
    "ewma",
    "ewma_mad",
    "ewma_mad_condition",
]


def _filter(rows: Iterable[EvalRow], threshold: str) -> list[EvalRow]:
    return [r for r in rows if r.threshold == threshold]


def f1_by_algorithm(rows: list[EvalRow], out: Path, threshold: str = "warning") -> None:
    """Macro-F1 across (archetype, metric) for each algorithm."""
    keep = _filter(rows, threshold)
    bucket: dict[str, list[float]] = {a: [] for a in DEFAULT_ALGO_ORDER}
    for r in keep:
        if r.algorithm in bucket:
            bucket[r.algorithm].append(r.f1)
    means = [sum(v) / len(v) if v else 0.0 for v in bucket.values()]

    fig, ax = plt.subplots(figsize=(9, 5))
    bars = ax.bar(list(bucket.keys()), means, color="#4C72B0")
    ax.set_ylim(0, 1)
    ax.set_ylabel("Macro-F1 (warning threshold)")
    ax.set_title(f"F1 by algorithm — threshold={threshold}")
    for b, v in zip(bars, means):
        ax.text(b.get_x() + b.get_width() / 2, v + 0.01, f"{v:.2f}", ha="center", fontsize=9)
    fig.autofmt_xdate(rotation=20)
    fig.tight_layout()
    fig.savefig(out, dpi=150)
    plt.close(fig)


def far_by_condition(rows: list[EvalRow], out: Path) -> None:
    """False-alarm rate per patient-week, grouped by archetype and algorithm.

    Validates Claim C: condition-aware variant should reduce FAR for chronic
    archetypes vs the un-condition-aware ewma_mad.
    """
    keep = _filter(rows, "warning")
    archetypes = sorted({r.archetype for r in keep})
    bucket: dict[str, dict[str, list[float]]] = {a: {alg: [] for alg in DEFAULT_ALGO_ORDER} for a in archetypes}
    for r in keep:
        if r.algorithm in DEFAULT_ALGO_ORDER:
            bucket[r.archetype][r.algorithm].append(r.far_per_week)

    width = 0.13
    x_positions = list(range(len(archetypes)))

    fig, ax = plt.subplots(figsize=(11, 5))
    for i, alg in enumerate(DEFAULT_ALGO_ORDER):
        ys = [
            sum(bucket[a][alg]) / len(bucket[a][alg]) if bucket[a][alg] else 0.0
            for a in archetypes
        ]
        offsets = [x + (i - len(DEFAULT_ALGO_ORDER) / 2) * width for x in x_positions]
        ax.bar(offsets, ys, width=width, label=alg)

    ax.set_xticks(x_positions)
    ax.set_xticklabels(archetypes, rotation=15)
    ax.set_ylabel("False alarms per simulated patient-week")
    ax.set_title("False-alarm rate by archetype and algorithm (lower is better)")
    ax.legend(fontsize=8, ncol=3)
    fig.tight_layout()
    fig.savefig(out, dpi=150)
    plt.close(fig)


def lead_time(rows: list[EvalRow], out: Path) -> None:
    """Mean detection lead-time on collective anomalies (in samples)."""
    keep = _filter(rows, "warning")
    bucket: dict[str, list[float]] = {a: [] for a in DEFAULT_ALGO_ORDER}
    for r in keep:
        if r.algorithm in bucket and r.mean_lead_samples == r.mean_lead_samples:
            bucket[r.algorithm].append(r.mean_lead_samples)
    means = [sum(v) / len(v) if v else 0.0 for v in bucket.values()]

    fig, ax = plt.subplots(figsize=(9, 5))
    bars = ax.bar(list(bucket.keys()), means, color="#55A868")
    ax.set_ylabel("Mean lead-samples to detection (lower is better)")
    ax.set_title("Detection lead-time on collective anomalies")
    for b, v in zip(bars, means):
        ax.text(b.get_x() + b.get_width() / 2, v + 0.05, f"{v:.1f}", ha="center", fontsize=9)
    fig.autofmt_xdate(rotation=20)
    fig.tight_layout()
    fig.savefig(out, dpi=150)
    plt.close(fig)
