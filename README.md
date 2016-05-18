![Kontinuous](docs/logo/logo-small.png)

Kontinuous - The Kubernetes Continuous Integration & Delivery Platform
==========

Are you sick of having to deal with Jenkins and its plugins? Getting a headache from trying to get your builds working in Kubernetes? *Kontinuous is here to save the day!*

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

The script `scripts/kontinuous-deploy` is a quick way of running `kontinuous` in a K8s cluster. The general syntax is:

```
$ kontinuous-deploy --namespace {k8s-namespace} --auth-secret {base64url encoded secret} --s3-access-key {s3 access key} --s3-secret-key {s3 secret key}
```

This will launch `kontinuous` via the locally configured `kubectl` in the given namespace together with `etcd`, `minio`, and a docker `registry`. This expects that the kubernetes cluster supports the LoadBalancer service.

Alternatively, for more customization, a sample yaml file for running kontinuous and its dependencies in Kubernetes can be found [here](./k8s-spec.yml.example). See [below](#running-in-kubernetes) for how to configure secrets.

Once running, add a [.pipeline.yml](#pipeline-spec) to the root of your Github repo and configure the webhooks.

Example pipelines can be found in [/examples](./examples)

The [CLI client](#clients) or [API](#api) can be used to view build status or logs.

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

```json
{
  "AuthSecret": "base64 encoded auth secret",
  "S3SecretKey": "s3 secret key",
  "S3AccessKey": "s3 access key",
  "GithubClientID": "github client ID",
  "GithubClientSecret": "github client secret"
}
```

`AuthSecret` is the secret for authenticating requests. This is needed by the clients to communicate with kontinuous through JWT.

`S3SecretKey` and `S3AccessKey` are the keys needed to access minio (or S3).

`GithubClientID` and `GithubClientSecret` are optional and only required for Github only authentication. (See below)

The secret needs to be mounted to the Pod to the path `/.secret`.

## Using Kontinuous

### Preparing the repository

#### Pipeline Spec

The repository needs to define a build pipeline in the repository root called `.pipeline.yml`

Here's a sample `.pipeline.yml`:

```yaml
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
        metadata:
          url: slackurl             #taken from secret
          username: slackuser       #taken from secret  
          channel: slackchannel     #taken from secret
    secrets:
      - notifcreds
      - docker-credentials
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
```

The format is something similar to K8s Specs. Here are more details on some of the fields:

 - `namespace` - the namespace to run the build
 - `matchLabels`/`labels` - the labels that are used for building the job
 - `stages` - defines the stages of the build pipeline

The general definition of a stage is:

```yaml
name: Friendly name
type: {docker_build,command,docker_publish}
params:
  key: value
```

- `type` can be: `docker_build`, `docker_publish`, `command`, or `deploy`.
- `params` is a map of parameters to be loaded as environment variables. 

#### Notification

- `type` can be: `slack`. 
- `metadata` is a map of values needed for a certain notification type. The metadata value should be a **key** from one of the secrets file defined

 `metadata` of notification `type=slack` has the following keys:
 
  - `url` is a slack incoming messages webhook url.
  - `channel` is optional. If set, it will override default channel
  - `username` is optional. If set, it will override default username
 	
In the future releases, kontinuous notification will support other notification services. e.g. email, hipchat, etc.

### Secrets

- `secrets` is a list of secrets that will be used as values for stages and notification.

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

Required secrets:

 - `dockeruser`
 - `dockerpass`
 - `dockeremail`

	These secrets needs to be defined in at least one of the secrets provided. 

The image that will be pushed is the image that was previously built. This does not work for now if no image was created. 

`command` runs a command on the newly create docker image or on the image specified. Required param is `command` which is a list of string defining the command to execute.

Optional params are:

 - `args` - a list of string to serve as the arguments for the command
 - `image` - the image to run the commands in. If not specified, the previous built image will be used.
 - `dependencies` - a list of Kubernetes spec files to run as dependencies for running the command. Useful when running integration tests.

`deploy` deploys a Kubernetes Spec file (yaml) to kubernetes. 


### Authentication

There are currently two ways of authenticating with kontinuous. One is by using Github's OAuth Web Application Flow and the other one is with JSON Web Tokens.

#### Github OAuth

This is the authorization process used by `kontinuous-ui`. This is a 3 step process detailed [here](https://developer.github.com/v3/oauth/#web-application-flow) with a slightly different variation:

1. Kontinuous needs to be registered as a Github OAuth Application [here](https://github.com/settings/applications/new). 

2. Redirect users to request Github Access (step 1 in web application flow):

```
GET https://github.com/login/oauth/authorize
```

3. Send authorization code to Kontinuous:

```
POST {kontinuous-url}/api/v1/login/github?code={auth_code}&state={state_from_step_1}
```

This should return a JSON Web Token in the body that can be used to authenticate further requests.


#### JSON Web Token

Currently, only Github Repositories are supported. A github token needs to be generated in order to access the repositories. 

To generate a github token, follow this [link](https://github.com/settings/tokens/new).

Make sure to enable access to the following:

 - repo
 - admin:repo_hook
 - user

The script `scripts/jwt-gen` can generate a JSON Web Token to be used for authentication with Kontinuous. 

```console
$ scripts/jwt-gen --secret {base64url encoded secret} --github-token {github-token}
```

This generates a JSON Web Token and can be added to the request header as `Authorization: Bearer {token}` to authenticate requests.

The generated token's validity can be verified at [jwt.io](https://jwt.io).

## API

kontinuous is accessible from it's API and docs can be viewed via Swagger.

The API doc can be accessed via `{kontinuous-address}/apidocs`

## Clients

At the moment, there is a basic cli client binary [here](https://github.com/AcalephStorage/kontinuous/releases) and code available [here](https://github.com/AcalephStorage/kontinuous/tree/develop/cli).

A Web based Dashboard is under development.

## Development

Building `kontinuous` from source is done by:

```console
$ make deps build
```

Build the docker image:

```console
$ docker build -t {tag} .
```

## Roadmap

- [ ] More stage types - wait/approvals, vulnerability/security testing, container slimming, load testing, deploy tools (Helm, DM, KPM, etc)
- [ ] Full stack tests - Spin up full environments for testing
- [ ] Advanced branch testing - Review/Sandbox environments
- [ ] Metrics - Compare build performance
- [ ] Notification service integration - Email, hipchat, etc
- [ ] Web based management Dashboard
