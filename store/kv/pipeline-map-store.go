package kv

import (
	"fmt"

	log "github.com/Sirupsen/logrus"
)

type PipelineMapStore struct {
	KVClient Client
}

func (pms *PipelineMapStore) AddMapping(name, id string) error {
	key := fmt.Sprintf("/kontinuous/pipeline-map/%s", name)
	if err := pms.KVClient.Update(key, []byte(id)); err != nil {
		log.WithError(err).Debug("unable to save new mapping")
		return err
	}
	return nil
}

func (pms *PipelineMapStore) GetPipelineID(name string) (id string, err error) {
	key := fmt.Sprintf("/kontinuous/pipeline-map/%s", name)
	idRaw, err := pms.KVClient.Restore(key)
	if err != nil {
		log.WithError(err).Debug("unable to find ID from etcd")
	}
	id = string(idRaw)
	return
}

func (pms *PipelineMapStore) DeleteMapping(name string) error {
	key := fmt.Sprintf("/kontinuous/pipeline-map/%s", name)
	err := pms.KVClient.Delete(key)
	if err != nil {
		log.WithError(err).Debug("unable to delete pipeline mapping")
	}
	return err
}
