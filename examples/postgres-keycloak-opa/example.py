import keycloak
import sqlalchemy
import logging
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
        client_id=admin.get_client_id("pgpets"),
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

p4 = Pets(
    name="Bearro",
    owner="michael",
    veterinarian="doctor",
    clinic="foodme",
    weight=10,
    age=2,
    hidden=False,
    deleted=True,
)
db.add(p4)
db.commit()

# What can I as the admin user see?
print("Admin user, no assignments")
for pet in db.query(Pets).all():
    print(pet.__dict__)

# Why? Let's investigate the query
print(
    requests.post(
        "http://localhost:10000/permissionapply",
        json={"username": user, "sql": str(db.query(Pets))},
    ).json()
)

# Login as richard
tokens = oid.token(username="richard", password="richard")

# Create connection using the API
resp = requests.post(
    "http://localhost:10000/connection",
    json={"access_token": tokens["access_token"], "refresh_token": tokens["refresh_token"]},
).json()
user = resp["username"]

engine = sqlalchemy.create_engine(f"postgresql+psycopg2://{user}@localhost:5432/postgres")
SLocal = sessionmaker(bind=engine)
db = SLocal()

print("Richard user, no assignments")
for pet in db.query(Pets).all():
    print(pet.__dict__)

# Why? Let's investigate the query
print(
    requests.post(
        "http://localhost:10000/permissionapply",
        json={"username": user, "sql": str(db.query(Pets))},
    ).json()
)

# Are the filters preserved?
for pet in db.query(Pets).filter(Pets.clinic == "foodme").all():
    print(pet.__dict__)


# Let's make the admin user alpha
admin.group_user_add(
    user_id=admin.get_user_id("admin"), group_id=admin.get_group_by_path("alpha")["id"]
)

# Now should be able to see all unhidden pets
tokens = oid.token(username="admin", password="admin")
resp = requests.post(
    "http://localhost:10000/connection",
    json={"access_token": tokens["access_token"], "refresh_token": tokens["refresh_token"]},
).json()
user = resp["username"]
engine = sqlalchemy.create_engine(f"postgresql+psycopg2://{user}@localhost:5432/postgres")
SLocal = sessionmaker(bind=engine)
db = SLocal()
for pet in db.query(Pets).all():
    print(pet.__dict__)


# What if admin user is a killer?
admin.group_user_add(
    user_id=admin.get_user_id("admin"), group_id=admin.get_group_by_path("killer")["id"]
)
tokens = oid.token(username="admin", password="admin")
resp = requests.post(
    "http://localhost:10000/connection",
    json={"access_token": tokens["access_token"], "refresh_token": tokens["refresh_token"]},
).json()
user = resp["username"]
engine = sqlalchemy.create_engine(f"postgresql+psycopg2://{user}@localhost:5432/postgres")
SLocal = sessionmaker(bind=engine)
db = SLocal()

try:
    db.query(Pets).all()
except Exception:
    logging.exception("Expected failure to read pets")

# Fails as expected, but admin groups can see everything, even if they are killers. Let's make
# admin part of the admin group
admin.group_user_add(
    user_id=admin.get_user_id("admin"), group_id=admin.get_group_by_path("admin")["id"]
)
tokens = oid.token(username="admin", password="admin")
resp = requests.post(
    "http://localhost:10000/connection",
    json={"access_token": tokens["access_token"], "refresh_token": tokens["refresh_token"]},
).json()
user = resp["username"]
engine = sqlalchemy.create_engine(f"postgresql+psycopg2://{user}@localhost:5432/postgres")
SLocal = sessionmaker(bind=engine)
db = SLocal()

for pet in db.query(Pets).all():
    print(pet.__dict__)
