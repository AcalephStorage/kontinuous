package handler

import (
	"encoding/json"
	"github.com/AcalephStorage/kontinuous/model"
)

// creates deploy keys
// creates
type PipelineGithubHandler struct {
}

func (pgh *PipelineGithubHandler) OnResourceChange(action ResourceAction, key string, value []byte) error {
	pipeline := new(model.Pipeline)
	if err := json.Unmarshal(value, pipeline); err != nil {
		return err
	}
	switch action {
	case Set:
		fallthrough
	case Update:
		return pgh.pipelineUpdated(pipeline)
	case Delete:
		return pgh.pipelineDeleted(pipeline)
	default:
		// ignore the rest
		return nil
	}
}

func (pgh *PipelineGithubHandler) pipelineUpdated(pipeline *model.Pipeline) error {

	return nil
}

func (pgh *PipelineGithubHandler) pipelineDeleted(pipeline *model.Pipeline) error {

	return nil
}
