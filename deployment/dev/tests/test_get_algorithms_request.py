import json
import os

import requests

BASE_URL = os.environ.get("MIP_DEV_URL", "http://localhost:8080")


def test_get_algorithms_request():
    url = f"{BASE_URL}/services/algorithms"
    headers = {"Content-type": "application/json", "Accept": "application/json"}
    response = requests.get(url, headers=headers)
    assert response.status_code == 200
    algorithms = json.loads(response.text)
    assert len(algorithms) == 28
