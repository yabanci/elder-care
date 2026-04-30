"""Algorithm comparators.

Each comparator is a thin wrapper around AlgoRunner that fixes the
estimator + profile policy. The harness sweeps over all six and produces
metrics per-algorithm so the ablation table writes itself.
"""
from __future__ import annotations

from dataclasses import dataclass
from typing import Callable

from .runner import AlgoInput, AlgoResult, AlgoRunner, Profile


@dataclass(frozen=True)
class Algorithm:
    """One algorithm comparator. `id` is stable across runs (used in result CSVs)."""

    id: str
    estimator: str  # "" → server default
    use_condition: bool  # if False, profile is forced to all-False
    description: str

    def run(self, runner: AlgoRunner, inp: AlgoInput) -> AlgoResult:
        # Mutating the input would surprise callers; copy the relevant bits.
        effective_profile = inp.profile if self.use_condition else Profile()
        adjusted = AlgoInput(
            kind=inp.kind,
            value=inp.value,
            history=inp.history,
            profile=effective_profile,
            estimator=self.estimator,
            now=inp.now,
        )
        return runner.call(adjusted)


# v1-faithful wrapper: ignores the streak gate and condition profile by
# emulating "raw mean+SD over history" directly. Lives outside the Go
# binary because v1's old behaviour (no streak gate, no condition profile,
# population variance, no Hampel) is not selectable through baseline.Input.
def _v1_python(inp: AlgoInput) -> AlgoResult:
    """Reproduces the v1 algorithm in Python for the static/legacy comparison.

    Used by the `static_v1` and `mean_std_v1` algorithms below. We keep
    the Go production binary clean of v1 logic; this function is the
    archaeological reference implementation.
    """
    # Hardcoded v1 SafetyLimits — same constants the v1 backend used.
    safety: dict[str, dict[str, float]] = {
        "pulse": {"clo": 40, "chi": 140, "wlo": 50, "whi": 110},
        "bp_sys": {"clo": 80, "chi": 180, "wlo": 100, "whi": 150},
        "bp_dia": {"clo": 50, "chi": 110, "wlo": 60, "whi": 95},
        "glucose": {"clo": 3.0, "chi": 15.0, "wlo": 4.0, "whi": 10.0},
        "temperature": {"clo": 35.0, "chi": 39.0, "wlo": 35.5, "whi": 37.8},
        "spo2": {"clo": 88, "wlo": 93},
        "weight": {},
    }
    th = safety.get(inp.kind, {})
    severity = "normal"
    reason = "normal"
    if "clo" in th and inp.value < th["clo"]:
        return AlgoResult("critical", "safety_below_min", 0, 0, 0, "v1_static", False, 0, "v1")
    if "chi" in th and inp.value > th["chi"]:
        return AlgoResult("critical", "safety_above_max", 0, 0, 0, "v1_static", False, 0, "v1")
    if "wlo" in th and inp.value < th["wlo"]:
        severity = "warning"
        reason = "safety_warn_low"
    if "whi" in th and inp.value > th["whi"]:
        severity = "warning"
        reason = "safety_warn_high"
    return AlgoResult(severity, reason, 0, 0, 0, "v1_static", False, len(inp.history), "v1")


def _v1_meanstd_python(inp: AlgoInput) -> AlgoResult:
    """Reproduces v1's mean+SD baseline: last 30 values, no streak gate, pop variance."""
    safety = _v1_python(inp)
    if safety.severity == "critical":
        return safety
    history = sorted(inp.history, key=lambda r: r.measured_at)[-30:]
    if len(history) < 5:
        return safety
    values = [r.value for r in history]
    mean = sum(values) / len(values)
    var = sum((v - mean) ** 2 for v in values) / len(values)  # population
    std = var ** 0.5
    if std < 1e-9:
        return safety
    z = abs(inp.value - mean) / std
    severity = safety.severity
    reason = safety.reason_code
    if z >= 3:
        severity = "critical"
        reason = "baseline_crit_z3"
    elif z >= 2 and severity != "critical":
        severity = "warning"
        reason = "baseline_warn_z2"
    return AlgoResult(severity, reason, mean, std, z, "v1_mean_std", True, len(history), "v1")


# Algorithms exposed via subprocess (Go production code).
ALGORITHMS_GO: list[Algorithm] = [
    Algorithm(
        id="median_mad",
        estimator="median_mad",
        use_condition=False,
        description="Robust estimator (median + 1.4826·MAD), no condition profile",
    ),
    Algorithm(
        id="ewma",
        estimator="ewma",
        use_condition=False,
        description="Time-decayed EWMA mean + sample variance, no condition profile",
    ),
    Algorithm(
        id="ewma_mad",
        estimator="ewma_mad",
        use_condition=False,
        description="Production v2 estimator without condition profile (Claim A main)",
    ),
    Algorithm(
        id="ewma_mad_condition",
        estimator="ewma_mad",
        use_condition=True,
        description="Production v2 + condition-aware thresholds (Claim C main)",
    ),
]


# Python-side comparators for v1 baselines.
PythonAlgorithm = Callable[[AlgoInput], AlgoResult]

ALGORITHMS_PY: dict[str, PythonAlgorithm] = {
    "static_v1": _v1_python,
    "mean_std_v1": _v1_meanstd_python,
}

ALGORITHM_DESCRIPTIONS: dict[str, str] = {
    "static_v1": "v1 static safety thresholds only (Claim A baseline)",
    "mean_std_v1": "v1 mean + population SD over last 30 values (Claim A baseline)",
    **{a.id: a.description for a in ALGORITHMS_GO},
}


def all_algorithm_ids() -> list[str]:
    """Stable ordering used in result CSVs and plots."""
    return ["static_v1", "mean_std_v1"] + [a.id for a in ALGORITHMS_GO]
