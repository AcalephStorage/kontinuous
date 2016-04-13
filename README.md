![Kontinuous](docs/logo/logo-small.png)

Kontinuous
==========

Kontinuous is a Continuous Integration & Delivery pipeline tool built specifically for Kubernetes. It aims to provide a platform for building applications using native Kubernetes Jobs and Pods. 

## Running Kontinuous

### Dependencies

Running kontinuous requires the following to be setup:

 - **etcd**
 
 	`etcd` is used as a backend for storing pipeline and build details. This is a dedicated instance to avoid issues with the Kubernetes etcd cluster.
 	
 - **minio**

	`minio` is used to store the logs and artifacts. S3 could also be used as it is compatible with `minio`, although this has not been tested yet.
	
- **docker registry**

	`registry` is used to store internal docker images. 

### Running in Kubernetes

Kontinuous is meant to run inside a kubernetes cluster, preferrably by a Replication Controller.

The docker image can be found here: [quay.io/acaleph/kontinuous](https://quay.io/acaleph/kontinuous)

The following environment variables needs to be defined:

| Environment Variable | Description                             | Example                |
|----------------------|-----------------------------------------|------------------------|
| KV_ADDRESS           | The etcd address                        | etcd:2379              |
| S3_URL               | The minio address                       | http://minio:9000      |
| KONTINUOUS_URL       | The address where kontinuous is running | http://kontinuous:3005 |
| INTERNAL_REGISTRY    | The internal registry address           | internal-registry:5000 |

A Kubernetes Secret also needs to be defined and mounted to the Pod. The secret should have a key named `kontinuous-secrets` and should contain the following data (must be base64 encoded):

```
{
  "AuthSecret": "base64 encoded auth secret",
  "S3SecretKey": "s3 secret key",
  "S3AccessKey": "s3 access key"
}
```

`AuthSecret` is the secret for authenticating requests. This is needed by the clients to communicate with kontinuous through JWT.

`S3SecretKey` and `S3AccessKey` are the keys needed to access minio (or S3).

The secret needs to be mounted to the Pod to the path `/.secret`.

A sample yaml file for running kontinuous can be found [here](./k8s-spec.yml.example).

## Using Kontinuous

### Preparing the repository

#### Pipeline Spec

The repository needs to define a build pipeline in the repository root called `.pipeline.yml`

Here's a sample `.pipeline.yml`:

```
---
apiVersion: v1alpha1
kind: Pipeline
metadata:
  name: kontinuous
  namespace: acaleph
spec:
  selector:
    matchLabels:
      app: kontinuous
      type: ci-cd
  template:
    metadata:
      name: kontinuous
      labels:
        app: kontinuous
        type: ci-cd
    stages:
      - name: Build Docker Image
        type: docker_build
      - name: Unit Test
        type: command
        params:
            command:
                - make
                - test
      - name: Publish to Quay
        type: docker_publish
        params:
          external_registry: quay.io
          external_image_name: acaleph/kontinuous
          require_credentials: "TRUE"
          username: user        # taken from secret
          password: password    # taken from secret
          email: email          # taken from secret
        secrets:
          - docker-credentials
```

The format is something similar to K8s Specs. Here are more details on some of the fields:

 - `namespace` - the namespace to run the build
 - `matchLabels`/`labels` - the labels that are used for building the job
 - `stages` - defines the stages of the build pipeline

The general definition of a stage is:

```
name: Friendly name
type: {docker_build,command,docker_publish}
params:
  key: value
secrets:
  - secret-name
```

- `type` can be: `docker_build`, `docker_publish`, or `command`.
- `params` is a map of parameters to be loaded as environment variables. 
- `secrets` is a list of secrets that will be used as values for `params`.

#### Stages

`docker_build` can work without additional params. By default, it uses the `Dockerfile` inside  the repository root. Optional params are:

 - `dockerfile_path` - the path where the Dockerfile is located
 - `dockerfile_name` - the file name of the Dockerfile

After a build, the image is stored inside the internal docker registry.

`docker_publish` requires the following params:

 - `external_registry` - the external registry name (eg. quay.io)
 - `external_image_name` - the name of the image (eg. acaleph/kontinuous)

Optional params:

 - `require_crendentials` - defaults to `false`. Set to `true` if registry requires authentication
 - `username` - the username. this should be a key from one of the secrets file defined
 - `password` - the password. this should be a key from one of the secrets file defined
 - `email` - the email. this should be a key from one of the secrets file 

The image that will be pushed is the image that was previously built. This does not work for now if no image was created. 

`command` runs a command on the newly create docker image or on the image specified. Required param is `command` which is a list of string defining the command to execute.

Optional params are:

 - `args` - a list of string to serve as the arguments for the command
 - `image` - the image to run the commands in. If not specified, the previous built image will be used.


### Authentication

#### Github Token

Currently, only Github Repositories are supported. A github token needs to be generated in order to access the repositories. 

To generate a github token, follow this [link](https://github.com/settings/tokens/new).

Make sure to enable access to the following:

 - repo
 - admin:repo_hook
 - user


#### JSON Web Token

Kontinuous uses JWT for authentication. To create a token, the `AuthSecret` (from kontinuous-secret) and the github token is required. One way of generating the token is using [jwt.io](https://jwt.io).

The header should be:

```
{
  "alg": "HS256",
  "typ": "JWT"
}
```

Payload:

```
{
  "identities": [
    {
      "access_token": "github token"
    }
  ]
}
```

and Signature:

```
HMACSHA256(
  base64UrlEncode(header) + "." +
  base64UrlEncode(payload),  
  AuthSecret
)

[x]secret base64 encoded
```

Once a token is generated, this can be added to the request header as `Authorization: Bearer {token}` to authenticate requests.

Alternatively the CLI Client can manage the token generation

## API

kontinuous is accessible from it's API. The API docs can be viewed via Swagger.

The API doc can be accessed via `{kontinuous-address}/apidocs`

## Clients

At the moment, there is a basic cli client [here](https://github.com/AcalephStorage/kontinuous/tree/develop/cli). A Web based Dashboard is under development.

## Development

Building `kontinuous` from source is done by:

```
$ make deps build
```

Build the docker image:

```
$ docker build -t {tag} .
```

## Notes

This is a Work In Progress designed to gather feedback from the community and has very basic functionality. Please file Issues (or better yet PRs!) so we can build the :ok_hand: CI/CD platform for K8s
