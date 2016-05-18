package pipeline

import (
	"errors"

	"encoding/json"

	"github.com/ghodss/yaml"
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
