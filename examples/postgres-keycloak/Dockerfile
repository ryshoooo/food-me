FROM python:3.11-slim

RUN apt-get update -y && \
    apt-get install -y unixodbc-dev odbc-postgresql libpq-dev gcc && \
    apt-get clean && \
    pip install poetry

WORKDIR /example
COPY . .
RUN poetry install
