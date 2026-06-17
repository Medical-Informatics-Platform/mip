#!/usr/bin/env bash
set -euo pipefail

set -a
source ./.env
set +a

compose_args=()
notebook_enabled=0
case "${NOTEBOOK_ENABLED:-0}" in
  1|true|TRUE|yes|YES)
    notebook_enabled=1
    compose_args+=(--profile notebook)
    echo "Notebook mode enabled: starting jupyterhub profile."
    ;;
  *)
    echo "Notebook mode disabled: starting core stack only."
    ;;
esac

docker compose "${compose_args[@]}" down -v --remove-orphans
if [ "${notebook_enabled}" = "1" ]; then
  welcome_src="../../../mip-jupyter/Welcome.ipynb"
  welcome_dst="./notebooks/Welcome.ipynb"
  if [ -f "${welcome_src}" ]; then
    mkdir -p ./notebooks
    cp "${welcome_src}" "${welcome_dst}"
    echo "Updated Welcome.ipynb from mip-jupyter."
  else
    echo "Warning: ${welcome_src} not found; skipping Welcome.ipynb sync." >&2
  fi
  # Rebuild notebook image so local mip-jupyter/python-client changes are applied.
  docker compose "${compose_args[@]}" up -d --build
else
  docker compose "${compose_args[@]}" up -d
fi

endpoint="http://172.17.0.1:8080/services/data-models"
max_attempts=30

for attempt in $(seq 1 "${max_attempts}"); do
  response="$(curl -fsS --max-time 5 "${endpoint}" 2>/dev/null || true)"
  if printf '%s' "${response}" | grep -q '"code"[[:space:]]*:[[:space:]]*"dementia_longitudinal"' \
    && printf '%s' "${response}" | grep -q '"code"[[:space:]]*:[[:space:]]*"dementia"' \
    && printf '%s' "${response}" | grep -q '"code"[[:space:]]*:[[:space:]]*"mentalhealth"' \
    && printf '%s' "${response}" | grep -q '"code"[[:space:]]*:[[:space:]]*"tbi"'
  then
    echo "Data models check passed: expected codes are available."
    echo "You can see the MIP at 172.17.0.1"
    if [ "${notebook_enabled}" = "1" ]; then
      echo "Notebook route is available at http://172.17.0.1/notebook"
      echo "Direct JupyterHub URL: http://172.17.0.1:8000/notebook"
    fi
    exit 0
  fi
  sleep 2
done

echo "Error: expected data model codes were not found at ${endpoint}." >&2
exit 1
