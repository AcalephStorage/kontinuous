package main

import (
	"bytes"
	b64 "encoding/base64"
	"errors"
	"fmt"
	"github.com/AcalephStorage/kontinuous/api"
	"net/http"
	"text/template"
)

var initDefinition = `
---
apiVersion: v1alpha1
kind: Pipeline
metadata:
  name: {{.ProjectName}}
  namespace: {{.Namespace}}
spec:
  selector:
    matchLabels:
      app: {{.Namespace}}
  template:
    metadata:
      name: {{.Namespace}}
      labels:
        app: {{.ProjectName}}
    stages:
      - name: Build Image
        type: docker_build
`

type InitDefinition struct {
	ProjectName string
	Namespace   string
}

func generateInitResource(templateStr string, init *InitDefinition) (string, error) {

	template := template.New("kontinuous init Template")
	template, _ = template.Parse(templateStr)
	var b bytes.Buffer

	err := template.Execute(&b, init)

	if err != nil {
		fmt.Println(err.Error())
	}

	return b.String(), nil

}

func generateInit(token, owner, repo, namespace string) error {

	initDef := InitDefinition{
		Namespace:   namespace,
		ProjectName: repo,
	}

	def, err := generateInitResource(initDefinition, &initDef)
	if err != nil {
		fmt.Printf("error", err.Error())
		return err
	}

	repoPath := fmt.Sprintf("/repos/%s/%s/contents/.pipeline.yml", owner, repo)

	//check if pipeline already exists
	client := http.DefaultClient
	contents, _ := api.SendGithubRequest(token, client, "GET", repoPath, nil)

	if contents != nil {
		return errors.New(".pipeline.yml already exists")
	}

	sEnc := b64.StdEncoding.EncodeToString([]byte(def))

	contentData := fmt.Sprintf(`{
    "committer":{
      "name":"kontinuous",
      "email":"admin@kontinuous.sg"
      },
    "message": "Commit Initial Pipeline",
    "content": "%s"}`, sEnc)

	data := []byte(contentData)
	_, err = api.SendGithubRequest(token, client, "PUT", repoPath, data)
	if err != nil {
		fmt.Printf("Error: %s", err.Error())
		return err
	}
	fmt.Println(".pipeline.yml is now available in the repository.")
	return nil
}

func Init(namespace, owner, repo, token string) error {

	fmt.Printf("Initializing Kontinuous in repository: %s/%s \n", owner, repo)
	err := generateInit(token, owner, repo, namespace)
	if err != nil {
		return err
	}
	return nil
}
