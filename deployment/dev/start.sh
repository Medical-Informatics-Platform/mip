#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "${script_dir}"

if [[ ! -f .env ]]; then
  echo "Error: .env not found. Copy .env.example to .env." >&2
  exit 1
fi

set -a
# shellcheck disable=SC1091
source ./.env
set +a

jupyter_token="${JUPYTER_TOKEN:-dev}"
data_models_url="http://localhost:8080/services/data-models"
jupyter_url="http://localhost:8888/lab/tree/examples/feres_analysis.ipynb?token=${jupyter_token}"
jupyter_status_url="http://localhost:8888/api/status?token=${jupyter_token}"

docker compose down -v
docker compose pull
docker compose up -d

data_models_ready=0
for _ in {1..45}; do
  response="$(curl -fsS --max-time 5 "${data_models_url}" 2>/dev/null || true)"
  if printf '%s' "${response}" | grep -q '"code"[[:space:]]*:[[:space:]]*"dementia_longitudinal"' \
    && printf '%s' "${response}" | grep -q '"code"[[:space:]]*:[[:space:]]*"dementia"' \
    && printf '%s' "${response}" | grep -q '"code"[[:space:]]*:[[:space:]]*"mentalhealth"' \
    && printf '%s' "${response}" | grep -q '"code"[[:space:]]*:[[:space:]]*"tbi"'
  then
    echo "Data models check passed: expected codes are available."
    data_models_ready=1
    break
  fi
  sleep 2
done

if [[ "${data_models_ready}" -ne 1 ]]; then
  echo "Error: expected data model codes were not found at ${data_models_url}." >&2
  exit 1
fi

jupyter_ready=0
for _ in {1..30}; do
  if curl -fsS --max-time 5 -o /dev/null "${jupyter_status_url}"; then
    echo "JupyterLab check passed."
    jupyter_ready=1
    break
  fi
  sleep 2
done

if [[ "${jupyter_ready}" -ne 1 ]]; then
  echo "Error: JupyterLab was not ready at ${jupyter_status_url}." >&2
  exit 1
fi

echo "MIP UI: http://localhost"
echo "JupyterLab: ${jupyter_url}"
echo "Kubernetes notebooks use JupyterHub behind /notebook/; this compose stack uses JupyterLab on :8888."
