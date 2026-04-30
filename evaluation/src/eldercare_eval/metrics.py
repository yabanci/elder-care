"""Evaluation metrics — precision / recall / F1 / AUROC / lead-time / FAR.

We treat each *sample* as a binary classification target: positive iff the
sample falls inside any planted anomaly window AND the sample's
oracle-defined severity is at least the threshold under evaluation
("warning" or "critical"). The algorithm's "prediction" is severity ≥
threshold for the same sample.

Lead-time is computed only on COLLECTIVE anomalies — for point/contextual
single-sample anomalies it isn't a meaningful concept.
"""
from __future__ import annotations

from dataclasses import dataclass
from typing import Iterable

from .runner import AlgoResult
from .simulator import AnomalyLabel, AnomalyType, Series

SEVERITY_RANK = {"normal": 0, "info": 1, "warning": 2, "critical": 3}
SECONDS_PER_DAY = 86_400


@dataclass(frozen=True)
class EvalRow:
    archetype: str
    metric: str
    algorithm: str
    threshold: str  # "warning" or "critical"
    n_samples: int
    n_positives: int
    tp: int
    fp: int
    fn: int
    tn: int
    precision: float
    recall: float
    f1: float
    far_per_week: float
    mean_lead_samples: float


def _label_truth(series: Series, threshold: str) -> list[bool]:
    th = SEVERITY_RANK[threshold]
    truth = [False] * len(series.values)
    for a in series.anomalies:
        if SEVERITY_RANK[a.severity_truth] < th:
            continue
        for i in range(a.start_idx, a.end_idx + 1):
            truth[i] = True
    return truth


def _predict(results: list[AlgoResult], threshold: str) -> list[bool]:
    th = SEVERITY_RANK[threshold]
    return [SEVERITY_RANK[r.severity] >= th for r in results]


def _far_per_week(false_positives: int, n_samples: int, samples_per_day: int) -> float:
    if n_samples == 0 or samples_per_day == 0:
        return 0.0
    n_weeks = n_samples / (samples_per_day * 7.0)
    if n_weeks == 0:
        return 0.0
    return false_positives / n_weeks


def evaluate(
    series: Series,
    results: list[AlgoResult],
    algorithm: str,
    threshold: str,
    samples_per_day: int,
) -> EvalRow:
    truth = _label_truth(series, threshold)
    pred = _predict(results, threshold)

    tp = sum(1 for t, p in zip(truth, pred) if t and p)
    fp = sum(1 for t, p in zip(truth, pred) if not t and p)
    fn = sum(1 for t, p in zip(truth, pred) if t and not p)
    tn = sum(1 for t, p in zip(truth, pred) if not t and not p)

    precision = tp / (tp + fp) if (tp + fp) > 0 else 0.0
    recall = tp / (tp + fn) if (tp + fn) > 0 else 0.0
    f1 = 2 * precision * recall / (precision + recall) if (precision + recall) > 0 else 0.0

    leads: list[int] = []
    for a in series.anomalies:
        if a.type != AnomalyType.COLLECTIVE:
            continue
        if SEVERITY_RANK[a.severity_truth] < SEVERITY_RANK[threshold]:
            continue
        for i in range(a.start_idx, a.end_idx + 1):
            if pred[i]:
                leads.append(i - a.start_idx)
                break

    mean_lead = sum(leads) / len(leads) if leads else float("nan")

    return EvalRow(
        archetype=series.archetype_id,
        metric=series.metric,
        algorithm=algorithm,
        threshold=threshold,
        n_samples=len(series.values),
        n_positives=sum(truth),
        tp=tp, fp=fp, fn=fn, tn=tn,
        precision=precision,
        recall=recall,
        f1=f1,
        far_per_week=_far_per_week(fp, len(series.values), samples_per_day),
        mean_lead_samples=mean_lead,
    )


def aggregate_f1(rows: Iterable[EvalRow]) -> dict[str, float]:
    """Macro-F1 by algorithm across (metric, archetype) combinations."""
    bucket: dict[str, list[float]] = {}
    for r in rows:
        bucket.setdefault(r.algorithm, []).append(r.f1)
    return {algo: sum(vs) / len(vs) if vs else 0.0 for algo, vs in bucket.items()}
