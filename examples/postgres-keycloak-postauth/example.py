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
        "clientId": "pg-access",
        "secret": "secret",
        "clientAuthenticatorType": "client-secret",
        "directAccessGrantsEnabled": True,
    },
    skip_exists=True,
)
admin.create_user(
    {
        "username": "admin",
        "email": "admin@example.com",
        "enabled": True,
        "credentials": [{"type": "password", "value": "admin"}],
    },
    exist_ok=True,
)
admin.create_user(
    {
        "username": "basic",
        "email": "basic@example.com",
        "enabled": True,
        "credentials": [{"type": "password", "value": "basic"}],
    },
    exist_ok=True,
)
try:
    admin.add_mapper_to_client(
        client_id=admin.get_client_id("pg-access"),
        payload={
            "config": {
                "access.token.claim": "false",
                "claim.name": "groups",
                "full.path": "false",
                "id.token.claim": "false",
                "userinfo.token.claim": "true",
            },
            "name": "groups",
            "protocol": "openid-connect",
            "protocolMapper": "oidc-group-membership-mapper",
        },
    )
except Exception:
    pass

try:
    admin.create_group(payload={"name": "pgadmin"})
except Exception:
    pass

# Get Keycloak token
oid = keycloak.KeycloakOpenID(
    server_url="http://localhost:8080",
    client_id="pg-access",
    client_secret_key="secret",
    realm_name="master",
)
tokens = oid.token(username="admin", password="admin")

# Connect as admin user
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

# is superuser query
is_superuser_sql = """
select current_user, usesuper from pg_user where usename = CURRENT_USER;
 """

# Execute a query
cur = conn.cursor()
cur.execute(is_superuser_sql)
print(cur.fetchone())

# Connect as basic user
tokens = oid.token(username="basic", password="basic")
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
cur.execute(is_superuser_sql)
print(cur.fetchone())

# Make basic user a superuser
admin.group_user_add(
    user_id=admin.get_user_id("basic"), group_id=admin.get_group_by_path("pgadmin")["id"]
)

# Reconnect, should be superuser
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

cur = conn.cursor()
cur.execute(is_superuser_sql)
print(cur.fetchone())

# Revoke superuser
admin.group_user_remove(
    user_id=admin.get_user_id("basic"), group_id=admin.get_group_by_path("pgadmin")["id"]
)
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

cur = conn.cursor()
cur.execute(is_superuser_sql)
print(cur.fetchone())
