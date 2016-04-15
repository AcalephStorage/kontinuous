![Kontinuous](docs/logo/logo-small.png)

Kontinuous - The Kubernetes Continuous Integration & Delivery Platform
==========

Are you sick of having to deal with Jenkins and its plugins? Getting a headache from trying to get your builds working in Kubernetes? *Kontinous is here to save the day!*

Kontinuous is a Continuous Integration & Delivery pipeline tool built specifically for Kubernetes. It aims to provide a platform for building and deploying applications using native Kubernetes Jobs and Pods.

> This is a **Work In Progress** designed to gather feedback from the community so has fairly basic functionality. Please file Issues (or better yet PRs!) so we can build the :ok_hand: CI/CD platform for K8s


## Features

Kontinuous currently offers the following features:

 - A simple Kubernetes-like spec for declaring delivery pipelines
 - Flexible stages - Command execution, Docker builds and publishing to local or remote registries
 - Integration with Github for builds and status
 - Slack notifications
 - A CLI tool for querying pipelines, build status and logs

We've got lots more planned, see the [Roadmap](#roadmap) or Github issues to get in on the action!

## Running Kontinuous

### Getting Started

A sample yaml file for running kontinuous and its dependencies in Kubernetes can be found [here](./k8s-spec.yml.example). See below for how to configure secrets.

### Dependencies

Running kontinuous requires the following to be setup:

 - **etcd**
 
 	`etcd` is used as a backend for storing pipeline and build details. This is a dedicated instance to avoid poluting with the Kubernetes etcd cluster.
 	
 - **minio**

	`minio` is used to store the logs and artifacts. S3 could also be used as it is compatible with `minio`, although this has not been tested yet.
	
- **docker registry**

	`registry` is used to store internal docker images.
	
- **kubernetes**

	Kontinuous uses Kubernetes Jobs heavily so will require at least version 1.1 with Jobs enabled


### Running in Kubernetes

Kontinuous is meant to run inside a kubernetes cluster, preferrably by a Deployment or Replication Controller.

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
    notif:
      - type: slack
        secret: notifcreds
        metadata:
          url: slackurl
          username: slackuser
          channel: slackchannel
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

#### Notification

- `type` can be: `slack`. 
- `secret` is optional. It is the secret name of the secret for notificaitions. Secret data key will be used by `metadata` as value. Recommended for public repositories.
- `metadata` is a map of values needed for certain notification type. By default, metadata value will be used. If `secret` is defined, the metadata value will be the secret data key. 

`metadata` of notification `type=slack` has the following keys:

- `url` is a slack incoming messages webhook url
- `channel` is optional. If set, it will override default channel
- `username` is optional. If set, it will override default username
 	
In the future releases, kontinuous notification will support other notification services. e.g. email, hipchat, etc.

#### Stages

`docker_build` builds a Docker Image and pushes the images to a internal registry. It can work without additional params. By default, it uses the `Dockerfile` inside  the repository root. Optional params are:

 - `dockerfile_path` - the path where the Dockerfile is located
 - `dockerfile_name` - the file name of the Dockerfile

After a build, the image is stored inside the internal docker registry.

`docker_publish` pushes the previously build Docker image to an external registry. It requires the following params:

 - `external_registry` - the external registry name (eg. quay.io)
 - `external_image_name` - the name of the image (eg. acaleph/kontinuous)

Optional params:

 - `require_crendentials` - defaults to `false`. Set to `true` if registry requires authentication
 - `username` - the username. This should be a **key** from one of the secrets file defined
 - `password` - the password. This should be a **key** from one of the secrets file defined
 - `email` - the email. This should be a **key** from one of the secrets file 

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

## Roadmap

- [ ] More stage types - wait/approvals, vulnerability/security testing, container slimming, load testing, deploy tools (Helm, DM, KPM, etc)
- [ ] Full stack tests - Spin up full environments for testing
- [ ] Advanced branch testing - Review/Sandbox environments
- [ ] Metrics - Compare build performance
- [ ] Notification service integration - Email, hipchat, etc
- [ ] Web based management Dashboard