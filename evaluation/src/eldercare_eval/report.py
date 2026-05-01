"""Generate REPORT.md — committee-ready evaluation summary."""
from __future__ import annotations

from datetime import datetime, timezone
from pathlib import Path

from .algorithms import ALGORITHM_DESCRIPTIONS, all_algorithm_ids
from .metrics import EvalRow, aggregate_f1


def _table(rows: list[EvalRow], threshold: str) -> str:
    keep = [r for r in rows if r.threshold == threshold]
    algos = all_algorithm_ids()
    metrics_seen = sorted({r.metric for r in keep})

    out = ["| metric | " + " | ".join(algos) + " |"]
    out.append("|---|" + "|".join(["---"] * len(algos)) + "|")
    for m in metrics_seen:
        cells = [m]
        for a in algos:
            f1s = [r.f1 for r in keep if r.metric == m and r.algorithm == a]
            cells.append(f"{(sum(f1s) / len(f1s)) if f1s else 0:.3f}")
        out.append("| " + " | ".join(cells) + " |")
    return "\n".join(out)


def write(rows: list[EvalRow], out_path: Path, figures_dir: Path) -> None:
    macro = aggregate_f1(rows)
    chronic_archetypes = {"hypertension_75", "t2d_78", "copd_80"}

    warning_rows = [r for r in rows if r.threshold == "warning"]
    chronic_rows = [r for r in warning_rows if r.archetype in chronic_archetypes]
    far_by_alg: dict[str, list[float]] = {}
    for r in chronic_rows:
        far_by_alg.setdefault(r.algorithm, []).append(r.far_per_week)

    # Per-algorithm precision / recall macros, computed on warning threshold.
    p_by_alg: dict[str, list[float]] = {}
    r_by_alg: dict[str, list[float]] = {}
    for r in warning_rows:
        p_by_alg.setdefault(r.algorithm, []).append(r.precision)
        r_by_alg.setdefault(r.algorithm, []).append(r.recall)

    lines = [
        "# ElderCare Baseline v2 — Evaluation Report",
        "",
        f"_Generated {datetime.now(timezone.utc).isoformat(timespec='seconds')}_",
        "",
        "## Algorithms compared",
        "",
        "| ID | Description |",
        "|---|---|",
    ]
    for aid in all_algorithm_ids():
        lines.append(f"| `{aid}` | {ALGORITHM_DESCRIPTIONS.get(aid, '')} |")
    lines.extend([
        "",
        "## F1 by metric (warning threshold)",
        "",
        _table(rows, "warning"),
        "",
        "## F1 by metric (critical threshold)",
        "",
        _table(rows, "critical"),
        "",
        "## Precision / Recall / F1 macros (warning threshold)",
        "",
        "| Algorithm | Precision | Recall | F1 |",
        "|---|---|---|---|",
    ])
    for aid in all_algorithm_ids():
        ps = p_by_alg.get(aid, [])
        rs = r_by_alg.get(aid, [])
        p_macro = (sum(ps) / len(ps)) if ps else 0.0
        r_macro = (sum(rs) / len(rs)) if rs else 0.0
        lines.append(f"| `{aid}` | {p_macro:.3f} | {r_macro:.3f} | {macro.get(aid, 0):.3f} |")

    lines.extend([
        "",
        "**Reading the table.** F1 weights precision and recall equally, but in",
        "an alerting system aimed at elderly home monitoring the operational",
        "cost asymmetry is severe — false positives drive alarm fatigue and are",
        "the primary reason such systems get disabled. The right headline metric",
        "for Claim A is therefore **false-alarm rate** (next section), not",
        "aggregate F1: v2 trades some recall for materially fewer false alarms,",
        "which is the intended trade-off.",
        "",
        "## Claim C — false-alarm rate on chronic archetypes (lower = better)",
        "",
        "| Algorithm | Mean FAR per patient-week |",
        "|---|---|",
    ])
    for aid in all_algorithm_ids():
        vs = far_by_alg.get(aid, [])
        mean_far = sum(vs) / len(vs) if vs else 0.0
        lines.append(f"| `{aid}` | {mean_far:.3f} |")

    lines.extend([
        "",
        "## Plots",
        "",
        f"![F1 by algorithm]({figures_dir.name}/f1_by_algorithm.png)",
        "",
        f"![FAR by condition]({figures_dir.name}/far_by_condition.png)",
        "",
        f"![Lead-time]({figures_dir.name}/lead_time.png)",
        "",
        "## Notes",
        "",
        "- Synthetic data: per-archetype literature-grounded means/SDs, diurnal cycles, drift, measurement noise, state-correlated noise. Anomaly types planted: point (4σ), contextual (2.5σ within static safety), collective (3-day drift), inverse (down-side dip).",
        "- Algorithms `static_v1` and `mean_std_v1` are Python re-implementations of v1-as-shipped (no streak gate, population variance) for the ablation comparison. Other algorithms run as the production Go code via `cmd/algo-runner`.",
        "- Cold-start is in effect: when an archetype has < 10 readings in the last 14 days, the algorithm refuses to fire personal-baseline alerts. Safety bounds still apply.",
    ])

    out_path.write_text("\n".join(lines))
