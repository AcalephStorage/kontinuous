package pipeline

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	"encoding/base64"
	"encoding/json"
	"text/template"

	"github.com/AcalephStorage/kontinuous/kube"
	"github.com/AcalephStorage/kontinuous/scm"
	"github.com/Masterminds/sprig"
	"github.com/Sirupsen/logrus"
)

// CreateJob creates a kubernetes Job for the given build information
func CreateJob(definition *Definition, jobInfo *JobBuildInfo, scmClient scm.Client) (j *kube.Job, err error) {

	newJob, _ := build(definition, jobInfo, scmClient)

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

func build(definition *Definition, jobInfo *JobBuildInfo, scmClient scm.Client) (j *kube.Job, err error) {

	namespace := getNamespace(definition)
	name := fmt.Sprintf("%s-%s-%s", jobInfo.PipelineUUID, jobInfo.Build, jobInfo.Stage)
	j = kube.NewJob(name, namespace)

	addJobDetail(j, definition, jobInfo)
	addSpecDetails(j, definition, jobInfo, scmClient)
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

func addSpecDetails(j *kube.Job, definitions *Definition, jobInfo *JobBuildInfo, scmClient scm.Client) {

	stage := getCurrentStage(definitions, jobInfo)
	kontinuousVars := getKontinuousVars(definitions, jobInfo)
	parseStageTemplate(stage, kontinuousVars, definitions.Spec.Template.Vars, stage.Vars)

	source := j.AddPodVolume("kontinuous-source", "/kontinuous/src")
	status := j.AddPodVolume("kontinuous-status", "/kontinuous/status")
	docker := j.AddPodVolume("kontinuous-docker", "/var/run/docker.sock")
	secrets := getSecrets(getNamespace(definitions), definitions.Spec.Template.Secrets, stage.Secrets)
	allVars := getVars(kontinuousVars, definitions.Spec.Template.Vars, stage.Vars)

	agentContainer := createAgentContainer(definitions, jobInfo)
	agentContainer.AddVolumeMountPoint(source, "/kontinuous/src", false)
	agentContainer.AddVolumeMountPoint(status, "/kontinuous/status", false)
	agentContainer.AddVolumeMountPoint(docker, "/var/run/docker.sock", false)
	setContainerEnv(agentContainer, secrets)
	setContainerEnv(agentContainer, allVars)
	addJobContainer(j, agentContainer)

	switch stage.Type {
	case "docker_build":

		dockerContainer := createDockerContainer(stage, jobInfo, "BUILD")
		dockerContainer.AddVolumeMountPoint(source, "/kontinuous/src", false)
		dockerContainer.AddVolumeMountPoint(status, "/kontinuous/status", false)
		dockerContainer.AddVolumeMountPoint(docker, "/var/run/docker.sock", false)
		setContainerEnv(dockerContainer, secrets)
		setContainerEnv(dockerContainer, allVars)
		addJobContainer(j, dockerContainer)

	case "docker_publish":
		dockerContainer := createDockerContainer(stage, jobInfo, "PUBLISH")
		dockerContainer.AddVolumeMountPoint(source, "/kontinuous/src", false)
		dockerContainer.AddVolumeMountPoint(status, "/kontinuous/status", false)
		dockerContainer.AddVolumeMountPoint(docker, "/var/run/docker.sock", false)
		setContainerEnv(dockerContainer, secrets)
		setContainerEnv(dockerContainer, allVars)
		addJobContainer(j, dockerContainer)

	case "command":
		commandContainer := createCommandContainer(stage, jobInfo)
		commandContainer.AddVolumeMountPoint(source, "/kontinuous/src", false)
		commandContainer.AddVolumeMountPoint(status, "/kontinuous/status", false)
		commandContainer.AddVolumeMountPoint(docker, "/var/run/docker.sock", false)
		setContainerEnv(commandContainer, secrets)
		setContainerEnv(commandContainer, allVars)

		keySlice := make([]string, 0)
		for _, env := range commandContainer.Env {
			keySlice = append(keySlice, env.Name)
		}
		keys := strings.Join(keySlice, " ")
		commandContainer.AddEnv("ENV_KEYS", keys)
		addJobContainer(j, commandContainer)

	case "deploy":
		deployContainer := createDeployContainer(allVars, stage, definitions, jobInfo, scmClient)
		deployContainer.AddVolumeMountPoint(source, "/kontinuous/src", false)
		deployContainer.AddVolumeMountPoint(status, "/kontinuous/status", false)
		deployContainer.AddVolumeMountPoint(docker, "/var/run/docker.sock", false)
		setContainerEnv(deployContainer, secrets)
		setContainerEnv(deployContainer, allVars)
		addJobContainer(j, deployContainer)
	}

	if stage.Artifacts != nil && len(stage.Artifacts) > 0 {
		j.AddAnnotations("kontinuous_artifacts", strings.Join(stage.Artifacts, " "))
	}

}

func getCurrentStage(definitions *Definition, jobInfo *JobBuildInfo) (stage *Stage) {

	index, _ := strconv.Atoi(jobInfo.Stage)

	if currentIndex := index - 1; 0 <= currentIndex && currentIndex < len(definitions.Spec.Template.Stages) {
		return &definitions.Spec.Template.Stages[currentIndex]
	}

	return &Stage{}
}

func getKontinuousVars(definitions *Definition, jobInfo *JobBuildInfo) map[string]interface{} {
	return map[string]interface{}{
		"KONTINUOUS_PIPELINE_ID":       jobInfo.PipelineUUID,
		"KONTINUOUS_BUILD_ID":          jobInfo.Build,
		"KONTINUOUS_STAGE_ID":          jobInfo.Stage,
		"KONTINUOUS_BRANCH":            jobInfo.Branch,
		"KONTINUOUS_NAMESPACE":         getNamespace(definitions),
		"KONTINUOUS_ARTIFACT_URL":      "",
		"KONTINUOUS_INTERNAL_REGISTRY": os.Getenv("INTERNAL_REGISTRY"),
		"KONTINUOUS_COMMIT":            jobInfo.Commit,
		"KONTINUOUS_URL":               os.Getenv("KONTINUOUS_URL"),
	}

}

func createAgentContainer(definitions *Definition, jobInfo *JobBuildInfo) *kube.Container {

	container := createJobContainer("kontinuous-agent", "quay.io/acaleph/kontinuous-agent:latest")
	envVars := map[string]string{
		"REQUIRE_SOURCE_CODE": "TRUE",
		"GIT_COMMIT":          jobInfo.Commit,
		"GIT_USER":            jobInfo.User,
		"GIT_REPO":            jobInfo.Repo,
		"GIT_OWNER":           jobInfo.Owner,
		"S3_URL":              os.Getenv("S3_URL"),
		"S3_ACCESS_KEY":       os.Getenv("S3_ACCESS_KEY"),
		"S3_SECRET_KEY":       os.Getenv("S3_SECRET_KEY"),
	}

	setContainerEnv(container, envVars)
	return container
}

func createDeployContainer(deploymentVars map[string]string, stage *Stage, definitions *Definition, jobInfo *JobBuildInfo, scmClient scm.Client) *kube.Container {
	container := createJobContainer("deploy-agent", "quay.io/acaleph/deploy-agent:latest")
	deployFile := fmt.Sprintf("%v", stage.Params["deploy_file"])
	deployDir := fmt.Sprintf("%v", stage.Params["deploy_dir"])
	expose, _ := strconv.ParseBool(fmt.Sprintf("%v", stage.Params["expose"]))

	ref := jobInfo.Commit
	if ref == "" {
		ref = jobInfo.Branch
	}
	fileToParse := make([]interface{}, 0)

	if deployFile != "" {
		file, ok := scmClient.GetFileContent(jobInfo.Owner, jobInfo.Repo, deployFile, ref)
		if ok {
			fileToParse = append(fileToParse, file)
		}
	}

	if deployDir != "" {
		dirContents, ok := scmClient.GetDirectoryContent(jobInfo.Owner, jobInfo.Repo, deployDir, ref)
		if ok {
			fileToParse = append(fileToParse, dirContents...)
		}
	}

	template := template.New("kontinuous")
	parsedFiles := make([]string, 0)

	for _, tobeParsed := range fileToParse {
		b := bytes.Buffer{}
		template, _ = template.Funcs(sprig.TxtFuncMap()).Parse(string(tobeParsed.([]byte)))
		err := template.ExecuteTemplate(&b, "kontinuous", deploymentVars)
		if err != nil {
			continue
		}

		toEncode := b.String()
		if !expose {
			toEncode = strings.Replace(toEncode, "LoadBalancer", "ClusterIP", -1)
		}

		parsedFiles = append(parsedFiles, base64.StdEncoding.EncodeToString([]byte(toEncode)))
	}

	deployFiles := strings.Join(parsedFiles, " ")
	container.AddEnv("DEPLOY_FILES", deployFiles)
	return container
}

func createDockerContainer(stage *Stage, jobInfo *JobBuildInfo, mode string) *kube.Container {
	imageName := fmt.Sprintf("%s-%s", jobInfo.PipelineUUID, jobInfo.Build)
	container := createJobContainer("docker-agent", "quay.io/acaleph/docker-agent:latest")

	envVar := map[string]string{
		"DOCKERFILE_NAME":     "Dockerfile",
		"DOCKERFILE_PATH":     ".",
		"REQUIRE_CREDENTIALS": "TRUE",
		"IMAGE_NAME":          imageName,
		"MODE":                mode,
		"IMAGE_TAG":           jobInfo.Commit,
	}

	for stageEnvKey, stageEnvValue := range stage.Params {
		envVar[strings.ToUpper(stageEnvKey)] = fmt.Sprintf("%v", stageEnvValue)
	}

	setContainerEnv(container, envVar)
	return container
}

func createCommandContainer(stage *Stage, jobInfo *JobBuildInfo) *kube.Container {

	containerName := "command-agent"
	cmdImageName := fmt.Sprintf("%s-%s", jobInfo.PipelineUUID, jobInfo.Build)
	cmdImage := fmt.Sprintf("%s/%s:%s", os.Getenv("INTERNAL_REGISTRY"), cmdImageName, jobInfo.Commit)
	imageName := "quay.io/acaleph/command-agent:latest"
	container := createJobContainer(containerName, imageName)
	container.Image = imageName
	container.AddEnv("IMAGE", cmdImage)
	container.WorkingDir = fmt.Sprintf("/kontinuous/src")
	container.AddEnv("WORKING_DIR", fmt.Sprintf("/kontinuous/src/%s/%s/%s", jobInfo.PipelineUUID, jobInfo.Build, jobInfo.Stage))

	for paramKey, paramValue := range stage.Params {

		switch strings.ToUpper(paramKey) {
		case "COMMAND":
			commands := paramValue.([]interface{})
			stringCommand := make([]string, len(commands))
			for i, c := range commands {
				stringCommand[i] = c.(string)
			}
			container.AddEnv("COMMAND", strings.Join(stringCommand, " "))
		case "ARGS":
			args := paramValue.([]interface{})
			stringArg := make([]string, len(args))
			for i, a := range args {
				stringArg[i] = a.(string)
			}
			container.SetArgs(stringArg...)
		case "IMAGE":
			container.AddEnv("IMAGE", paramValue.(string))
		case "WORKING_DIR":
			container.WorkingDir = paramValue.(string)
			container.AddEnv("WORKING_DIR", paramValue.(string))
		case "DEPENDENCIES":
			dependencies := paramValue.([]interface{})
			stringDep := make([]string, len(dependencies))
			for i, d := range dependencies {
				stringDep[i] = d.(string)
			}
			container.AddEnv("DEPENDENCIES", strings.Join(stringDep, " "))
		default:
			container.AddEnv(strings.ToUpper(paramKey), fmt.Sprintf("%v", paramValue))
		}
	}

	return container
}

func deployJob(j *kube.Job) error {
	kubeClient, _ := kube.NewClient("https://kubernetes.default")
	return kubeClient.CreateJob(j)
}

func setContainerEnv(container *kube.Container, envVars map[string]string) {
	for key, value := range envVars {
		container.AddEnv(key, value)
	}

}

func getSecrets(namespace string, allSecrets ...[]string) map[string]string {
	kubeClient, _ := kube.NewClient("https://kubernetes.default")
	secrets := make(map[string]string)

	for _, secretArr := range allSecrets {
		for _, secret := range secretArr {
			secretEnv, err := kubeClient.GetSecret(namespace, secret)
			if err != nil {
				logrus.Printf("Unable to get secret %s", secret)
				continue
			}
			for key, value := range secretEnv {
				secrets[key] = strings.TrimSpace(value)
			}
		}
	}
	return secrets
}

func getVars(varMaps ...map[string]interface{}) map[string]string {

	allVars := make(map[string]string)
	for _, varMap := range varMaps {
		for key, value := range varMap {
			allVars[key] = fmt.Sprintf("%v", value)
		}
	}

	return allVars
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
