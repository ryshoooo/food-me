import keycloak
import psycopg2
import pyodbc
import requests

# Get Keycloak token
oid = keycloak.KeycloakOpenID(
    server_url="http://localhost:8080", client_id="admin-cli", realm_name="master"
)
tokens = oid.token(username="admin", password="admin")

# 1. Direct connection with libpq
user = f"access_token={tokens['access_token']};refresh_token={tokens['refresh_token']}"
dsn = f"dbname=postgres host=localhost port=5432 user={user}"
conn = psycopg2.connect(dsn)

# Execute a query
cur = conn.cursor()
cur.execute("SELECT current_user;")
print(cur.fetchone())

# 2. Create connection using the API
resp = requests.post(
    "https://foodme:10000/connection",
    json={"access_token": tokens["access_token"], "refresh_token": tokens["refresh_token"]},
    verify="foodme.crt",
).json()
user = resp["username"]
dsn = f"dbname=postgres host=localhost port=5432 user={user}"
conn = psycopg2.connect(dsn)
conn.autocommit = True

# Execute a query
cur = conn.cursor()
cur.execute("SELECT current_user;")
print(cur.fetchone())

# 3. Using PyODBC - Only works with the connection API
# Pyodbc
conn_str = (
    "DRIVER={PostgreSQL Unicode};"
    "DATABASE=postgres;"
    f"UID={user};"
    "SERVER=localhost;"
    "PORT=5432;"
    "SSLMode=prefer;"
)
cnxn = pyodbc.connect(conn_str)

# Execute a query
c = cnxn.cursor()
c.execute("SELECT current_user;")
print(c.fetchone())
