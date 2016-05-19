Kontinuous Pipelines
====================

A repository needs to define a pipeline spec by adding `.pipeline.yml` to the root directory of the repository.

## Pipeline Specification

Here's an example pipeline specification.

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

Some important fields on the spec:

| Field                 | Description                                               |
|-----------------------|-----------------------------------------------------------|
| metadata.namespace    | Defines which namespace to run the builds of the pipeline |
| spec.template.notif   | Defines the notification used by this pipeline            |
| spec.template.secrets | Defines the secrets used by this pipeline                 | 
| spec.stages           | Defines the build stages                                  |

### Notification

Currently only slack is supported. More notifiers will be added in the future. 

```yaml
spec:
  template:
    notif:
      - type: slack
```

Slack details are taken from the secrets. The following secrets needs to be defined:

| secret name  | details                                        |
|--------------|------------------------------------------------|
| slackchannel | the channel to post the notifications          |
| slackurl     | the slack url                                  |
| slackuser    | the user to display when showing notifications |

### Secrets

Kubernetes secrets can be added to the pipeline. Each of the secret entries will be added as environment variables.

```yaml
spec:
  template:
    secrets:
      - secret1
      - secret2
```

### Stages

Stages are the build definitions. Currently there are four different stage types. 

```yaml
spec:
  template:
    stages:
      - name: Friendly name
        type: docker_build
        params: {}  
```

| Stage          | Description                                                       | 
|----------------|-------------------------------------------------------------------|
| docker_build   | build a docker image                                              |
| docker_publish | publish a docker image to an external registry                    |
| command        | run commands against a previously built image or a specific image | 
| deploy         | deploys a kubernetes spec file to kubernetes                      |

#### docker_build

Builds a Docker Image and pushes the images to the internal registry. It can work without additional params. By default, it uses the `Dockerfile` inside the repository root. 

Optional params are:

| Parameter       | Description                              |
|-----------------|------------------------------------------|
| dockerfile_path | the path where the Dockerfile is located |
| dockerfile_name | the file name of the Dockerfile          |

#### docker_publish

pushes the previously build Docker image to an external registry

Required Params:

| Parameter           | Description                                      |
|---------------------|--------------------------------------------------|
| external_registry   | the external registry name (eg. quay.io)         |
| external\_image\_name | the name of the image (eg. acaleph/kontinuous) |

Optional params:

| Parameter            | Description                                      |
|----------------------|--------------------------------------------------|
| require_crendentials | true/false. flag to require registry credentials |

Required secrets:

| Secret Name | Details             |
|-------------|---------------------|
| dockeruser  | the docker user     |
| dockerpass  | the docker password |
| dockeremail | the docker email    |

#### command

Runs a command on the newly create docker image or on the image specified. 

Required params:

| Parameter | Description                                    |
|-----------|------------------------------------------------|
| command   | list of string defining the command to execute |

Optional params:

| Parameter    | Description                                                  |
|--------------|--------------------------------------------------------------|
| args         | list of string defining the args for the command             |
| image        | custom image to use for running the build                    |
| dependencies | list of dependencies to run, these are kubernetes spec files |
| working_dir  | change the working directory                                 |

#### deploy

Deploys a kubernetes spec.

Required params:

| Parameter   | Description                        |
|-------------|------------------------------------|
| deploy_file | the kubernetes spec file to deploy |

