"""Dev-only JupyterHub config matching the platform-ui /notebook route."""

import os
import shutil
import sys

if "/srv/jupyterhub" not in sys.path:
    sys.path.insert(0, "/srv/jupyterhub")

from jupyterhub_shared import (
    bool_env,
    build_singleuser_csp_arg,
    configure_dummy_auth,
    configure_generic_oauth,
    extract_access_token,
    install_platform_token_handler,
)

c = get_config()

AUTHENTICATION_ENABLED = bool_env("AUTHENTICATION", default=True)
FRAME_ANCESTORS = os.environ.get("JH_FRAME_ANCESTORS", "'self' http://localhost:4200")

# Serve hub and single-user servers behind /notebook.
c.JupyterHub.bind_url = "http://:8000/notebook"

# Allow embedding inside the Platform UI iframe.
c.JupyterHub.tornado_settings = {
    "headers": {
        "Content-Security-Policy": f"frame-ancestors {FRAME_ANCESTORS}",
    }
}

keycloak_auth_url = os.environ.get("KEYCLOAK_AUTH_URL", "https://iam.ebrains.eu/auth/")
realm = os.environ.get("KEYCLOAK_REALM", "MIP")

if AUTHENTICATION_ENABLED:
    configure_generic_oauth(
        c,
        auth_url=keycloak_auth_url,
        realm=realm,
        client_id=(os.environ.get("JH_OAUTH_CLIENT_ID") or os.environ.get("KEYCLOAK_CLIENT_ID") or "mippublic"),
        client_secret=(os.environ.get("JH_OAUTH_CLIENT_SECRET") or os.environ.get("KEYCLOAK_CLIENT_SECRET") or ""),
        callback_url=os.environ.get(
            "JH_OAUTH_CALLBACK_URL",
            "http://localhost:4200/notebook/hub/oauth_callback",
        ),
    )
else:
    configure_dummy_auth(c, password=os.environ.get("JH_DUMMY_PASSWORD", ""))


async def _inject_portal_token(spawner):
    # Refresh Welcome notebook from the image on every spawn.
    src = "/opt/portal-notebooks/Welcome.ipynb"
    dst_dir = "/home/jovyan/work"
    dst = os.path.join(dst_dir, "Welcome.ipynb")
    os.makedirs(dst_dir, exist_ok=True)
    if os.path.exists(src):
        shutil.copy2(src, dst)

    spawner.environment.setdefault(
        "PLATFORM_BACKEND_URL",
        os.environ.get("PLATFORM_BACKEND_URL", "http://platform-backend:8080/services"),
    )

    if not AUTHENTICATION_ENABLED:
        static_token = os.environ.get("PORTAL_TOKEN")
        if static_token:
            spawner.environment["PORTAL_TOKEN"] = static_token
        return

    # Inject bearer token for platform-backend API calls.
    auth_state = await spawner.user.get_auth_state()
    token = extract_access_token(auth_state)
    if not token:
        spawner.log.warning("No access_token in auth_state; PORTAL_TOKEN will not be set")
        return

    spawner.environment["PORTAL_TOKEN"] = token


# Keep dev simple: one-process local spawner.
c.JupyterHub.spawner_class = "simple"

# Launch JupyterLab for single-user sessions.
c.Spawner.cmd = ["jupyter-labhub"]
c.Spawner.default_url = "/lab/tree/Welcome.ipynb"
c.Spawner.notebook_dir = "/home/jovyan/work"
c.Spawner.args = [
    "--allow-root",
    # Hub CSP is not enough; single-user server must also allow iframe embedding.
    build_singleuser_csp_arg(FRAME_ANCESTORS),
]
c.Spawner.pre_spawn_hook = _inject_portal_token

# Exposed at /notebook/hub/api/portal-token
install_platform_token_handler(c, authentication_enabled=AUTHENTICATION_ENABLED)
