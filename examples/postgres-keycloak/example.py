import keycloak
import psycopg2
import pyodbc
import base64

# Get Keycloak token
oid = keycloak.KeycloakOpenID(
    server_url="http://localhost:8080", client_id="admin-cli", realm_name="master"
)
tokens = oid.token(username="admin", password="admin")

user = base64.b64encode(
    f"access_token={tokens['access_token']};refresh_token={tokens['refresh_token']}".encode()
).decode()
# Connect to PostgreSQL
dsn = f"dbname=postgres host=localhost port=5432 user={user}"
conn = psycopg2.connect(dsn)

# Execute a query
cur = conn.cursor()
cur.execute("SELECT current_user;")
print(cur.fetchone())
cur.execute("SELECT tablename FROM pg_catalog.pg_tables")
print(cur.fetchall())

# Pyodbc
conn_str = (
    "DRIVER={PostgreSQL Unicode};"
    "DATABASE=postgres;"
    f"UID={user};"
    "SERVER=localhost;"
    "PORT=5432;"
)
cnxn = pyodbc.connect(conn_str)
