Kontinuous API
==============

Kontinuous provides an API to access pipeline and build details. 

## Swagger

The API doc can be accessed from a running kontinuous instance via `{kontinuous-address}/apidocs`. 

## Authentication

Kontinuous API uses JSON Web Tokens for authentication. This is useful when accessing the API directly. There are several ways of getting a token. 

The token needs to be added to the header as `Authorization: Bearer {token}` to authenticate requests.

### Github

This is the method used by Kontinuous UI. This is a slightly modified [Github's OAuth Web Application Flow](https://developer.github.com/v3/oauth/#web-application-flow).

1. Kontinuous needs to be registered as a [Github OAuth Application](https://github.com/settings/applications/new). The Client ID and Secret needs to be defined in the kontinuous secret.

2. Redirect users to request Github Access (step 1 in web application flow):

```
GET https://github.com/login/oauth/authorize?client_id={clientid}&redirect_uri={redirect_url}& scope=user,repo,admin:repo_hook&state={random string}
```

3. Send authorization code to Kontinuous:

```
POST {kontinuous-url}/api/v1/login/github?code={auth_code}&state={state_from_step_1}
```

This will return a JSON Web Token that can be used to access the API.

### Auth0

[Auth0](https://auth0.com) is a service for managing authentications. This can be used to generate an auth secret and provide Github access for kontinuous. 

1. Use Auth0 to create an auth secret to be added to Kontinuous.
2. The Web Interface needs to authenticate against Auth0 to use Kontinuous.

Auth0 will provide the JSON Web Token.

### JWT Creation

A JSON Web Token can be manually created. This requires a github token and the auth secret used by Kontinuous.

To generate a github token, follow this [link](https://github.com/settings/tokens/new).

Make sure to enable access to the following:

 - repo
 - admin:repo_hook
 - user

The script `scripts/jwt-gen` can generate a JSON Web Token to be used for authentication with Kontinuous. 

```console
$ scripts/jwt-gen --secret {secret} --github-token {github-token}
```
The generated token's validity can be verified at [jwt.io](https://jwt.io).



