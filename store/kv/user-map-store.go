package kv

import (
	"fmt"

	log "github.com/Sirupsen/logrus"

	"github.com/AcalephStorage/kontinuous/model"
)

type UserMapStore struct {
	KVClient Client
}

func (ums *UserMapStore) AddMapping(userType model.UserType, username, uuid string) error {
	key := fmt.Sprintf("/kontinuous/user-map/%s/%s", userType, username)
	if err := ums.KVClient.Put(key, uuid); err != nil {
		log.WithError(err).Debug("unable to save new mapping")
		return err
	}
	return nil
}

func (ums *UserMapStore) GetUserID(userType model.UserType, username string) (uuid string, err error) {
	key := fmt.Sprintf("/kontinuous/user-map/%s/%s", userType, username)
	uuid, err = ums.KVClient.Get(key)
	if err != nil {
		log.WithError(err).Debug("unable to find UUID from etcd")
	}
	return
}
