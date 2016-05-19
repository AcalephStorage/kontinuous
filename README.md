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

Before running Kontinuous, it needs to be added as a github OAuth Application [here](https://github.com/settings/applications/new). The `Client ID` and `Client Secret` will be used in running Kontinuous.

The `kubernetes-cli` can bootstrap a kontinuous setup on a running Kubernetes cluster. This requires `kubectl` to be in the `PATH` and configured to access the cluster. 

```
$ kotinuous-cli --namespace {namespace} \
    --auth-secret {base64 encoded secret} \
    --github-client-id {github client id} \
    --github-client-secret {github client secret}
```

Parameters:

| parameter              | description                                                                                                                  |
|------------------------|------------------------------------------------------------------------------------------------------------------------------|
| --namespace            | The namespace to deploy Kontinuous to. This defaults to `kontinuous`                                                         |
| --auth-secret          | A base64 encoded secret. This is used by kontinuous to provide JWT for authentication. This can be any base64 encoded string |
| --github-client-id     | The Github client ID provided when registering kontinuous as a Github OAuth application                                      |
| --github-client-secret | The Github client secret provided when registering kontinuous as a Github OAuth application                                  |


This will launch `kontinuous` via the locally configured `kubectl` in the given namespace together with `etcd`, `minio`, a docker `registry`, and `kontinuous-ui`. This expects that the kubernetes cluster supports the LoadBalancer service.

Once a public IP for `kontinuous-ui` is available, the Github OAuth Application settings needs to be modified to reflect the actual IP address of `kontinuous-ui` for the Homepage and Callback URL.

Alternatively, for more customization, a sample yaml file for running kontinuous and its dependencies in Kubernetes can be found [here](./k8s-spec.yml.example). More details can be found [here](docs/setup.md).

Once running, add a [.pipeline.yml](#pipeline-spec) to the root of your Github repo and configure the webhooks. More details about pipeline spec creation can be found [here](docs/pipeline.md).

Example pipelines can be found in [/examples](./examples)

The [CLI client](#clients) or [API](#api) can be used to view build status or logs.

## Clients

There are two clients currently available:

### Kontinuous CLI

The CLI tool is the one that is used in the gettings started section. It can bootstrap Kontinuous to a running Kubernetes Cluster and can access details on Kontinuous pipelines and builds.

More info about the CLI can be found [here](https://github.com/AcalephStorage/kontinuous/tree/develop/cli) and the binary can be downloaded [here](https://github.com/AcalephStorage/kontinuous/releases).

### Kontinuous UI

Kontinuous UI is a web based client for Kontinuous. Bootstrapping Kontinuous using the CLI will install the UI on the Kubernetes Cluster too. More info about the UI can be found [here](https://github.com/AcalephStorage/kontinuous-ui).

## API

Kontinuous is accessible from it's API and docs can be viewed via Swagger. More details about using the API and Authentication can be found [here](docs/api.md).

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
