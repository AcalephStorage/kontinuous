package kv

import (
	"fmt"

	"encoding/json"

	log "github.com/Sirupsen/logrus"

	"github.com/AcalephStorage/kontinuous/model"
)

type UserStore struct {
	KVClient Client
}

func (us *UserStore) SaveUser(user *model.User) error {
	key := fmt.Sprintf("/kontinuous/users/%s", user.UUID)
	data, err := json.Marshal(user)
	if err != nil {
		log.WithError(err).Debug("unable to json marshal user data")
		return err
	}
	if err := us.KVClient.Put(key, string(data)); err != nil {
		log.WithError(err).Debug("unable to save user data to etcd")
		return err
	}
	return nil
}

func (us *UserStore) GetUser(uuid string) (*model.User, error) {
	key := fmt.Sprintf("/kontinuous/users/%s", uuid)
	data, err := us.KVClient.Get(key)
	if err != nil {
		log.WithError(err).Debug("unable to get user from etcd")
		return nil, err
	}
	user := &model.User{}
	if err := json.Unmarshal([]byte(data), user); err != nil {
		log.WithError(err).Debug("unable to unmarshal user data")
		return nil, err
	}
	return user, nil
}
