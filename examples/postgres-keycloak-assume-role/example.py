import keycloak
import psycopg2
import requests

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
cur.execute("CREATE USER michael;")
conn.commit()
conn.close()

# Connect as michael user
tokens = oid.token(username="michael", password="michael")

# Connect as postgres user
resp = requests.post(
    "http://localhost:10000/connection",
    json={"access_token": tokens["access_token"], "refresh_token": tokens["refresh_token"]},
).json()
user = resp["username"]
dsn = f"dbname=postgres host=localhost port=5432 user={user}"
conn = psycopg2.connect(dsn)

# Execute a query
cur = conn.cursor()
cur.execute("SELECT current_user;")
print(cur.fetchone())

# Can I escape it?
cur.execute("RESET SESSION AUTHORIZATION")
cur.execute("SELECT current_user;")
print(cur.fetchone())
