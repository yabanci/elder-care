"""Replay backend/internal/baseline/testdata/parity_v2.jsonl through the
Go algo-runner subprocess and assert each produced output matches the
saved expected output. Exits non-zero on any drift, used in CI as a
cross-language regression guard."""
from __future__ import annotations

import json
import math
import sys
from datetime import datetime
from pathlib import Path

from .runner import AlgoInput, AlgoResult, Profile, Reading, algo_runner

ABS_TOL = 1e-6


def _parse_reading(raw: dict) -> Reading:
    return Reading(
        value=raw["value"],
        measured_at=datetime.fromisoformat(raw["measured_at"].replace("Z", "+00:00")),
    )


def _parse_input(raw: dict) -> AlgoInput:
    p = raw.get("profile") or {}
    n = raw.get("now")
    return AlgoInput(
        kind=raw["kind"],
        value=raw["value"],
        history=[_parse_reading(r) for r in raw.get("history") or []],
        profile=Profile(
            hypertension=bool(p.get("hypertension", False)),
            t2d=bool(p.get("t2d", False)),
            copd=bool(p.get("copd", False)),
        ),
        estimator=raw.get("estimator", ""),
        now=datetime.fromisoformat(n.replace("Z", "+00:00")) if n else None,
    )


def _results_match(got: AlgoResult, want: AlgoResult) -> tuple[bool, str]:
    fields_str = ("severity", "reason_code", "estimator", "algorithm_version")
    fields_bool_int = ("used_history", "history_size")
    fields_float = ("mean", "std", "z_score")
    for f in fields_str + fields_bool_int:
        if getattr(got, f) != getattr(want, f):
            return False, f"{f}: got={getattr(got, f)!r} want={getattr(want, f)!r}"
    for f in fields_float:
        if not math.isclose(getattr(got, f), getattr(want, f), abs_tol=ABS_TOL):
            return False, f"{f}: got={getattr(got, f)} want={getattr(want, f)}"
    return True, ""


def replay(jsonl_path: Path) -> int:
    failures: list[str] = []
    with algo_runner() as runner:
        with jsonl_path.open() as f:
            for lineno, line in enumerate(f, start=1):
                line = line.strip()
                if not line:
                    continue
                rec = json.loads(line)
                inp = _parse_input(rec["input"])
                want = AlgoResult.from_json(rec["expected"])
                got = runner.call(inp)
                ok, why = _results_match(got, want)
                if not ok:
                    failures.append(f"  case {rec.get('name', f'#{lineno}')}: {why}")
    if failures:
        print(f"FAIL: {len(failures)} parity drift(s):", file=sys.stderr)
        for f in failures:
            print(f, file=sys.stderr)
        return 1
    print("PASS: all parity cases match")
    return 0


def main() -> int:
    if len(sys.argv) != 2:
        print("usage: python -m eldercare_eval.parity <path/to/parity_v2.jsonl>", file=sys.stderr)
        return 2
    path = Path(sys.argv[1])
    if not path.exists():
        print(f"file not found: {path}", file=sys.stderr)
        return 2
    return replay(path)


if __name__ == "__main__":
    raise SystemExit(main())
