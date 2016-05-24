package pipeline

import (
	"errors"
	"fmt"
	"time"

	"encoding/base64"
	"encoding/json"

	"github.com/ghodss/yaml"

	"github.com/AcalephStorage/kontinuous/scm"
)

type (
	SpecDetails struct {
		Selector struct {
			MatchLabels map[string]interface{} `json:"matchLabels"`
		} `json:"selector"`
		Template TemplateDetails `json:"template"`
	}

	TemplateDetails struct {
		Metadata  map[string]interface{} `json:"metadata"`
		Stages    []Stage                `json:"stages"`
		Notifiers []*Notifier            `json:"notif,omitempty"`
		Secrets   []string               `json:"secrets,omitempty"`
		Vars      map[string]interface{} `json:"vars,omitempty"`
	}
)

type Definition struct {
	APIVersion string                 `json:"apiVersion"`
	Kind       string                 `json:"kind"`
	Metadata   map[string]interface{} `json:"metadata"`
	Spec       SpecDetails            `json:"spec"`
}

// DefinitionFile holds repository metadata of the definition file (PipelineYAML)
type DefinitionFile struct {
	Content *string `json:"content,omitempty"`
	SHA     *string `json:"sha"`
}

func (d *Definition) GetStages() []*Stage {
	stages := make([]*Stage, len(d.Spec.Template.Stages))

	for i := range d.Spec.Template.Stages {
		stages[i] = &d.Spec.Template.Stages[i]
	}

	return stages
}

func (d *DefinitionFile) SaveToRepo(c scm.Client, owner, repo string, commit map[string]string) (*DefinitionFile, error) {
	source, exists := c.GetRepository(owner, repo)
	if !exists {
		return nil, fmt.Errorf("Unable to find repository %s/%s", owner, repo)
	}
	defaultBranch := source.DefaultBranch
	branch := defaultBranch

	if commit["option"] == "pull_request" {
		branch = commit["branch_name"]
		if len(branch) == 0 {
			return nil, fmt.Errorf("Branch name not provided")
		}

		head, err := c.GetHead(owner, repo, defaultBranch)
		if err != nil {
			return nil, err
		}
		_, err = c.CreateBranch(owner, repo, branch, head)
		if err != nil {
			return nil, err
		}
		// wait for branch to be created (todo: verify existence of branch)
		time.Sleep(3 * time.Second)
	}

	decodedContent, err := base64.URLEncoding.DecodeString(*d.Content)
	if err != nil {
		return nil, err
	}

	file := &scm.RepositoryContent{}
	if d.SHA != nil {
		if len(commit["message"]) == 0 {
			commit["message"] = fmt.Sprintf("Update %s", PipelineYAML)
		}
		file, err = c.UpdateFile(owner, repo, PipelineYAML, *d.SHA, commit["message"], branch, decodedContent)
	} else {
		if len(commit["message"]) == 0 {
			commit["message"] = fmt.Sprintf("Create %s", PipelineYAML)
		}
		file, err = c.CreateFile(owner, repo, PipelineYAML, commit["message"], branch, decodedContent)
	}
	if err != nil {
		return nil, err
	}

	if commit["option"] == "pull_request" {
		err = c.CreatePullRequest(owner, repo, defaultBranch, branch, commit["message"])
		if err != nil {
			return nil, err
		}
	}

	return &DefinitionFile{SHA: file.SHA}, nil
}

func GetDefinition(definition []byte) (payload *Definition, err error) {

	if len(definition) == 0 {
		return nil, errors.New("Empty YAML file")
	}

	data, err := yaml.YAMLToJSON(definition)
	if err = json.Unmarshal(data, &payload); err != nil {
		return nil, err
	}

	namespace := "default"
	if payload.Metadata["namespace"] != nil {
		namespace = payload.Metadata["namespace"].(string)
	}

	for idx := range payload.Spec.Template.Stages {
		payload.Spec.Template.Stages[idx].Namespace = namespace
	}

	return payload, nil
}
