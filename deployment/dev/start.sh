#!/usr/bin/env bash
set -euo pipefail

docker compose down -v
docker compose up -d --build

endpoint="http://172.17.0.1:8080/services/data-models"
for _ in {1..30}; do
  response="$(curl -fsS --max-time 5 "${endpoint}" 2>/dev/null || true)"
  if printf "%s" "${response}" | grep -q "\"code\"[[:space:]]*:[[:space:]]*\"dementia\""; then
    echo "Data model check passed: dementia is available."
    echo "MIP UI: http://localhost"
    echo "JupyterLab: http://localhost:8888/lab/tree/workspace/examples/feres_analysis.ipynb?token=dev"
    exit 0
  fi
  sleep 2
done

echo "Error: dementia data model was not found at ${endpoint}." >&2
exit 1
