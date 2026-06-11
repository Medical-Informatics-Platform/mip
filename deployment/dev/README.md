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
