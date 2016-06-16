package controller

import (
	"errors"
	"time"

	log "github.com/Sirupsen/logrus"

	"github.com/AcalephStorage/kontinuous/controller/handler"
	"github.com/AcalephStorage/kontinuous/model"
	"github.com/AcalephStorage/kontinuous/store/kv"
)

type PipelineController struct {
	*kv.PipelineStore
	*kv.PipelineMapStore
	ChangeHandlers []handler.Handler
}

func (pc *PipelineController) CreatePipeline(user string, pipeline *model.Pipeline) error {

	// check if pipeline exists
	_, err := pc.PipelineMapStore.GetPipelineID(pipeline.Name)
	if err == nil {
		// id exists. tsk tsk
		err = errors.New("pipeline already exists")
		log.WithError(err).Debug("unable to create duplicate pipeline")
		return err
	}

	// create pipeline (created status) and mapping
	pipeline.Status = model.PipelineCreated
	key, err := pc.PipelineStore.Create(pipeline)
	if err != nil {
		log.WithError(err).Debug("unable to create pipeline")
		return err
	}
	// update it with the key
	pipeline.ID = key
	pipeline.Creator = user
	pipeline.Created = time.Now().UnixNano()
	err = pc.PipelineStore.Update(pipeline)
	if err != nil {
		log.WithError(err).Debug("unable to update pipeline with created id")
		return err
	}
	// then add the mapping
	err = pc.PipelineMapStore.AddMapping(pipeline.Name, key)
	if err != nil {
		log.WithError(err).Debug("unable to add pipeline mapping")
	}
	return err
}

func (pc *PipelineController) GetPipeline(name string) (pipeline *model.Pipeline, err error) {
	// get pipeline id
	id, err := pc.PipelineMapStore.GetPipelineID(name)
	if err != nil {
		log.WithError(err).Debug("unable to get pipeline id")
		return
	}

	pipeline, err = pc.PipelineStore.Get(id)
	if err != nil {
		log.WithError(err).Debug("unable to get pipeline")
	}
	return
}

func (pc *PipelineController) UpdatePipeline(pipelineName string, pipeline *model.Pipeline) error {
	err := pc.PipelineStore.Update(pipeline)
	if err != nil {
		log.WithError(err).Debug("unable to update pipeline")
	}
	// change mapping if the name has changed
	if pipeline.Name != pipelineName {
		pc.PipelineMapStore.DeleteMapping(pipelineName)
		err := pc.PipelineMapStore.AddMapping(pipeline.Name, pipeline.ID)
		if err != nil {
			log.WithError(err).Debug("unable to create new mapping")
		}
	}
	return err
}

func (pc *PipelineController) DeletePipeline(name string) error {
	// get pipeline id
	id, err := pc.PipelineMapStore.GetPipelineID(name)
	if err != nil {
		log.WithError(err).Debug("unable to get pipeline id")
		return err
	}
	err = pc.PipelineStore.Delete(id)
	if err != nil {
		log.WithError(err).Debug("unable to delete pipeline")
	}
	// delete mapping
	err = pc.PipelineMapStore.DeleteMapping(name)
	if err != nil {
		log.WithError(err).Debug("unable to delete pipeline mapping")
	}
	return err
}

func (pc *PipelineController) ListPipelines() (pipelines []*model.Pipeline, err error) {
	pipelines, err = pc.PipelineStore.List()
	if err != nil {
		log.WithError(err).Debug("unable to get list of pipelines")
	}
	return
}
