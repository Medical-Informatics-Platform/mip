#!/bin/bash
set -e          # stop the pipeline if any command fails
set -o pipefail # Prevents silent pipeline successes if the curl download drops
set -u          # treat unset variables as an error

trap 'echo "[setup-tools] ERROR: command failed (exit $?) at line $LINENO: $BASH_COMMAND" >&2' ERR

# Tool versions and the SHA256 of the release asset we download.
# The "# renovate:" markers let Renovate bump version and checksum together
# (see renovate.json). When overriding a *_VERSION via env, the matching
# *_SHA256 must be overridden as well or verification will fail.

# renovate: datasource=github-release-attachments depName=aquasecurity/trivy
TRIVY_VERSION="${TRIVY_VERSION:-v0.76.1}"
TRIVY_SHA256="${TRIVY_SHA256:-c7cbe3137fb8a48a17574bede3dd6beb16742d1e9772929940fc90082c39d9c0}"

# renovate: datasource=github-release-attachments depName=google/osv-scanner
OSV_SCANNER_VERSION="${OSV_SCANNER_VERSION:-v2.7.0}"
OSV_SCANNER_SHA256="${OSV_SCANNER_SHA256:-160c6253e2922e9e0a6a9771c5c4c40a4b3c6090b34a3cdd5e80c96cc2e4218b}"

# renovate: datasource=github-release-attachments depName=opengrep/opengrep
OPENGREP_VERSION="${OPENGREP_VERSION:-v1.32.0}"
OPENGREP_SHA256="${OPENGREP_SHA256:-7360d60174880bd9214f2dfbe439b7bce82b7a3d44cfe945ba833c85cc8bda75}"

# renovate: datasource=git-refs depName=https://github.com/semgrep/semgrep-rules
SEMGREP_RULES_REF="${SEMGREP_RULES_REF:-16b9e6d45f10649f6a531ec4838ed9924372c3bc}"
SEMGREP_RULES_DIR="semgrep-rules"

# renovate: datasource=github-release-attachments depName=hadolint/hadolint
HADOLINT_VERSION="${HADOLINT_VERSION:-v2.15.1}"
HADOLINT_SHA256="${HADOLINT_SHA256:-c7187db94eeeeca956519a6af171adc31453941a1e777961f6e680f697c8c507}"

# renovate: datasource=github-release-attachments depName=gitleaks/gitleaks
GITLEAKS_VERSION="${GITLEAKS_VERSION:-v8.30.1}"
GITLEAKS_SHA256="${GITLEAKS_SHA256:-551f6fc83ea457d62a0d98237cbad105af8d557003051f41f3e7ca7b3f2470eb}"

# renovate: datasource=npm depName=@cyclonedx/cyclonedx-npm
CYCLONEDX_NPM_VERSION="${CYCLONEDX_NPM_VERSION:-6.0.1}"

TMP_DIR="$(mktemp -d)"
trap 'rm -rf "${TMP_DIR}"' EXIT

# --- Flag parsing -----------------------------------------------------
INSTALL_TOOL="none"
SBOM_ECOSYSTEM="none"

while [[ $# -gt 0 ]]; do
  case "$1" in
    --install-tool)
      [[ $# -ge 2 ]] || { echo "[setup-tools] --install-tool requires a value (e.g. trivy,osv-scanner|all)" >&2; exit 1; }
      INSTALL_TOOL="$2"
      shift 2
      ;;
    --sbom-ecosystem)
      [[ $# -ge 2 ]] || { echo "[setup-tools] --sbom-ecosystem requires a value (e.g. maven|npm|none)" >&2; exit 1; }
      SBOM_ECOSYSTEM="$2"
      shift 2
      ;;
    *)
      echo "[setup-tools] Unknown flag: $1" >&2
      exit 1
      ;;
  esac
done

should_install() {
  [[ "$INSTALL_TOOL" == "all" || ",$INSTALL_TOOL," == *",$1,"* ]]
}

# Download a file and refuse to proceed unless its SHA256 matches the pinned one
download_and_verify() {
  local url="$1" dest="$2" sha256="$3"
  curl -fsSL --retry 3 "${url}" -o "${dest}"
  echo "${sha256}  ${dest}" | sha256sum -c -
}

# --- Trivy --------------------------------------------------------------
if should_install "trivy"; then
  echo "[setup-tools] Installing Trivy ${TRIVY_VERSION}"
  TRIVY_TARBALL="trivy_${TRIVY_VERSION#v}_Linux-64bit.tar.gz"
  download_and_verify \
    "https://github.com/aquasecurity/trivy/releases/download/${TRIVY_VERSION}/${TRIVY_TARBALL}" \
    "${TMP_DIR}/${TRIVY_TARBALL}" \
    "${TRIVY_SHA256}"
  sudo tar -xzf "${TMP_DIR}/${TRIVY_TARBALL}" -C /usr/local/bin trivy
  trivy --version
  echo "Trivy installed OK"
fi

# --- OSV Scanner ----------------------------------------------------------
if should_install "osv-scanner"; then
  echo "[setup-tools] Installing OSV Scanner ${OSV_SCANNER_VERSION}"
  download_and_verify \
    "https://github.com/google/osv-scanner/releases/download/${OSV_SCANNER_VERSION}/osv-scanner_linux_amd64" \
    "${TMP_DIR}/osv-scanner" \
    "${OSV_SCANNER_SHA256}"
  sudo install -m 0755 "${TMP_DIR}/osv-scanner" /usr/local/bin/osv-scanner
  osv-scanner --version
  echo "OSV Scanner installed OK"
fi

# --- OpenGrep -------------------------------------------------------------
if should_install "opengrep"; then
  echo "[setup-tools] Installing OpenGrep ${OPENGREP_VERSION}"
  download_and_verify \
    "https://github.com/opengrep/opengrep/releases/download/${OPENGREP_VERSION}/opengrep_manylinux_x86" \
    "${TMP_DIR}/opengrep" \
    "${OPENGREP_SHA256}"
  sudo install -m 0755 "${TMP_DIR}/opengrep" /usr/local/bin/opengrep
  opengrep --version
  echo "OpenGrep installed OK"
fi

# --- Semgrep community ruleset (cloned, not registry) ---------
if should_install "semgrep-rules"; then
  echo "[setup-tools] Cloning semgrep-rules @ ${SEMGREP_RULES_REF}"
  rm -rf "${SEMGREP_RULES_DIR}"
  git clone --quiet https://github.com/semgrep/semgrep-rules.git "${SEMGREP_RULES_DIR}"
  git -C "${SEMGREP_RULES_DIR}" checkout --quiet "${SEMGREP_RULES_REF}"
  echo "semgrep-rules ready at ${SEMGREP_RULES_DIR} (ref: ${SEMGREP_RULES_REF})"
fi

# --- Hadolint ---------------------------------------------------------
if should_install "hadolint"; then
  echo "[setup-tools] Installing Hadolint ${HADOLINT_VERSION}"
  download_and_verify \
    "https://github.com/hadolint/hadolint/releases/download/${HADOLINT_VERSION}/hadolint-linux-x86_64" \
    "${TMP_DIR}/hadolint" \
    "${HADOLINT_SHA256}"
  sudo install -m 0755 "${TMP_DIR}/hadolint" /usr/local/bin/hadolint
  hadolint --version
  echo "Hadolint installed OK"
fi

# --- Gitleaks -----------------------------------------------------------
if should_install "gitleaks"; then
  echo "[setup-tools] Installing Gitleaks ${GITLEAKS_VERSION}"
  GITLEAKS_TARBALL="gitleaks_${GITLEAKS_VERSION#v}_linux_x64.tar.gz"
  download_and_verify \
    "https://github.com/gitleaks/gitleaks/releases/download/${GITLEAKS_VERSION}/${GITLEAKS_TARBALL}" \
    "${TMP_DIR}/${GITLEAKS_TARBALL}" \
    "${GITLEAKS_SHA256}"
  sudo tar -xzf "${TMP_DIR}/${GITLEAKS_TARBALL}" -C /usr/local/bin gitleaks
  gitleaks version
  echo "Gitleaks installed OK"
fi

# --- SBOM generation ----------------------------------------------------
case "$SBOM_ECOSYSTEM" in
  maven)
    echo "Generating SBOM for Maven project"
    mvn -B -ntp dependency:resolve -q
    mvn -B -ntp org.cyclonedx:cyclonedx-maven-plugin:makeAggregateBom -q
    ;;
  npm)
    echo "Generating SBOM for NPM project"
    npm ci
    npx --yes "@cyclonedx/cyclonedx-npm@${CYCLONEDX_NPM_VERSION}" --output-file target/bom.json
    ;;
  none)
    echo "No SBOM generation needed"
    ;;
  *)
    echo "Unknown SBOM_ECOSYSTEM: $SBOM_ECOSYSTEM" >&2
    exit 1
    ;;
esac
