import keycloak
import sqlalchemy
import requests
from sqlalchemy.orm import declarative_base, sessionmaker

# Create client
admin = keycloak.KeycloakAdmin(
    server_url="http://localhost:8080", username="admin", password="admin"
)
admin.create_client(
    {
        "enabled": True,
        "clientId": "pgpets",
        "secret": "petsarenice",
        "clientAuthenticatorType": "client-secret",
        "directAccessGrantsEnabled": True,
    },
    skip_exists=True,
)

# Create groups
try:
    admin.create_group(payload={"name": "alpha"})
except Exception:
    pass

try:
    admin.create_group(payload={"name": "admin"})
except Exception:
    pass

try:
    admin.create_group(payload={"name": "killer"})
except Exception:
    pass

# Create users
admin.create_user(
    {
        "username": "michael",
        "email": "michael@example.com",
        "enabled": True,
        "credentials": [{"type": "password", "value": "michael"}],
    },
    exist_ok=True,
)
admin.create_user(
    {
        "username": "richard",
        "email": "richard@example.com",
        "enabled": True,
        "credentials": [{"type": "password", "value": "richard"}],
    },
    exist_ok=True,
)


# Add groups to userinfo
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


# Get Keycloak token
oid = keycloak.KeycloakOpenID(
    server_url="http://localhost:8080",
    client_id="pgpets",
    realm_name="master",
    client_secret_key="petsarenice",
)
tokens = oid.token(username="admin", password="admin")

# Create connection using the API
resp = requests.post(
    "http://localhost:10000/connection",
    json={"access_token": tokens["access_token"], "refresh_token": tokens["refresh_token"]},
).json()
user = resp["username"]

engine = sqlalchemy.create_engine(f"postgresql+psycopg2://{user}@localhost:5432/postgres")

# Define tables
Base = declarative_base()


class Pets(Base):
    __tablename__ = "pets"
    id = sqlalchemy.Column(sqlalchemy.Integer, primary_key=True)
    name = sqlalchemy.Column(sqlalchemy.String)
    owner = sqlalchemy.Column(sqlalchemy.String)
    veterinarian = sqlalchemy.Column(sqlalchemy.String)
    clinic = sqlalchemy.Column(sqlalchemy.String)
    weight = sqlalchemy.Column(sqlalchemy.Float)
    age = sqlalchemy.Column(sqlalchemy.Integer)
    hidden = sqlalchemy.Column(sqlalchemy.Boolean)
    deleted = sqlalchemy.Column(sqlalchemy.Boolean)


Base.metadata.create_all(engine)
SLocal = sessionmaker(bind=engine)
db = SLocal()

# Add a couple of pets
p1 = Pets(
    name="Rex",
    owner="michael",
    veterinarian="doctor",
    clinic="foodme",
    weight=10,
    age=2,
    hidden=False,
    deleted=False,
)
db.add(p1)
db.commit()

p2 = Pets(
    name="Snailo",
    owner="richard",
    veterinarian="doctor",
    clinic="foodme",
    weight=0.04,
    age=1,
    hidden=False,
    deleted=False,
)
db.add(p2)
db.commit()

p3 = Pets(
    name="Snaila",
    owner="richard",
    veterinarian="doctor",
    clinic="foodme",
    weight=0.03,
    age=1,
    hidden=True,
    deleted=False,
)
db.add(p3)
db.commit()

# What can I as the admin user see?
print("Admin user, no assignments")
for pet in db.query(Pets).all():
    print(pet.__dict__)

# Why? Let's investigate the query
print(
    requests.post(
        "http://localhost:10000/permissionapply",
        json={"username": user, "sql": "select * from pets"},
    ).json()
)

# # Execute a query
# cur = conn.cursor()
# cur.execute("SELECT current_user;")
# print(cur.fetchone())

# # 3. Using PyODBC - Only works with the connection API
# # Pyodbc
# conn_str = (
#     "DRIVER={PostgreSQL Unicode};"
#     "DATABASE=postgres;"
#     f"UID={user};"
#     "SERVER=localhost;"
#     "PORT=5432;"
# )
# cnxn = pyodbc.connect(conn_str)

# # Execute a query
# c = cnxn.cursor()
# c.execute("SELECT current_user;")
# print(c.fetchone())
