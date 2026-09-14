import os

import requests

JUPYTER_URL = os.environ.get("MIP_JUPYTER_URL", "http://localhost:8888")
JUPYTER_TOKEN = os.environ.get("JUPYTER_TOKEN", "dev")


def test_jupyter_lab_is_up():
    response = requests.get(
        f"{JUPYTER_URL}/api/status",
        params={"token": JUPYTER_TOKEN},
        timeout=10,
    )
    assert response.status_code == 200


def test_feres_notebook_is_reachable():
    response = requests.get(
        f"{JUPYTER_URL}/lab/tree/examples/feres_analysis.ipynb",
        params={"token": JUPYTER_TOKEN},
        timeout=10,
        allow_redirects=True,
    )
    assert response.status_code == 200
