# Examples

The folder contains different examples of how to use the FOOD-Me proxy with different functionalities, databases and drivers.

## Requirements

Docker/Podman/Container technology.

I'm just going to assume you have `docker` and `docker compose` commands available going forward.

## How to run the examples?

1. Navigate to the example directory
2. Build the example: `docker build -f Dockerfile -t foodme:example .`
3. Start the ensemble: `docker compose up --build`
4. Wait until all the services are running
5. Run the example image `docker run --network host foodme:example`

Alternatively, you can run the example container image in an interactive mode `docker run --network host -it foodme:example bash`, initiate the poetry shell `poetry shell` and start a python interactive shell `python`. Then you can run the code step by step, manually or experiment yourself with the database connection.
