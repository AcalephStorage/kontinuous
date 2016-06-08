package controller

import (
	log "github.com/Sirupsen/logrus"

	"github.com/AcalephStorage/kontinuous/model"
	"github.com/AcalephStorage/kontinuous/store/kv"
)

type UserController struct {
	*kv.UserStore
	*kv.UserMapStore
}

func (uc *UserController) SaveUser(userType model.UserType, username string, user *model.User) error {
	// save user
	if err := uc.UserStore.SaveUser(user); err != nil {
		log.WithError(err).Debug("unable to save user details")
		return err
	}
	// save user-map
	if err := uc.UserMapStore.AddMapping(userType, username, user.UUID); err != nil {
		log.WithError(err).Debug("unable to save user mapping")
		return err
	}
	return nil
}

func (uc *UserController) GetUser(userType model.UserType, username string) (*model.User, error) {
	// get user uuid from user-map
	uuid, err := uc.UserMapStore.FindUUID(userType, username)
	if err != nil {
		log.WithError(err).Debug("unable to find UUID for user")
		return nil, err
	}

	user, err := uc.UserStore.GetUser(uuid)
	if err != nil {
		log.WithError(err).Debug("unable to get user")
		return nil, err
	}
	return user, nil
}
