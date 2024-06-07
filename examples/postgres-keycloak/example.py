import keycloak
import psycopg2

# Get Keycloak token
oid = keycloak.KeycloakOpenID(
    server_url="http://localhost:8080", client_id="admin-cli", realm_name="master"
)
tokens = oid.token(username="admin", password="admin")

# Connect to PostgreSQL
dsn = f"dbname=postgres host=localhost port=5432 user=access_token={tokens['access_token']};refresh_token={tokens['refresh_token']}"
conn = psycopg2.connect(dsn)

# Exectute a query
cur = conn.cursor()
cur.execute("SELECT current_user;")
print(cur.fetchone())
cur.execute("SELECT tablename FROM pg_catalog.pg_tables")
print(cur.fetchall())
