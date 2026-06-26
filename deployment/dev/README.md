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
    The script rebuilds and starts the MIP stack plus direct JupyterLab, then checks that the `dementia` data model is available.

    Open:
    ```
    http://localhost
    http://localhost:8888/lab/tree/workspace/examples/feres_analysis.ipynb?token=dev
    ```

    JupyterLab uses the compose backend URL `http://platform-backend:8080/services` inside Docker.

4. To stop the MIP stack run the 'stop.sh' script to stop all the containers:
    ```
    ./stop.sh
    ```
