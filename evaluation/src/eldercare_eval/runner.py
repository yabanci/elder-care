"""Subprocess driver for the Go algo-runner binary.

The Go binary at `backend/cmd/algo-runner` reads one JSON Input per stdin
line and writes one JSON Result per stdout line. We keep one subprocess
alive for an entire algorithm sweep (typically thousands of calls) so
per-invocation startup cost is paid once.
"""
from __future__ import annotations

import json
import os
import subprocess
import threading
from contextlib import contextmanager
from dataclasses import asdict, dataclass, field
from datetime import datetime, timezone
from pathlib import Path
from typing import Iterator, Optional


@dataclass
class Reading:
    value: float
    measured_at: datetime

    def to_json(self) -> dict:
        return {
            "value": self.value,
            "measured_at": self.measured_at.astimezone(timezone.utc).isoformat().replace("+00:00", "Z"),
        }


@dataclass
class Profile:
    hypertension: bool = False
    t2d: bool = False
    copd: bool = False

    def to_json(self) -> dict:
        return asdict(self)


@dataclass
class AlgoInput:
    kind: str
    value: float
    history: list[Reading] = field(default_factory=list)
    profile: Profile = field(default_factory=Profile)
    estimator: str = ""
    now: Optional[datetime] = None

    def to_json(self) -> dict:
        out: dict = {
            "kind": self.kind,
            "value": self.value,
            "history": [r.to_json() for r in self.history],
            "profile": self.profile.to_json(),
        }
        if self.estimator:
            out["estimator"] = self.estimator
        if self.now is not None:
            out["now"] = self.now.astimezone(timezone.utc).isoformat().replace("+00:00", "Z")
        return out


@dataclass
class AlgoResult:
    severity: str
    reason_code: str
    mean: float
    std: float
    z_score: float
    estimator: str
    used_history: bool
    history_size: int
    algorithm_version: str

    @classmethod
    def from_json(cls, raw: dict) -> "AlgoResult":
        return cls(
            severity=raw["severity"],
            reason_code=raw["reason_code"],
            mean=raw.get("mean", 0.0) or 0.0,
            std=raw.get("std", 0.0) or 0.0,
            z_score=raw.get("z_score", 0.0) or 0.0,
            estimator=raw["estimator"],
            used_history=raw["used_history"],
            history_size=raw["history_size"],
            algorithm_version=raw["algorithm_version"],
        )


def _resolve_binary() -> Path:
    env = os.environ.get("ALGO_RUNNER")
    if env:
        return Path(env).resolve()
    # Default: assume we're invoked from evaluation/ and the binary is at
    # ../backend/_bin/algo-runner.
    fallback = Path(__file__).resolve().parents[3] / "backend" / "_bin" / "algo-runner"
    return fallback


class AlgoRunner:
    """Long-lived subprocess driver. Not thread-safe."""

    def __init__(self, binary: Optional[Path] = None) -> None:
        self.binary = binary or _resolve_binary()
        if not self.binary.exists():
            raise FileNotFoundError(
                f"algo-runner binary not found at {self.binary}. "
                "Run `make algo-runner` (or set ALGO_RUNNER env var)."
            )
        self._proc: Optional[subprocess.Popen] = None
        self._stderr_thread: Optional[threading.Thread] = None
        self._lock = threading.Lock()

    def __enter__(self) -> "AlgoRunner":
        self.start()
        return self

    def __exit__(self, *exc) -> None:
        self.stop()

    def start(self) -> None:
        if self._proc is not None:
            return
        self._proc = subprocess.Popen(
            [str(self.binary)],
            stdin=subprocess.PIPE,
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE,
            text=True,
            bufsize=1,  # line-buffered
        )
        self._stderr_thread = threading.Thread(
            target=self._drain_stderr, daemon=True
        )
        self._stderr_thread.start()

    def _drain_stderr(self) -> None:
        assert self._proc is not None and self._proc.stderr is not None
        for line in self._proc.stderr:
            # Forward go log.Printf lines to our stderr — useful for triage.
            print(f"[algo-runner] {line.rstrip()}", flush=True)

    def stop(self) -> None:
        if self._proc is None:
            return
        try:
            if self._proc.stdin is not None:
                self._proc.stdin.close()
            self._proc.wait(timeout=5)
        except subprocess.TimeoutExpired:
            self._proc.kill()
            self._proc.wait()
        self._proc = None

    def call(self, inp: AlgoInput) -> AlgoResult:
        with self._lock:
            if self._proc is None:
                raise RuntimeError("AlgoRunner not started")
            assert self._proc.stdin is not None and self._proc.stdout is not None
            self._proc.stdin.write(json.dumps(inp.to_json()) + "\n")
            self._proc.stdin.flush()
            line = self._proc.stdout.readline()
            if not line:
                raise RuntimeError("algo-runner closed stdout unexpectedly")
            raw = json.loads(line)
            if "error" in raw:
                raise RuntimeError(f"algo-runner error: {raw['error']}")
            return AlgoResult.from_json(raw)

    def call_many(self, inputs: list[AlgoInput]) -> list[AlgoResult]:
        return [self.call(i) for i in inputs]


@contextmanager
def algo_runner(binary: Optional[Path] = None) -> Iterator[AlgoRunner]:
    r = AlgoRunner(binary=binary)
    r.start()
    try:
        yield r
    finally:
        r.stop()
