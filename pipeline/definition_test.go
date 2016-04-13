package pipeline

import (
	"testing"
)

var validyamlSpec string

func TestReadValidPipeline(t *testing.T) {
	validyamlSpec = `
apiVersion: v1alpha1
kind: Pipeline
metadata:
  name: my-pipeline
  namespace: acaleph
spec:
  selector:
    matchLabels:
      app: my-pipeline-infra # can be used as a selector for finding infra launched during a build?
  template: # taken from Job spec... needed?
    metadata:
      name: my-pipeline
      labels:
        app: my-pipeline
      image: acaleph/deploy-base # Overridable?
    stages:
    - name: Test Infra
      type: command
      params:
        command: ["test_infra.sh", "--hack-the-gibson"]
        env: # env vars
          - name: MYVAR
            value: my value
        artifact_paths: # sent to minio?
          - "logs/**/*"
          - "coverage/**/*"
        timeout: 60 # kills after X minutes?
    - name: "Have you finished testing?" # friendly name
      type: block # waits for the user to approve
    - name: Teardown Deploy local test infra
      type: deploy_cleanup # stop
      selector:
        build: my-val-stage-1 # selects build to stop
    - name: "Do you want to Publish to Production?" # friendly name
      type: block # waits for the user to approve
    - name: Deploy production
      type: deploy
      params:
        template: production.yaml # DM template
        properties: # DM properties
          external_service: true
          replicas: 3
      labels: # labels to cleanup build
        state: canary
 `
	_, err := GetDefinition([]byte(validyamlSpec))

	if err == nil {
		t.Log("Pipeline Parser able to read parse yaml file")
	} else {
		t.Fatalf("Pipeline Parser must be able to parse yaml")
	}

}

func TestReadInvalidPipeline(t *testing.T) {

	_, err := GetDefinition([]byte("---invalid yaml string"))

	if err != nil {
		t.Log("Pipeline Parser returns error on invalid yaml file")
	} else {
		t.Fatalf("Pipeline Parser must return error on invalid yaml")
	}

}

func TestReadEmptyPipeline(t *testing.T) {

	_, err := GetDefinition([]byte{})

	if err != nil {
		t.Log("Pipeline Parser returns error on empty yaml file")
	} else {
		t.Fatalf("Pipeline Parser must return error on empty yaml file")
	}
}
