package controller

import (
	"errors"
	"github.com/AcalephStorage/kontinuous/model"
)

type RepositoryController struct {
	*UserController
}

func (rc *RepositoryController) List(user string, repoType model.RepositoryType) (repos []*model.Repository, err error) {
	switch repoType {
	case model.GithubRepository:
		return rc.listGithubRepositories(user)
	default:
		err = errors.New("unsuported scm")
		return
	}
}

func (rc *RepositoryController) listGithubRepositories(user string) (repos []*model.Repository, err error) {

	return
}
