import json
import os

import requests

BASE_URL = os.environ.get("MIP_DEV_URL", "http://localhost:8080")


def test_get_pathologies_request():
    url = f"{BASE_URL}/services/data-models"
    headers = {"Content-type": "application/json", "Accept": "application/json"}
    response = requests.get(url, headers=headers)
    assert response.status_code == 200
    pathologies = json.loads(response.text)
    assert len(pathologies) == 4

    pathology_codes = {pathology["code"] for pathology in pathologies}
    assert pathology_codes == {"dementia_longitudinal", "dementia", "mentalhealth", "tbi"}

    datasets_count_by_code = {pathology["code"]: len(pathology["datasets"]) for pathology in pathologies}
    assert datasets_count_by_code == {
        "dementia": 3,
        "dementia_longitudinal": 1,
        "mentalhealth": 1,
        "tbi": 1,
    }

    dataset_enum_count_by_code = {pathology["code"]: count_datasets_from_cdes(pathology) for pathology in pathologies}
    assert dataset_enum_count_by_code == {
        "dementia": 3,
        "dementia_longitudinal": 4,
        "mentalhealth": 1,
        "tbi": 1,
    }

    cdes_count_by_code = {pathology["code"]: count_cdes(pathology) for pathology in pathologies}
    assert cdes_count_by_code == {
        "dementia": 184,
        "dementia_longitudinal": 185,
        "mentalhealth": 191,
        "tbi": 20,
    }


def count_cdes(metadata_hierarchy) -> int:
    counter = 0

    if "variables" in metadata_hierarchy:
        counter += len(metadata_hierarchy["variables"])

    if "groups" in metadata_hierarchy:
        for cde in metadata_hierarchy["groups"]:
            counter += count_cdes(cde)
    return counter


def count_datasets_from_cdes(metadata_hierarchy) -> int:
    if "variables" in metadata_hierarchy:
        for variable in metadata_hierarchy["variables"]:
            if variable["code"] == "dataset":
                return len(variable["enumerations"])

    if "groups" in metadata_hierarchy:
        for cde in metadata_hierarchy["groups"]:
            counter = count_datasets_from_cdes(cde)
            if counter != 0:
                return counter
    return 0
