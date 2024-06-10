# FOOD-Me

_**F**inally a <ins>G**OO**D</ins> **D**atabase **M**iddlewar**e**_

Nom nom nom :P

A simple generic database middleware, that supports OIDC as an authentication method and allows for deep access control.

## Introduction

OIDC/OAuth2 is becoming the industry standard for handling authentication as well as authorization. However, when it comes to database communications, most databases lack OIDC as the authorization mechanism, and for a good reason! OIDC was meant to be the authentication mechanism for WebApps, websites that allow you to log in with your favorite social media provider instead of creating a new username and password that you'll definitely forget unless put into the password manager. And, the authentication flow performs website redirects, which don't really work well with direct TCP database connections.

But, the world of cloud computing is moving forward, and instead of having a beefy laptop/PC, people and nerds (developers) started using cloud computing and desktops in their browsers instead of their own devices. And, most of these cloud desktop platforms actually use OIDC to log you in, meaning you have potentially access to your bearer token very natively.

Also if you don't, direct access grants do exist in the OIDC world, so you can get your tokens directly as well. But that's a bit boring...

Anyway, long story short, this project is offering you a way to use the bearer tokens to authenticate with your favorite database, without actually changing the database itself or installing any plugins into the database etc. Instead, we have a proxy that lives just in front of the database, handles the bearer tokens gracefully and forwards your TCP packets to the database and back if authenticated successfully.

Pretty neat and yummy. Right?

## Features

- OIDC authentication and authorization
- User impersonation in the session
- Most common drivers and databases (wishful thinking, needs work ¯\_(ツ)\_/¯)
- OPA integration (wishful thinking, needs work ¯\_(ツ)\_/¯)

## How does it work?

The basic flow is (from the client's perspective):

1. Retrieve access and refresh tokens from your OIDC provider
2. Make a connection request to your database via FOOD-Me proxy with the access and refresh tokens
3. The proxy verifies the tokens, if everything is A-OK, the proxy will authenticate to the database as the configured\* user
4. Afterward, any packets received from the client are linked to the bearer tokens. If a new packet arrives from the client, the proxy will check the validity of the access token. If it is invalid, the proxy will try to refresh it using the refresh token. If it fails, the packet is then denied and the connection gets terminated, otherwise the packets are proxied to the database and the response back to the client.

\*The configured user is the username+password combination given to the proxy as a configuration value. This should be a superuser capable of doing all the possible damage in the database that's possible.

But hold on, that sounds extremely insecure! So basically I can only use the FOOD-Me middleware to perform an OIDC method to authenticate as a superuser in the database? What good is that for?

Well, it is good for some things, but yeah otherwise it sucks. Especially if all your users exist as in the database with their complicated permissions schemes. Don't worry though, the proxy offers you a way out as well!

You can decide for yourself what the proxy should do upon successful authentication. There are essentially 3 options:

1. Continue the connection as the superuser
2. Based on the UserInfo data, pick up the username from the OIDC claims and assume the user for the connection session
3. Continue as the superuser, but use OPA for handling access and modifying the input queries

That's right, FOOD-Me does more than you'd initially think!

### How do I send access and refresh tokens to the proxy?

There are 2 methods to do this:

1. Directly in the DSN as a user specification. Instead of the `username` and `password` fields, you can just omit the `password` field and specify the `username` as `username=access_token=${my_access_token};refresh_token=${my_refresh_token}`. The proxy will automatically parse this and use it to fetch the OIDC identity.
2. Use the proxy RestAPI endpoint. The main problem with the direct DSN entry is that many common drivers (such as ODBC) restrict the length of the username to 255 characters. That's not enough to send long JWT tokens. The proxy thus offers you to set these via a RestAPI endpoint `POST :${API_PORT}/connection`, which expects you to send the access and refresh tokens, and in return will give you a unique `username` to be used in the DSN connection. A simple Python example (assuming `localhost` for simplicity)

   ```python
   username = requests.post("http://localhost:10000/connection", json={"access_token": "ACCESS", "refresh_token": "REFRESH"}).json()["username"]
   dsn = f"host=localhost port=2099 user={username} database=test"
   ```

### How do I configure the OIDC client?

Simple really. This is just a configuration option in the proxy when you start it up. See the [full list of all configuration options](#configuration-options).

However, there is an option to have multiple clients configured for a single database! Usually, a single database does not consist of a single database (sounds weird, but it's true). This is also why you specify the `database` field in your DSN, you are also specifying which database you want to connect to. Well, FOOD-Me allows you to define different OIDC clients for different databases. This way you can control who has access to which database in your OIDC provider instead!

# Technical specification

Jokes aside, let's get into some nitty-gritty boring nerd stuff.

## Supported database

- [x] Postgres
- [ ] Microsoft SQL Server
- [ ] MySQL/MariaDB

## Configuration options

| Name                                      | Description                                                                                               | CLI                                     | Environment variable                  | Options & types                         |
| ----------------------------------------- | --------------------------------------------------------------------------------------------------------- | --------------------------------------- | ------------------------------------- | --------------------------------------- |
| Log Level                                 | Logging level                                                                                             | --log-level                             | LOG_LEVEL                             | trace,debug,info,warn,error,fatal,panic |
| Log Format                                | Log formatting                                                                                            | --log-format                            | LOG_FORMAT                            | text,json,pretty                        |
| Destination Host                          | The database destination hostname                                                                         | --destination-host                      | DESTINATION_HOST                      | string                                  |
| Destination Port                          | The database destination port number                                                                      | --destination-port                      | DESTINATION_PORT                      | number                                  |
| Destination Type                          | The database type                                                                                         | --destination-database-type             | DESTINATION_TYPE                      | postgres                                |
| Destination Username                      | The superuser username                                                                                    | --destination-username                  | DESTINATION_USERNAME                  | string                                  |
| Destination Password                      | The superuser password                                                                                    | --destination-password                  | DESTINATION_PASSWORD                  | string                                  |
| Destination Log Upstream                  | Flag whether to perform a debug log of all packets coming from the destination datbase                    | --destination-log-upstream              | DESTINATION_LOG_UPSTREAM              | boolean                                 |
| Destination Log Downstream                | Flag whether to perform a debug log of all packets coming from the client                                 | --destination-log-downstream            | DESTINATION_LOG_DOWNSTREAM            | boolean                                 |
| OIDC Enabled                              | Flag specifying whether OIDC verification is enabled                                                      | --oidc-enabled                          | OIDC_ENABLED                          | boolean                                 |
| OIDC Client ID                            | The global OIDC client ID                                                                                 | --oidc-client-id                        | OIDC_CLIENT_ID                        | string                                  |
| OIDC Client Secret                        | The global OIDC client secret                                                                             | --oidc-client-secret                    | OIDC_CLIENT_SECRET                    | string                                  |
| OIDC Token URL                            | URL for the token endpoint                                                                                | --oidc-token-url                        | OIDC_TOKEN_URL                        | URL                                     |
| OIDC UserInfo URL                         | URL for the userinfo endpoint                                                                             | --oidc-user-info-url                    | OIDC_USER_INFO_URL                    | URL                                     |
| OIDC Database Client ID Mapping           | A mapping between the database names and Client IDs                                                       | --oidc-database-client-id               | OIDC_DATABASE_CLIENT_ID               | key1=value1,key2=value2                 |
| OIDC Database Client Secret Mapping       | A mapping between the database names and Client secrets                                                   | --oidc-database-client-secret           | OIDC_DATABASE_CLIENT_SECRET           | key1=value1,key2=value2                 |
| OIDC Database Fallback to the Base Client | Flag whether to fallback on the global client ID in case there is no match in the database client mapping | --oidc-database-fallback-to-base-client | OIDC_DATABASE_FALLBACK_TO_BASE_CLIENT | boolean                                 |
| Port                                      | Port where the proxy is started (default 2099)                                                            | --port                                  | PORT                                  | number                                  |
| API Port                                  | Port where the proxy will serve the RestAPI                                                               | --api-port                              | API_PORT                              | number                                  |

TODO:

- Add examples
- Also, code this shit up
