package pipeline

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	etcd "github.com/coreos/etcd/client"

	"github.com/AcalephStorage/kontinuous/kube"
	"github.com/AcalephStorage/kontinuous/notif"
	"github.com/AcalephStorage/kontinuous/store/kv"
	"github.com/Sirupsen/logrus"
)

// Build contains the details needed to run a build
type Build struct {
	ID           string   `json:"id"`
	Number       int      `json:"number"`
	Status       string   `json:"status"`
	Created      int64    `json:"created"`
	Started      int64    `json:"started"`
	Finished     int64    `json:"finished"`
	CurrentStage int      `json:"current_stage"`
	Branch       string   `json:"branch"`
	Commit       string   `json:"commit"`
	Author       string   `json:"author"`
	Event        string   `json:"event"`
	CloneURL     string   `json:"clone_url"`
	Pipeline     string   `json:"-"`
	Stages       []*Stage `json:"stages,omitempty"`
}

func getBuild(path string, kvClient kv.KVClient) *Build {
	b := new(Build)
	b.ID, _ = kvClient.Get(path + "/uuid")
	b.Status, _ = kvClient.Get(path + "/status")
	b.Branch, _ = kvClient.Get(path + "/branch")
	b.Commit, _ = kvClient.Get(path + "/commit")
	b.Author, _ = kvClient.Get(path + "/author")
	b.Event, _ = kvClient.Get(path + "/event")
	b.CloneURL, _ = kvClient.Get(path + "/clone-url")
	b.Pipeline, _ = kvClient.Get(path + "/pipeline")
	b.Number, _ = kvClient.GetInt(path + "/number")
	b.CurrentStage, _ = kvClient.GetInt(path + "/current-stage")
	created, _ := kvClient.Get(path + "/created")
	started, _ := kvClient.Get(path + "/started")
	finished, _ := kvClient.Get(path + "/finished")
	b.Created, _ = strconv.ParseInt(created, 10, 64)
	b.Started, _ = strconv.ParseInt(started, 10, 64)
	b.Finished, _ = strconv.ParseInt(finished, 10, 64)
	b.GetStages(kvClient)

	return b
}

// Save persists the build details to `etcd`
func (b *Build) Save(kvClient kv.KVClient) (err error) {
	buildsPrefix := fmt.Sprintf("%s%s/builds", pipelineNamespace, b.Pipeline)
	path := fmt.Sprintf("%s/%d", buildsPrefix, b.Number)
	isNew := false

	_, err = kvClient.GetDir(path)
	if err != nil || etcd.IsKeyNotFound(err) {
		isNew = true
	}

	// strings
	if err := kvClient.Put(path+"/uuid", b.ID); err != nil {
		return handleSaveError(path, isNew, err, kvClient)
	}
	if err := kvClient.Put(path+"/status", b.Status); err != nil {
		return handleSaveError(path, isNew, err, kvClient)
	}
	if err := kvClient.Put(path+"/branch", b.Branch); err != nil {
		return handleSaveError(path, isNew, err, kvClient)
	}
	if err := kvClient.Put(path+"/commit", b.Commit); err != nil {
		return handleSaveError(path, isNew, err, kvClient)
	}
	if err := kvClient.Put(path+"/author", b.Author); err != nil {
		return handleSaveError(path, isNew, err, kvClient)
	}
	if err := kvClient.Put(path+"/event", b.Event); err != nil {
		return handleSaveError(path, isNew, err, kvClient)
	}
	if err := kvClient.Put(path+"/clone-url", b.CloneURL); err != nil {
		return handleSaveError(path, isNew, err, kvClient)
	}
	if err := kvClient.Put(path+"/pipeline", b.Pipeline); err != nil {
		return handleSaveError(path, isNew, err, kvClient)
	}
	// int as string
	if err := kvClient.Put(path+"/created", strconv.FormatInt(b.Created, 10)); err != nil {
		return handleSaveError(path, isNew, err, kvClient)
	}
	if err := kvClient.Put(path+"/started", strconv.FormatInt(b.Started, 10)); err != nil {
		return handleSaveError(path, isNew, err, kvClient)
	}
	if err := kvClient.Put(path+"/finished", strconv.FormatInt(b.Finished, 10)); err != nil {
		return handleSaveError(path, isNew, err, kvClient)
	}
	// integers
	if err := kvClient.PutInt(path+"/number", b.Number); err != nil {
		return handleSaveError(path, isNew, err, kvClient)
	}
	if err := kvClient.PutInt(path+"/current-stage", b.CurrentStage); err != nil {
		return handleSaveError(path, isNew, err, kvClient)
	}
	// save stages
	if isNew {
		if err := b.CreateStages(kvClient); err != nil {
			return handleSaveError(path, isNew, err, kvClient)
		}
	}

	return nil
}

// CreateStages perists the build's stage details
func (b *Build) CreateStages(kvClient kv.KVClient) (err error) {
	buildsPrefix := fmt.Sprintf("%s%s/builds", pipelineNamespace, b.Pipeline)
	stagesPrefix := fmt.Sprintf("%s/%d/stages", buildsPrefix, b.Number)

	for idx, stage := range b.Stages {
		stage.Status = BuildPending
		stage.Index = idx + 1
		stage.ID = generateUUID()

		if err := stage.Save(stagesPrefix, kvClient); err != nil {
			return err
		}
	}

	return nil
}

// GetStages fetches all stages of the build from the store
func (b *Build) GetStages(kvClient kv.KVClient) ([]*Stage, error) {
	stagesPrefix := fmt.Sprintf("%s%s/builds/%d/stages", pipelineNamespace, b.Pipeline, b.Number)
	stageDirs, err := kvClient.GetDir(stagesPrefix)
	if err != nil {
		if etcd.IsKeyNotFound(err) {
			return make([]*Stage, 0), nil
		}
		return nil, err
	}

	b.Stages = make([]*Stage, len(stageDirs))
	for i, pair := range stageDirs {
		b.Stages[i] = getStage(pair.Key, kvClient)
	}

	return b.Stages, nil
}

// GetStage fetches a specific stage by its index
func (b *Build) GetStage(idx int, kvClient kv.KVClient) (*Stage, bool) {
	path := fmt.Sprintf("%s%s/builds/%d/stages/%d", pipelineNamespace, b.Pipeline, b.Number, idx)
	_, err := kvClient.GetDir(path)
	if err != nil || etcd.IsKeyNotFound(err) {
		return nil, false
	}

	return getStage(path, kvClient), true
}

func (b *Build) Notify(kvClient kv.KVClient) error {
	stageStatus := b.getStatus(kvClient)
	p := getPipeline(fmt.Sprintf("%s%s", pipelineNamespace, b.Pipeline), kvClient)
	var appNotifier notif.AppNotifier

	//TODO: will add more notification engines

	for _, notifier := range p.Notifiers {

		switch notifier.Type {
		case "slack":
			appNotifier = &notif.Slack{}
			metadata := make(map[string]interface{})
			metadata["channel"] = "slackchannel"
			metadata["url"] = "slackurl"
			metadata["username"] = "slackuser"
			notifier.Metadata = b.getSecrets(p.Secrets, notifier.Namespace, metadata)
			logrus.Info("Slack Info %s %s %s ", notifier.Metadata["channel"], notifier.Metadata["url"], notifier.Metadata["username"])
		}

		if appNotifier != nil {
			isPosted := appNotifier.PostMessage(b.Pipeline, b.Number, b.Status, stageStatus, notifier.Metadata)
			if !isPosted {
				return errors.New("Unable to post Message!")
			}
		}
	}

	return nil
}

func (b *Build) getSecrets(pipelineSecrets []string, namespace string, metadata map[string]interface{}) map[string]interface{} {
	logrus.Info("Get Slack Details from Secrets ")
	secrets := make(map[string]string)

	for _, secretName := range pipelineSecrets {
		kubeClient, _ := kube.NewClient("https://kubernetes.default")
		secretEnv, err := kubeClient.GetSecret(namespace, secretName)
		if err != nil {
			continue
		}
		for key, value := range secretEnv {
			secrets[key] = strings.TrimSpace(value)
		}
	}

	updatedMetadata := make(map[string]interface{})
	for key, value := range metadata {
		logrus.Info(fmt.Sprintf("Replace metadata: %s : value %s ", key, secrets[value.(string)]))
		updatedMetadata[key] = secrets[value.(string)]

	}
	return updatedMetadata
}

func (b *Build) getStatus(kvClient kv.KVClient) []notif.StageStatus {

	stages := []notif.StageStatus{}
	storedStages, err := b.GetStages(kvClient)

	if err != nil {
		return nil
	}

	for _, stage := range storedStages {
		stageStatus := &notif.StageStatus{}
		stageStatus.Name = stage.Name
		stageStatus.Status = stage.Status
		stages = append(stages, *stageStatus)
	}
	return stages

}
