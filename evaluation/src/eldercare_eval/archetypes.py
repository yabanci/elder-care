"""Patient archetypes — population-grounded baselines for the simulator.

References (cited in the thesis):
  - AHA / ACC 2017 Hypertension Guidelines (BP norms by age, condition)
  - WHO Cardiovascular Disease Risk Charts (population SDs)
  - ATP / ADA Standards of Care (glucose targets, T2D)
  - GOLD Report (SpO2 ranges in COPD)
  - Resting heart rate by age, Mayo Clinic
"""
from __future__ import annotations

from dataclasses import dataclass

from .runner import Profile


@dataclass(frozen=True)
class MetricSpec:
    """Per-metric model parameters for one archetype."""

    mean: float            # baseline mean
    std: float             # measurement-noise σ
    diurnal_amp: float     # AM-PM amplitude (sinusoidal)
    drift_sigma: float     # daily random-walk σ (long-term drift)
    samples_per_day: int   # natural cadence


@dataclass(frozen=True)
class Archetype:
    id: str
    description: str
    profile: Profile
    metrics: dict[str, MetricSpec]


# Each archetype defines parameters for the seven metrics. Values are
# illustrative (literature-grounded but not precise clinical norms — for a
# thesis-MVP simulator this is what we ground the synthetic data in).
HEALTHY_70 = Archetype(
    id="healthy_70",
    description="Healthy 70-year-old; no chronic conditions",
    profile=Profile(),
    metrics={
        "pulse":       MetricSpec(mean=72,   std=4,   diurnal_amp=8,  drift_sigma=0.3, samples_per_day=4),
        "bp_sys":      MetricSpec(mean=130,  std=6,   diurnal_amp=10, drift_sigma=0.5, samples_per_day=3),
        "bp_dia":      MetricSpec(mean=80,   std=4,   diurnal_amp=6,  drift_sigma=0.3, samples_per_day=3),
        "glucose":     MetricSpec(mean=5.5,  std=0.4, diurnal_amp=1.2, drift_sigma=0.05, samples_per_day=4),
        "temperature": MetricSpec(mean=36.6, std=0.2, diurnal_amp=0.4, drift_sigma=0.02, samples_per_day=2),
        "spo2":        MetricSpec(mean=97,   std=1,   diurnal_amp=0.5, drift_sigma=0.05, samples_per_day=4),
        "weight":      MetricSpec(mean=68,   std=0.4, diurnal_amp=0.0, drift_sigma=0.05, samples_per_day=1),
    },
)

HYPER_75 = Archetype(
    id="hypertension_75",
    description="Hypertensive 75-year-old (controlled), shifted BP baseline",
    profile=Profile(hypertension=True),
    metrics={
        "pulse":       MetricSpec(mean=78,   std=5,   diurnal_amp=10, drift_sigma=0.4, samples_per_day=4),
        "bp_sys":      MetricSpec(mean=145,  std=8,   diurnal_amp=12, drift_sigma=0.8, samples_per_day=3),
        "bp_dia":      MetricSpec(mean=90,   std=5,   diurnal_amp=7,  drift_sigma=0.4, samples_per_day=3),
        "glucose":     MetricSpec(mean=5.7,  std=0.5, diurnal_amp=1.3, drift_sigma=0.05, samples_per_day=4),
        "temperature": MetricSpec(mean=36.6, std=0.2, diurnal_amp=0.4, drift_sigma=0.02, samples_per_day=2),
        "spo2":        MetricSpec(mean=96,   std=1,   diurnal_amp=0.5, drift_sigma=0.05, samples_per_day=4),
        "weight":      MetricSpec(mean=72,   std=0.4, diurnal_amp=0.0, drift_sigma=0.05, samples_per_day=1),
    },
)

T2D_78 = Archetype(
    id="t2d_78",
    description="Type-2 diabetic 78-year-old; shifted glucose baseline",
    profile=Profile(t2d=True),
    metrics={
        "pulse":       MetricSpec(mean=80,   std=5,   diurnal_amp=10, drift_sigma=0.4, samples_per_day=4),
        "bp_sys":      MetricSpec(mean=138,  std=7,   diurnal_amp=11, drift_sigma=0.7, samples_per_day=3),
        "bp_dia":      MetricSpec(mean=85,   std=5,   diurnal_amp=7,  drift_sigma=0.4, samples_per_day=3),
        "glucose":     MetricSpec(mean=7.2,  std=0.9, diurnal_amp=2.0, drift_sigma=0.10, samples_per_day=4),
        "temperature": MetricSpec(mean=36.6, std=0.2, diurnal_amp=0.4, drift_sigma=0.02, samples_per_day=2),
        "spo2":        MetricSpec(mean=97,   std=1,   diurnal_amp=0.5, drift_sigma=0.05, samples_per_day=4),
        "weight":      MetricSpec(mean=80,   std=0.4, diurnal_amp=0.0, drift_sigma=0.05, samples_per_day=1),
    },
)

COPD_80 = Archetype(
    id="copd_80",
    description="COPD 80-year-old; lower SpO2 baseline, faster pulse reserve",
    profile=Profile(copd=True),
    metrics={
        "pulse":       MetricSpec(mean=88,   std=6,   diurnal_amp=12, drift_sigma=0.5, samples_per_day=4),
        "bp_sys":      MetricSpec(mean=132,  std=7,   diurnal_amp=10, drift_sigma=0.6, samples_per_day=3),
        "bp_dia":      MetricSpec(mean=82,   std=5,   diurnal_amp=7,  drift_sigma=0.4, samples_per_day=3),
        "glucose":     MetricSpec(mean=5.6,  std=0.4, diurnal_amp=1.2, drift_sigma=0.05, samples_per_day=4),
        "temperature": MetricSpec(mean=36.7, std=0.3, diurnal_amp=0.5, drift_sigma=0.03, samples_per_day=2),
        "spo2":        MetricSpec(mean=93,   std=1.5, diurnal_amp=0.7, drift_sigma=0.10, samples_per_day=4),
        "weight":      MetricSpec(mean=65,   std=0.4, diurnal_amp=0.0, drift_sigma=0.05, samples_per_day=1),
    },
)


ALL_ARCHETYPES: list[Archetype] = [HEALTHY_70, HYPER_75, T2D_78, COPD_80]
ARCHETYPE_BY_ID: dict[str, Archetype] = {a.id: a for a in ALL_ARCHETYPES}

# Metrics evaluated. Order is stable (drives column ordering in result CSVs).
EVALUATED_METRICS: list[str] = ["pulse", "bp_sys", "bp_dia", "glucose", "temperature", "spo2", "weight"]
