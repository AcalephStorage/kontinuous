package pipeline

import (
	"encoding/json"
	"os"
	"testing"
)

var (
	validYamlSpec = `
---
apiVersion: v1alpha1
kind: Pipeline
metadata:
  name: my-pipeline
  namespace: acaleph
spec:
  selector:
    matchLabels:
      app: my-pipeline-infra
  template:
    metadata:
      name: awesome-webapp
      labels:
        app: awesome-webapp
    notif:
      - type: slack
        metadata: 
          url: slackurl
          password: slackpassword
          email: slackemail    
    stages:
      - name: Build Image
        type: docker_build
        params:
          dockerfile_path: /
          dockerfile_name: Dockerfile

      - name: Deploy Image
        type: docker_publish
        params:
          external_registry: quay.io
          external_image_name: acaleph/kontinuous-agent
          require_credentials: "TRUE"
          username: acaleph
          password: ...
          email: ...

 `
	validCommandYamlSpec = `
---
apiVersion: v1alpha1
kind: Pipeline
metadata:
  name: my-pipeline
spec:
  selector:
    matchLabels:
      app: command-pipeline
  template:
    metadata:
      name: awesome-webapp
      labels:
        app: awesome-webapp
    stages:
      - name: Build Image
        type: command
        params:
          image: busybox:latest
          command:
            - command1
            - command2
            - command3
          args:
            - args1
            - args2
            - args3

      - name: Deploy Image
        type: docker_publish
        params:
          external_registry: quay.io
          external_image_name: acaleph/kontinuous-agent
          require_credentials: "TRUE"
          username: acaleph
          password: ...
          email: ...

 `

	validJobBuildInfo = `{
  "pipeline": "TestPipeline",
  "pipeline_uuid": "XYZ",
  "build": "1",
  "stage": "1",
  "commit": "1348767821643" ,
  "user": "acale" ,
  "repo": "acaleRepo"  ,
  "owner": "acaleph" ,
  "deployUrl": "localhost:1234"
}
`

	invalidYamlSpec = `
---
apiVersions: v1alpha1
kinds: Pipeline
metadata:
    name: my-pipeline
  namespace: acaleph
spec:
  selector:
     matchLabels:
      app: my-pipeline-infra

 `

	invalidJobBuildInfo = `{
  "pipeline": "TestPipeline",
  "pipeline_uuid": "XYZ",
  "build": "1",
  "stage": "2",
  "commit": "1348767821643" ,
  "user": "acale" ,
  "repo": "acaleRepo"  ,
  "owner": "acaleph" ,
  "deployUrl": "localhost:1234",
}
`

	envVar = map[string]string{
		"DEPLOY_URL":        "http://localhost:4200",
		"S3_URL":            "http://123.123.123:4000",
		"S3_ACCESS_KEY":     "123TOKEN",
		"S3_SECRET_KEY":     "123SECRET",
		"INTERNAL_REGISTRY": "internal-registry",
	}
)

func setEnvVariables(env map[string]string) {
	for key, value := range env {
		os.Setenv(key, value)
	}
}

func TestValidInfoBuildJob(t *testing.T) {

	definition, _ := GetDefinition([]byte(validYamlSpec))
	jobInfo, _ := GetJobBuildInfo([]byte(validJobBuildInfo))
	setEnvVariables(envVar)
	//set Environment Variables

	job, _ := build(definition, jobInfo)

	result, err := json.MarshalIndent(job, "", "\t")
	if err != nil {
		t.Fatalf("Create K8s Job - Failed. Should return a valid json object")
	}

	if result != nil {
		t.Log("Create K8s Job - Successful!")
	}

}

func TestInvalidDefinitionInfoBuildJob(t *testing.T) {

	_, err := GetDefinition([]byte(invalidYamlSpec))

	if err == nil {
		t.Fatalf("Create K8s Job Should Fail due to Invalid Definition YAML Info")
	} else {
		t.Log("Create K8s Job Fails due to Invalid Definition YAML Info")
	}
}

func TestInvalidJobBuildInfoBuildJob(t *testing.T) {

	_, err := GetJobBuildInfo([]byte(invalidJobBuildInfo))
	if err == nil {
		t.Fatal("Create K8s Job should Fail due to Invalid JobBuildInfo JSON")
	} else {
		t.Log("Create K8s Job Fails due to Invalid JobBuildInfo JSON")
	}

}

func TestValidInfoCommandJob(t *testing.T) {

	definition, _ := GetDefinition([]byte(validCommandYamlSpec))
	jobInfo, _ := GetJobBuildInfo([]byte(validJobBuildInfo))

	job, _ := build(definition, jobInfo)
	result, err := json.MarshalIndent(job, "", "\t")

	if err != nil {
		t.Fatal("Create K8s Job Fails: Unable to Parse yaml")
	}

	if result != nil {
		if len(job.Spec.Template.Spec.Containers) <= 0 {
			t.Fatal("Create K8s Job Fails: Unable to create command docker")
		} else {
			t.Log(len(job.Spec.Template.Spec.Containers))
		}
	}

}
