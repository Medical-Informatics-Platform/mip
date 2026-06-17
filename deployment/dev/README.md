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

2. Install docker-compose


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

3. To start the MIP stack run the 'start.sh' script to setup all the containers:
    ```
    ./start.sh
    ```
    The script waits for `http://172.17.0.1:8080/services/data-models` and verifies 4 data models are loaded.

4. To test if the MIP stack is properly setup run the 'test.sh':
    ```
    ./test.sh 
    ```
   
5. To stop the MIP stack run the 'stop.sh' script to stop all the containers:
    ```
    ./stop.sh
    ```

## Notebook mode
To run the stack with notebook support behind `/notebook`:

1. In `.env`, set:
   ```
   NOTEBOOK_ENABLED=1
   JUPYTER_SERVER=jupyterhub:8000
   JUPYTER_CONTEXT=notebook
   JUPYTER_LANDING_PATH=/hub/spawn
   JH_OAUTH_CALLBACK_URL=http://localhost/notebook/hub/oauth_callback
   ```
2. Start as usual:
   ```
   ./start.sh
   ```
3. Open:
   * Platform UI: `http://172.17.0.1`
   * Notebook route: `http://172.17.0.1/notebook`

Notes:
* `start.sh` enables the `jupyterhub` compose profile automatically when `NOTEBOOK_ENABLED=1`.
* If `AUTHENTICATION=1`, make sure Keycloak values in `.env` are valid.
* If `AUTHENTICATION=0`, JupyterHub uses a local dummy authenticator (no Keycloak dependency).
* `jupyterhub_config.py` keeps deployment-specific wiring while shared auth/token helper logic lives in `mip-jupyter/jupyterhub_shared.py`.
* `platform_ui` uses a local `nginx.conf.template` override in this folder to enable Docker DNS resolution for notebook upstream routing. Keep this until the selected `hbpmip/platform-ui:${PLATFORM_UI}` image includes the resolver fix.
