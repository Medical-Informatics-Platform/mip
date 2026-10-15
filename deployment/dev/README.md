# Development deployment

## Requirements
### Hardware
* 40 GB HDD
* 8 GB RAM
* 2 CPU Cores

### Software
* Ubuntu Server (minimal installation, without GUI)

### Prerequisites

1. Install [python3.10](https://www.python.org/downloads/ "python3.10")

2. Install Docker Engine with the Docker Compose plugin (`docker compose`).


## Instructions to deploy:

1. Clone the repo

2. Go to the dev deployment folder:
    ```
    cd mip/deployment/dev/
    ``` 

3. Copy the .env file:
    ```
    cp .env.example .env
    ```

    The development helpers are reset-oriented: both `start.sh` and `stop.sh` run
    `docker compose down -v`. The notebook container has no persistent home mount;
    copy personal notebooks out before using these scripts. Use `docker compose stop` and
    `docker compose start` to stop and resume existing containers.
    See [Backup and Recovery](../docs/BackupAndRecovery.md) for preserving data.

4. To start the MIP stack run the 'start.sh' script to setup all the containers:
    ```
    ./start.sh
    ```
    The script starts the MIP stack plus direct JupyterLab, then checks that the `dementia` data model is available.

    Open:
    ```
    http://localhost
    http://localhost:8888/lab/tree/Welcome.ipynb?token=dev
    ```

    JupyterLab uses the compose backend URL `http://platform-backend:8080/services` inside Docker. It is published on `127.0.0.1` only; change `JUPYTER_TOKEN` in `.env` if the host is shared. With `AUTHENTICATION=1`, set `MIP_TOKEN` in `.env` to a Keycloak access token so the notebook client can call the backend.

    Examples live in the public repo `Medical-Informatics-Platform/mip-examples`. To have them
    locally, run `git clone https://github.com/Medical-Informatics-Platform/mip-examples.git mip-examples`
    in this folder (or set `MIP_EXAMPLES_DIR`) before the first `./start.sh`, then open
    `examples/FERES/feres_analysis.ipynb` (or set `MIP_NOTEBOOK=examples/FERES/feres_analysis.ipynb` in `.env`).
    They are mounted read-only; copy one into `scratch/` to edit it. Without a checkout, Docker
    creates an empty, root-owned `mip-examples` folder and `examples/` stays empty; remove that
    folder with `sudo rmdir mip-examples` before cloning.

5. To stop the MIP stack run the 'stop.sh' script to stop all the containers:
    ```
    ./stop.sh
    ```

## Dataset onboarding and analysis preparation

For source CSVs, data-model metadata, and dataset validation, follow the
[Data Management Guide](../../documentation/MIP_Data_management_documentation.md).
Configure per-analysis filters and transformations in the Experiment Studio or
notebook pipelines.
The development setup exposes JupyterLab directly; portal-integrated JupyterHub
and authenticated notebook spawning are covered by the
[Kubernetes guide](../kubernetes/README.md).
