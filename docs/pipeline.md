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
    vars:
      name: kontinuous
      namespace: acaleph
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
        secrets:
          - docker-credentials
      - name: Deploy Kontinuous
        type: deploy
        params:
          deploy_file: manifest.yml
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

Kubernetes secrets can be added to the pipeline. Each of the secret entries will be added as environment variables. This is accessible to all stages.

```yaml
spec:
  template:
    secrets:
      - secret1
      - secret2
```

### Vars

Users can define variables that will be accessible to all stages. These variables can also be used to replace template fields.


Here is the list of available Kontinuous variables.


| Vars                            | Description                                                                               |
|---------------------------------|-------------------------------------------------------------------------------------------|
| `KONTINUOUS_PIPELINE_ID`        |  Generated UUID for Kontinuous pipeline                                                   |
| `KONTINUOUS_BUILD_ID`           |  Current build number                                                                     |
| `KONTINUOUS_STAGE_ID`           |  Current stage number                                                                     |
| `KONTINUOUS_BRANCH`             |  Build Branch                                                                             |
| `KONTINUOUS_NAMESPACE`          |  Namespace defined in the .pipeline.yml                                                   |
| `KONTINUOUS_ARTIFACT_URL`       |  Artifact path specified by user                                                          | 
| `KONTINUOUS_INTERNAL_REGISTRY`  |  Used by kontinuous as its own registry. Default value from System env. INTERNAL_REGISTRY |
| `KONTINUOUS_COMMIT`             |  The commit of the build                                                                  |
| `KONTINUOUS_URL`                |  Current url of Kontinuous                                                                |


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
| require_crendentials | TRUE/FALSE. flag to require registry credentials |

Required secrets:

| Secret Name     | Details             |
|-----------------|---------------------|
| dockeruser      | the docker user     |
| dockerpassword  | the docker password |
| dockeremail     | the docker email    |

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

Deploys a kubernetes spec in the cluster. 

Params:

| Parameter   | Description                                                          |
|-------------|----------------------------------------------------------------------|
| deploy_file | the kubernetes spec file to deploy                                   |
| deploy_dir  | the directory for kubernetes  spec files to deploy                   |
| expose      | TRUE/FALSE. flag to expose services. Default is set to false         |

Note: Specification files in yaml format supports template. 


#### vars and secrets

Stage specific vars and secrets. 

Notes: If vars and secrets exists in the global scope, stage vars and secrets will override the value.

## Templates

Kontinuous supports template in `.pipeline.yml`, `deploy_file` and files in under `deploy_dir` directory.

```
Syntax:
 {{.<template variable>}}
```

```
eg. 
...
metadata:
  name: {{.name}}
  namespace: {{.namespace}}

```

To supply value in a template field you may use `vars`

```
eg.
vars:
  name: kontinuous
  namespace: acaleph
```

Kontinuous replaces the template field with the corresponding `vars` value.

```
eg. 
...
metadata:
  name: kontinuous
  namespace: acaleph

```