package pipeline

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	"encoding/json"

	"github.com/AcalephStorage/kontinuous/kube"
	"github.com/Sirupsen/logrus"
)

// CreateJob creates a kubernetes Job for the given build information
func CreateJob(definition *Definition, jobInfo *JobBuildInfo) (j *kube.Job, err error) {

	newJob, _ := build(definition, jobInfo)

	err = deployJob(newJob)
	if err != nil {
		logrus.WithError(err).Errorln("Unable to Create Job")
		return nil, err
	}

	return newJob, nil
}

func GetJobBuildInfo(jobInfo []byte) (payload *JobBuildInfo, err error) {

	if len(jobInfo) == 0 {
		return nil, errors.New("Empty JSON String")
	}

	if err = json.Unmarshal(jobInfo, &payload); err != nil {
		return nil, err
	}

	return payload, nil
}

func build(definition *Definition, jobInfo *JobBuildInfo) (j *kube.Job, err error) {

	namespace := getNamespace(definition)
	name := fmt.Sprintf("%s-%s-%s", jobInfo.PipelineUUID, jobInfo.Build, jobInfo.Stage)
	j = kube.NewJob(name, namespace)

	addJobDetail(j, definition, jobInfo)
	addSpecDetails(j, definition, jobInfo)
	return j, nil

}

func addJobDetail(j *kube.Job, definition *Definition, jobInfo *JobBuildInfo) {

	selectors := map[string]string{
		"pipeline": jobInfo.PipelineUUID,
		"build":    jobInfo.Build,
		"stage":    jobInfo.Stage,
	}

	for key, value := range selectors {
		j.AddSelectorMatchLabel(key, value)
	}
}

func addSpecDetails(j *kube.Job, definitions *Definition, jobInfo *JobBuildInfo) {

	stage := getCurrentStage(definitions, jobInfo)

	if len(stage.Artifacts) != 0 {
		artifacts := getArtifacts(stage)
		j.AddAnnotations("kontinuous_artifacts", artifacts)
	}
	source := j.AddPodVolume("kontinuous-source", "/kontinuous/src")
	status := j.AddPodVolume("kontinuous-status", "/kontinuous/status")
	docker := j.AddPodVolume("kontinuous-docker", "/var/run/docker.sock")
	secrets := getSecrets(stage, getNamespace(definitions))

	agentContainer := createAgentContainer(definitions, jobInfo)
	agentContainer.AddVolumeMountPoint(source, "/kontinuous/src", false)
	agentContainer.AddVolumeMountPoint(status, "/kontinuous/status", false)
	agentContainer.AddVolumeMountPoint(docker, "/var/run/docker.sock", false)
	setContainerEnv(agentContainer, secrets)
	addJobContainer(j, agentContainer)

	switch stage.Type {
	case "docker_build":

		dockerContainer := createDockerContainer(stage, jobInfo, "BUILD")
		dockerContainer.AddVolumeMountPoint(source, "/kontinuous/src", false)
		dockerContainer.AddVolumeMountPoint(status, "/kontinuous/status", false)
		dockerContainer.AddVolumeMountPoint(docker, "/var/run/docker.sock", false)
		setContainerEnv(dockerContainer, secrets)
		addJobContainer(j, dockerContainer)

	case "docker_publish":
		dockerContainer := createDockerContainer(stage, jobInfo, "PUBLISH")
		dockerContainer.AddVolumeMountPoint(source, "/kontinuous/src", false)
		dockerContainer.AddVolumeMountPoint(status, "/kontinuous/status", false)
		dockerContainer.AddVolumeMountPoint(docker, "/var/run/docker.sock", false)
		setContainerEnv(dockerContainer, secrets)
		addJobContainer(j, dockerContainer)

	case "command":
		commandContainer := createCommandContainer(stage, jobInfo)
		commandContainer.AddVolumeMountPoint(docker, "/var/run/docker.sock", false)
		setContainerEnv(commandContainer, secrets)
		addJobContainer(j, commandContainer)
	}

}

func getCurrentStage(definitions *Definition, jobInfo *JobBuildInfo) (stage *Stage) {

	index, _ := strconv.Atoi(jobInfo.Stage)

	if currentIndex := index - 1; 0 <= currentIndex && currentIndex < len(definitions.Spec.Template.Stages) {
		return &definitions.Spec.Template.Stages[currentIndex]
	}

	return &Stage{}
}

func createAgentContainer(definitions *Definition, jobInfo *JobBuildInfo) *kube.Container {

	container := createJobContainer("kontinuous-agent", "quay.io/acaleph/kontinuous-agent:latest")
	envVars := map[string]string{
		"REQUIRE_SOURCE_CODE": "TRUE",
		"GIT_COMMIT":          jobInfo.Commit,
		"GIT_USER":            jobInfo.User,
		"GIT_REPO":            jobInfo.Repo,
		"GIT_OWNER":           jobInfo.Owner,
		"PIPELINE_ID":         jobInfo.PipelineUUID,
		"BUILD_ID":            jobInfo.Build,
		"STAGE_ID":            jobInfo.Stage,
		"S3_URL":              os.Getenv("S3_URL"),
		"S3_ACCESS_KEY":       os.Getenv("S3_ACCESS_KEY"),
		"S3_SECRET_KEY":       os.Getenv("S3_SECRET_KEY"),
		"KONTINUOUS_URL":      os.Getenv("KONTINUOUS_URL"),
		"NAMESPACE":           getNamespace(definitions),
		"ARTIFACT_URL":        "",
	}

	setContainerEnv(container, envVars)
	return container
}

func createDockerContainer(stage *Stage, jobInfo *JobBuildInfo, mode string) *kube.Container {
	imageName := fmt.Sprintf("%s-%s", jobInfo.PipelineUUID, jobInfo.Build)
	container := createJobContainer("docker-agent", "quay.io/acaleph/docker-agent:latest")

	envVar := map[string]string{
		"INTERNAL_REGISTRY":   os.Getenv("INTERNAL_REGISTRY"),
		"DOCKERFILE_NAME":     "Dockerfile",
		"DOCKERFILE_PATH":     ".",
		"REQUIRE_CREDENTIALS": "TRUE",
		"IMAGE_NAME":          imageName,
		"MODE":                mode,
		"PIPELINE_ID":         jobInfo.PipelineUUID,
		"BUILD_ID":            jobInfo.Build,
		"STAGE_ID":            jobInfo.Stage,
		"IMAGE_TAG":           jobInfo.Commit,
		"BRANCH":              jobInfo.Branch,
	}

	for stageEnvKey, stageEnvValue := range stage.Params {
		envVar[strings.ToUpper(stageEnvKey)] = stageEnvValue.(string)
	}

	setContainerEnv(container, envVar)
	return container
}

func createCommandContainer(stage *Stage, jobInfo *JobBuildInfo) *kube.Container {

	containerName := fmt.Sprintf("%s-%s", jobInfo.PipelineUUID, jobInfo.Build)
	imageName := fmt.Sprintf("%s/%s:%s", os.Getenv("INTERNAL_REGISTRY"), containerName, jobInfo.Commit)
	container := createJobContainer(containerName, imageName)

	for paramKey, paramValue := range stage.Params {

		switch strings.ToUpper(paramKey) {
		case "COMMAND":
			commands := paramValue.([]interface{})
			stringCommand := make([]string, len(commands))
			for i, c := range commands {
				stringCommand[i] = c.(string)
			}
			container.SetCommand(stringCommand...)
		case "ARGS":
			args := paramValue.([]interface{})
			stringArg := make([]string, len(args))
			for i, a := range args {
				stringArg[i] = a.(string)
			}
			container.SetArgs(stringArg...)
		case "IMAGE":
			container.Image = paramValue.(string)
		}
	}
	return container
}

func deployJob(j *kube.Job) error {
	jobClient, _ := kube.NewClient("https://kubernetes.default")
	return jobClient.CreateJob(j)
}

func setContainerEnv(container *kube.Container, envVars map[string]string) {
	for key, value := range envVars {
		container.AddEnv(key, value)
	}

}

func getSecrets(stage *Stage, namespace string) map[string]string {
	jobClient, _ := kube.NewClient("https://kubernetes.default")
	secrets := make(map[string]string)

	for _, secret := range stage.Secrets {
		secretEnv, err := jobClient.GetSecret(namespace, secret)
		if err != nil {
			logrus.Printf("Unable to get secret %s", secret)
			continue
		}
		logrus.Printf("Secret retrieved %s", secretEnv)
		for key, value := range secretEnv {
			secrets[key] = value
		}
	}
	return secrets
}

func getArtifacts(stage *Stage) string {
	return fmt.Sprintf("{%s}", strings.Join(stage.Artifacts, ","))
}

func createJobContainer(name string, image string) *kube.Container {
	container := &kube.Container{
		Name:            name,
		Image:           image,
		ImagePullPolicy: "Always",
	}
	return container
}

func addJobContainer(j *kube.Job, container *kube.Container) {
	j.Spec.Template.Spec.Containers = append(j.Spec.Template.Spec.Containers, container)
}

func getNamespace(definition *Definition) string {
	if definition.Metadata["namespace"] == nil {
		return "default"
	}
	return definition.Metadata["namespace"].(string)
}
