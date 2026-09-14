# Development deployment

Local Docker Compose runs the MIP UI/backend/engine plus a **standalone JupyterLab** container. Kubernetes notebooks are different: JupyterHub behind platform-ui `/notebook/` with Keycloak. Compose cannot run that Hub path (KubeSpawner needs Kubernetes).

## Requirements
### Hardware
* 40 GB HDD
* 8 GB RAM
* 2 CPU Cores

### Software
* Ubuntu Server (minimal installation, without GUI)
* Docker Compose
* Python 3.10 or newer, only if you run `./test.sh`

## Instructions to deploy:

1. Clone the repo

2. Go to the dev deployment folder:
    ```
    cd deployment/dev/
    ```

3. Copy the .env file:
    ```
    cp .env.example .env
    ```

4. To start the MIP stack run the `start.sh` script:
    ```
    ./start.sh
    ```
    The script pulls images, starts the stack plus JupyterLab, then checks that the four data models and JupyterLab are available.

    Open:
    ```
    http://localhost
    http://localhost:8888/lab/tree/examples/feres_analysis.ipynb?token=dev
    ```

    JupyterLab uses the compose backend URL `http://platform-backend:8080/services` inside Docker. The notebook route in the UI stays disabled (`NOTEBOOK_ENABLED=0`) because that iframe talks to JupyterHub, not this Lab container.

5. Optional smoke tests against the running stack:
    ```
    ./test.sh
    ```

6. To stop the MIP stack run the `stop.sh` script:
    ```
    ./stop.sh
    ```
