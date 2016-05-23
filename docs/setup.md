Kontinuous Setup
================

This document details the setup of Kontinuous without using the CLI bootstrap. 

## Dependencies

Kontinuous is dependent of the following:

### Kubernetes

Kontinuous should runs on top of a Kubernetes cluster. It uses Jobs heavily so it will require  at least v1.1 with Jobs enabled.

### etcd

etcd is used as a backend for storing pipeline and build details. This is a dedicated instance  to avoid polluting the Kubernetes etcd cluster.

### Minio

Minio is used to store logs and artifacts. S3 could also be used as it is compatible with minio although this hasn't been tested yet.

### Docker Registry

Kontinuous stores docker registry internal and uses an internal docker registry.

## Running in Kubernetes

Kontinuous is meant to run inside a kubernetes cluster, preferrably by a Deployment or Replication Controller.

### Docker Image

The docker image can be found here: [quay.io/acaleph/kontinuous](https://quay.io/acaleph/kontinuous)

### Environment Variables

The following environment variables needs to be defined:

| Environment Variable | Description                             | Example                |
|----------------------|-----------------------------------------|------------------------|
| KV_ADDRESS           | The etcd address                        | etcd:2379              |
| S3_URL               | The minio address                       | http://minio:9000      |
| KONTINUOUS_URL       | The address where kontinuous is running | http://kontinuous:8080 |
| INTERNAL_REGISTRY    | The internal registry address           | internal-registry:5000 |

### Secrets

A Kubernetes Secret needs to be defined and mounted on `/.secret`. The secret should have a key named `kontinuous-secrets` and contains the following data (must be base64 encoded):

```json
{
  "AuthSecret": "base64 encoded auth secret",
  "S3SecretKey": "s3 secret key",
  "S3AccessKey": "s3 access key",
  "GithubClientID": "github client ID",
  "GithubClientSecret": "github client secret"
}
```

#### AuthSecret

AuthSecret is the secret used for signing JSON Web Tokens used for authentication. This can be any base64 encoded string. More details about Authentication can be found [here](docs/api.md).

#### S3AccessKey & S3SecretKey

S3AccessKey and S3SecretKey are the keys taken from Minio. These can be retrieved from minio using the following command:

```
$ kubectl logs --namespace={namespace} {minio-pod-name}
```

#### GithubClientID & GithubClientSecret

GithubClientID and GithubClientSecret are optional. They are needed if running Kontinuous UI as the UI requires Github login. These are taken from the Github OAuth Application details. More details about Authentication can be found [here](docs/api.md)

### Ports

Kontinuous uses port `3005`. This needs to be exposed.


## Notes

Kontinuous internal registry uses Cluster IP. Should there be any changes on the IP address, please execute the following command:

```
kubectl apply -f < KONTINUOUS_SPEC_FILE.yml >
```


