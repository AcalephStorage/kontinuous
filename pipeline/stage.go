package pipeline

import (
	"fmt"
	"strconv"

	"encoding/json"

	etcd "github.com/coreos/etcd/client"

	"github.com/AcalephStorage/kontinuous/kube"
	"github.com/AcalephStorage/kontinuous/scm"
	"github.com/AcalephStorage/kontinuous/store/kv"
)

type (
	// StatusUpdate contains details for stage patch updates
	StatusUpdate struct {
		Status      string `json:"status"`
		JobName     string `json:"job_name"`
		PodName     string `json:"pod_name"`
		Timestamp   int64  `json:"timestamp"`
		DockerImage string `json:"docker_image"`
	}

	// JobBuildInfo contains the required details for creating a job
	JobBuildInfo struct {
		PipelineUUID string `json:"pipeline_uuid"`
		Build        string `json:"build"`
		Stage        string `json:"stage"`
		Commit       string `json:"commit"`
		Branch       string `json:"branch"`
		User         string `json:"user,omitempty"`
		Repo         string `json:"repo,omitempty"`
		Owner        string `json:"owner,omitempty"`
	}
)

// Stage contains the current state of a job
type Stage struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Type        string                 `json:"type"`
	Index       int                    `json:"index"`
	Params      map[string]interface{} `json:"params"`
	Labels      map[string]interface{} `json:"labels"`
	Started     int64                  `json:"started,omitempty"`
	Finished    int64                  `json:"finished,omitempty"`
	Message     string                 `json:"message,omitempty"`
	Status      string                 `json:"status"`
	Namespace   string                 `json:"pod_namespace"`
	JobName     string                 `json:"job_name,omitempty"`
	PodName     string                 `json:"pod_name,omitempty"`
	DockerImage string                 `json:"docker_image,omitempty"`
	Artifacts   []string               `json:"artifacts,omitempty"`
	Vars        map[string]interface{} `json:"vars,omitempty"`
}

func getStage(path string, kvClient kv.KVClient) *Stage {
	s := new(Stage)
	started, _ := kvClient.Get(path + "/started")
	finished, _ := kvClient.Get(path + "/finished")
	params, _ := kvClient.Get(path + "/params")
	labels, _ := kvClient.Get(path + "/labels")
	vars, _ := kvClient.Get(path + "/vars")

	s.ID, _ = kvClient.Get(path + "/uuid")
	s.Index, _ = kvClient.GetInt(path + "/index")
	s.Name, _ = kvClient.Get(path + "/name")
	s.Type, _ = kvClient.Get(path + "/type")
	s.DockerImage, _ = kvClient.Get(path + "/docker-image")
	s.PodName, _ = kvClient.Get(path + "/pod-name")
	s.JobName, _ = kvClient.Get(path + "/job-name")
	s.Namespace, _ = kvClient.Get(path + "/namespace")
	s.Status, _ = kvClient.Get(path + "/status")
	s.Message, _ = kvClient.Get(path + "/message")
	s.Started, _ = strconv.ParseInt(started, 10, 64)
	s.Finished, _ = strconv.ParseInt(finished, 10, 64)

	json.Unmarshal([]byte(params), &s.Params)
	json.Unmarshal([]byte(labels), &s.Labels)
	json.Unmarshal([]byte(vars), &s.Vars)

	return s
}

// Save persists the stage details to `etcd`
func (s *Stage) Save(namespace string, kvClient kv.KVClient) (err error) {
	stagePrefix := namespace + "/" + strconv.Itoa(s.Index)
	isNew := false

	_, err = kvClient.GetDir(stagePrefix)
	if err != nil || etcd.IsKeyNotFound(err) {
		isNew = true
	}

	if err = kvClient.Put(stagePrefix+"/uuid", s.ID); err != nil {
		return handleSaveError(stagePrefix, isNew, err, kvClient)
	}
	if err = kvClient.Put(stagePrefix+"/name", s.Name); err != nil {
		return handleSaveError(stagePrefix, isNew, err, kvClient)
	}
	if err = kvClient.Put(stagePrefix+"/type", s.Type); err != nil {
		return handleSaveError(stagePrefix, isNew, err, kvClient)
	}
	if err = kvClient.Put(stagePrefix+"/status", s.Status); err != nil {
		return handleSaveError(stagePrefix, isNew, err, kvClient)
	}
	if err = kvClient.Put(stagePrefix+"/namespace", s.Namespace); err != nil {
		return handleSaveError(stagePrefix, isNew, err, kvClient)
	}
	params, _ := json.Marshal(s.Params)
	if err = kvClient.Put(stagePrefix+"/params", string(params)); err != nil {
		return handleSaveError(stagePrefix, isNew, err, kvClient)
	}
	labels, _ := json.Marshal(s.Labels)
	if err = kvClient.Put(stagePrefix+"/labels", string(labels)); err != nil {
		return handleSaveError(stagePrefix, isNew, err, kvClient)
	}
	vars, _ := json.Marshal(s.Vars)
	if err = kvClient.Put(stagePrefix+"/vars", string(vars)); err != nil {
		return handleSaveError(stagePrefix, isNew, err, kvClient)
	}

	if err = kvClient.PutInt(stagePrefix+"/index", s.Index); err != nil {
		return handleSaveError(stagePrefix, isNew, err, kvClient)
	}
	if err = kvClient.Put(stagePrefix+"/pod-name", s.PodName); err != nil {
		kvClient.DeleteTree(namespace)
		return err
	}
	if err = kvClient.Put(stagePrefix+"/job-name", s.JobName); err != nil {
		kvClient.DeleteTree(namespace)
		return err
	}
	if err = kvClient.Put(stagePrefix+"/docker-image", s.DockerImage); err != nil {
		kvClient.DeleteTree(namespace)
		return err
	}
	if err = kvClient.Put(stagePrefix+"/started", strconv.FormatInt(s.Started, 10)); err != nil {
		kvClient.DeleteTree(namespace)
		return err
	}
	if err = kvClient.Put(stagePrefix+"/finished", strconv.FormatInt(s.Finished, 10)); err != nil {
		kvClient.DeleteTree(namespace)
		return err
	}

	return nil
}

func (s *Stage) Deploy(p *Pipeline, b *Build, c scm.Client) error {

	deployFile := fmt.Sprintf("%v", s.Params["deploy_file"])
	valueMap := p.Vars

	for key, varVal := range s.Vars {
		valueMap[key] = varVal
	}

	ref := b.Commit
	if ref == "" {
		ref = b.Branch
	}

	file, ok := c.GetFileContent(p.Owner, p.Repo, deployFile, b.Commit)

	if !ok {
		return fmt.Errorf("%s not found for %s/%s on %s",
			deployFile,
			p.Owner,
			p.Repo,
			ref)
	}

	kubeClient, _ := kube.NewClient("https://kubernetes.default")
	if err := kubeClient.DeployResourceFile(file, valueMap); err != nil {
		return err
	}

	return nil
}

// UpdateStatus updates the status of a build stage
func (s *Stage) UpdateStatus(u *StatusUpdate, p *Pipeline, b *Build, kv kv.KVClient, c scm.Client) (*Stage, error) {
	var scmStatus string
	// only update build status when job is running or has failed
	// update success only if this is the last stage
	switch u.Status {
	case BuildRunning:
		b.Status = BuildRunning
		s.Started = u.Timestamp
		scmStatus = scm.StatePending
		if s.Index == 1 {
			b.Started = u.Timestamp
		}
	case BuildSuccess:
		s.Finished = u.Timestamp
		scmStatus = scm.StateSuccess
	case BuildFailure:
		b.Status = BuildFailure
		b.Finished = u.Timestamp
		s.Finished = u.Timestamp
		scmStatus = scm.StateFailure
	}

	s.Status = u.Status
	s.DockerImage = u.DockerImage
	s.JobName = u.JobName
	s.PodName = u.PodName

	// ideally only build will be saved, which will also update the stage details
	namespace := fmt.Sprintf("%s%s/builds/%d/stages", pipelineNamespace, b.Pipeline, b.Number)
	if err := s.Save(namespace, kv); err != nil {
		return nil, err
	}

	if err := b.Save(kv); err != nil {
		return nil, err
	}

	if b.Branch != b.Commit {
		if err := c.CreateStatus(p.Owner, p.Repo, b.Commit, s.Index, s.Name, scmStatus); err != nil {
			return nil, err
		}
	}

	if s.Status == BuildSuccess {
		// trigger next stage if available
		nextIdx := s.Index + 1
		var nextStage *Stage

		if nextStage, _ = b.GetStage(nextIdx, kv); nextStage != nil {
			b.CurrentStage = nextIdx
		} else {
			// update build to finished if stage doesn't have a successor
			b.Status = BuildSuccess
			b.Finished = u.Timestamp
		}

		if err := b.Save(kv); err != nil {
			return nil, err
		}

		if nextStage != nil {
			return nextStage, nil
		}
	}

	if b.Finished != 0 {
		err := b.Notify(kv)
		if err != nil {
			return nil, err
		}
	}

	return nil, nil
}
