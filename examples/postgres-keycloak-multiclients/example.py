import keycloak
import psycopg2
import requests

# Get Keycloak token
oid = keycloak.KeycloakOpenID(
    server_url="http://localhost:8080", client_id="admin-cli", realm_name="master"
)
tokens = oid.token(username="admin", password="admin")

# Connect to the postgres database
resp = requests.post(
    "http://localhost:10000/connection",
    json={"access_token": tokens["access_token"], "refresh_token": tokens["refresh_token"]},
).json()
user = resp["username"]
dsn = f"dbname=postgres host=localhost port=5432 user={user}"
conn = psycopg2.connect(dsn)
conn.autocommit = True

c = conn.cursor()
c.execute("DROP DATABASE IF EXISTS test")
c.execute("CREATE DATABASE test")

# Create new client for the database
admin = keycloak.KeycloakAdmin(
    server_url="http://localhost:8080", username="admin", password="admin"
)
admin.create_client(
    {
        "enabled": True,
        "clientId": "my-client",
        "secret": "my-client-secret",
        "clientAuthenticatorType": "client-secret",
        "directAccessGrantsEnabled": True,
    },
    skip_exists=True,
)
admin.create_user(
    {
        "username": "dummy",
        "email": "dummy@example.com",
        "enabled": True,
        "credentials": [{"type": "password", "value": "dummy"}],
    },
    exist_ok=True,
)

# Get Keycloak token for the new client and user
oid = keycloak.KeycloakOpenID(
    server_url="http://localhost:8080",
    client_id="my-client",
    realm_name="master",
    client_secret_key="my-client-secret",
)
tokens = oid.token(username="dummy", password="dummy")

# Connect to the database with the new client
resp = requests.post(
    "http://localhost:10000/connection",
    json={"access_token": tokens["access_token"], "refresh_token": tokens["refresh_token"]},
).json()
user = resp["username"]

# Should fail to connect to the postgres database
dsn = f"dbname=postgres host=localhost port=5432 user={user}"
conn = psycopg2.connect(dsn)

# Should succeed to the test database
dsn = f"dbname=test host=localhost port=5432 user={user}"
conn = psycopg2.connect(dsn)
