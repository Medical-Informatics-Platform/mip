import json
import os
from dataclasses import dataclass

GATE_FAIL_THRESHOLD = float(os.getenv("GATE_FAIL_THRESHOLD", "8.0"))
GATE_WARN_THRESHOLD = float(os.getenv("GATE_WARN_THRESHOLD", "5.0"))

def drop_suppressed(path):
    # opengrep keeps `# nosemgrep` findings in SARIF (suppressions: inSource),
    # and the GitHub Security tab still opens alerts for them.
    if not os.path.exists(path):
        return
    with open(path) as f:
        sarif = json.load(f)
    for run in sarif.get("runs", []):
        run["results"] = [r for r in run.get("results", []) if not r.get("suppressions")]
    with open(path, "w") as f:
        json.dump(sarif, f)

@dataclass
class EvaluationResult:
    gate_failed: bool
    gate_warn: bool

def evaluate(sarif_paths):
    if isinstance(sarif_paths, str):
        sarif_paths = [sarif_paths]

    max_score = 0.0
    unscored = 0
    for path in sarif_paths:
        try:
            with open(path) as f:
                sarif = json.load(f, strict=False)
        except FileNotFoundError:
            raise FileNotFoundError(f"[parse_sarif] SARIF file not found: {path}")
        except json.JSONDecodeError as e:
            raise ValueError(f"[parse_sarif] SARIF file at {path} is not valid JSON: {e}")

        for run in sarif.get("runs", []):
            try:
                rules = run["tool"]["driver"].get("rules", [])
            except KeyError as e:
                raise KeyError(f"[parse_sarif] SARIF file at {path} has a run missing expected field {e}")

            severities = {r["id"]: r.get("properties", {}).get("security-severity") for r in rules}

            for result in run.get("results", []):
                score = severities.get(result["ruleId"])
                if score is not None:
                    max_score = max(max_score, float(score))
                else:
                    unscored += 1

    # The gate cannot judge these, so they must not read as a clean pass.
    if unscored:
        print(f"::warning title=Unscored findings::{unscored} result(s) in {', '.join(sarif_paths)} have no security-severity; the CVSS gate did not evaluate them.")

    return EvaluationResult(
            gate_failed=max_score >= GATE_FAIL_THRESHOLD,
            gate_warn=unscored > 0 or GATE_WARN_THRESHOLD <= max_score < GATE_FAIL_THRESHOLD,
        )
