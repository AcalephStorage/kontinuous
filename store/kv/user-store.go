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

func (us *UserStore) Save(user *model.User) error {
	data, err := json.Marshal(user)
	if err != nil {
		log.WithError(err).Debug("unable to json marshal user data")
		return err
	}
	key := fmt.Sprintf("/kontinuous/users/%s", user.User)
	err = us.KVClient.Update(key, data)
	if err != nil {
		log.WithError(err).Debug("unable to save user data to etcd")
	}
	return err
}

func (us *UserStore) Delete(userID string) error {
	key := fmt.Sprintf("/kontinuous/users/%s", userID)
	err := us.KVClient.Delete(key)
	if err != nil {
		log.WithError(err).Debug("unable to delete user data")
	}
	return err
}

func (us *UserStore) GetUser(userID string) (*model.User, error) {
	key := fmt.Sprintf("/kontinuous/users/%s", userID)
	data, err := us.KVClient.Restore(key)
	if err != nil {
		log.WithError(err).Debug("unable to get user from etcd")
		return nil, err
	}
	user := &model.User{}
	if err := json.Unmarshal(data, user); err != nil {
		log.WithError(err).Debug("unable to unmarshal user data")
		return nil, err
	}
	return user, nil
}
