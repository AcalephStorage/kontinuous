package kv

import (
	"fmt"
	"strings"

	"encoding/json"

	log "github.com/Sirupsen/logrus"

	"github.com/AcalephStorage/kontinuous/model"
)

type PipelineStore struct {
	KVClient Client
}

func (ps *PipelineStore) Create(pipeline *model.Pipeline) (key string, err error) {
	data, err := json.Marshal(pipeline)
	if err != nil {
		return
	}
	dir := "/kontinuous/pipelines"
	key, err = ps.KVClient.CreateInDir(dir, data)
	key = strings.Replace(key, dir+"/", "", -1)
	if err != nil {
		log.WithError(err).Debug("unable to create new pipeline")
		return
	}
	return
}

func (ps *PipelineStore) Update(pipeline *model.Pipeline) error {
	data, err := json.Marshal(pipeline)
	if err != nil {
		return err
	}
	key := fmt.Sprintf("/kontinuous/pipelines/%s", pipeline.ID)
	err = ps.KVClient.Update(key, data)
	if err != nil {
		log.WithError(err).Debug("unable to update pipeline")
	}
	return err
}

func (ps *PipelineStore) Get(id string) (pipeline *model.Pipeline, err error) {
	key := fmt.Sprintf("/kontinuous/pipelines/%s", id)
	data, err := ps.KVClient.Restore(key)
	if err != nil {
		log.WithError(err).Debug("unable to get pipeline data from etcd")
		return
	}
	var p model.Pipeline
	err = json.Unmarshal(data, &p)
	pipeline = &p
	if err != nil {
		log.WithError(err).Debug("unable to unmarshal pipeline data")
	}
	return
}

func (ps *PipelineStore) List() (list []*model.Pipeline, err error) {
	dir := "/kontinuous/pipelines"
	values, err := ps.KVClient.List(dir)
	if err != nil {
		log.WithError(err).Debug("unable to get directory list of pipelines")
		return
	}
	list = make([]*model.Pipeline, len(values))
	for i, value := range values {
		var pipeline model.Pipeline
		err = json.Unmarshal(value, &pipeline)
		list[i] = &pipeline
		if err != nil {
			log.WithError(err).Debug("unable to unmarshal pipeline data")
			return nil, err
		}
	}
	return
}

func (ps *PipelineStore) Delete(id string) error {
	key := fmt.Sprintf("/kontinuous/pipelines/%s", id)
	err := ps.KVClient.Delete(key)
	if err != nil {
		log.WithError(err).Debug("unable to delete pipeline data")
	}
	return err
}
