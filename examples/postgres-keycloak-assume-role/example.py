import logging
import sys

import keycloak
import psycopg2
import pyodbc
import requests

USE_ODBC = sys.argv[1] == "odbc"

# Create postgres user and client
admin = keycloak.KeycloakAdmin(
    server_url="http://localhost:8080", username="admin", password="admin"
)
admin.create_client(
    {
        "enabled": True,
        "clientId": "pg-superadmin",
        "secret": "secret",
        "clientAuthenticatorType": "client-secret",
        "directAccessGrantsEnabled": True,
    },
    skip_exists=True,
)
admin.create_user(
    {
        "username": "postgres",
        "email": "admin@example.com",
        "enabled": True,
        "credentials": [{"type": "password", "value": "admin"}],
    },
    exist_ok=True,
)

# Get Keycloak token
oid = keycloak.KeycloakOpenID(
    server_url="http://localhost:8080",
    client_id="pg-superadmin",
    client_secret_key="secret",
    realm_name="master",
)
tokens = oid.token(username="postgres", password="admin")

# Connect as postgres user
resp = requests.post(
    "http://localhost:10000/connection",
    json={"access_token": tokens["access_token"], "refresh_token": tokens["refresh_token"]},
).json()
user = resp["username"]

if USE_ODBC:
    dsn = (
        "DRIVER={PostgreSQL Unicode};"
        "DATABASE=postgres;"
        f"UID={user};"
        "SERVER=localhost;"
        "PORT=5432;"
    )
    conn = pyodbc.connect(dsn)
else:
    dsn = f"dbname=postgres host=localhost port=5432 user={user}"
    conn = psycopg2.connect(dsn)

# Execute a query
cur = conn.cursor()
cur.execute("SELECT current_user;")
print(cur.fetchone())

# Create new user
admin.create_user(
    {
        "username": "michael",
        "email": "michael@example.com",
        "enabled": True,
        "credentials": [{"type": "password", "value": "michael"}],
    },
    exist_ok=True,
)
create_user_sql = """
DO
$do$
BEGIN
   IF EXISTS (
      SELECT FROM pg_catalog.pg_roles
      WHERE  rolname = 'michael') THEN

      RAISE NOTICE 'Role "michael" already exists. Skipping.';
   ELSE
      CREATE USER michael;
   END IF;
END
$do$;
"""
cur.execute(create_user_sql)
conn.commit()
conn.close()

# Connect as michael user
tokens = oid.token(username="michael", password="michael")
resp = requests.post(
    "http://localhost:10000/connection",
    json={"access_token": tokens["access_token"], "refresh_token": tokens["refresh_token"]},
).json()
user = resp["username"]

if USE_ODBC:
    dsn = (
        "DRIVER={PostgreSQL Unicode};"
        "DATABASE=postgres;"
        f"UID={user};"
        "SERVER=localhost;"
        "PORT=5432;"
    )
    conn = pyodbc.connect(dsn)
else:
    dsn = f"dbname=postgres host=localhost port=5432 user={user}"
    conn = psycopg2.connect(dsn)

# Execute a query
cur = conn.cursor()
cur.execute("SELECT current_user;")
print(cur.fetchone())

# Can I escape it?
try:
    cur.execute("RESET SESSION AUTHORIZATION")
except Exception:
    logging.exception("Expected not allowed escape")

try:
    cur.execute("SET SESSION AUTHORIZATION postgres")
except Exception:
    logging.exception("Expected not allowed escape")

cur.execute("SELECT current_user;")
print(cur.fetchone())
