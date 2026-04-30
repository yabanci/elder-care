SHELL := /bin/bash
.PHONY: venv install algo-runner clean parity smoke eval results

PY := python3
VENV := .venv
ACTIVATE := source $(VENV)/bin/activate
ALGO_RUNNER := ../backend/_bin/algo-runner

venv:
	@test -d $(VENV) || $(PY) -m venv $(VENV)

install: venv
	@$(ACTIVATE) && pip install -q -U pip && pip install -q -r requirements.txt

algo-runner:
	@mkdir -p ../backend/_bin
	@cd ../backend && go build -o _bin/algo-runner ./cmd/algo-runner
	@echo "built $(ALGO_RUNNER)"

# parity: replay backend/internal/baseline/testdata/parity_v2.jsonl through
# the Go algo-runner and assert each output matches the saved expectation.
parity: install algo-runner
	@$(ACTIVATE) && ALGO_RUNNER=$(ALGO_RUNNER) PYTHONPATH=src $(PY) -m eldercare_eval.parity \
		../backend/internal/baseline/testdata/parity_v2.jsonl

# smoke: tiny eval run (1 archetype, 100 samples) — F1 regression guard for CI.
smoke: install algo-runner
	@$(ACTIVATE) && ALGO_RUNNER=$(ALGO_RUNNER) PYTHONPATH=src $(PY) -m eldercare_eval.cli smoke

# eval: full evaluation — all archetypes, all algorithms, all metrics, plots, report.
eval: install algo-runner
	@$(ACTIVATE) && ALGO_RUNNER=$(ALGO_RUNNER) PYTHONPATH=src $(PY) -m eldercare_eval.cli eval

# eval-stretch: optional real-data slide (BIDMC). Currently stubbed.
eval-stretch: install algo-runner
	@$(ACTIVATE) && ALGO_RUNNER=$(ALGO_RUNNER) PYTHONPATH=src $(PY) -m eldercare_eval.cli eval-stretch

clean:
	rm -rf $(VENV) data figures __pycache__ .pytest_cache src/*.egg-info
	rm -f ../backend/_bin/algo-runner
