package api

import (
	"fmt"

	"net/http"

	"github.com/AcalephStorage/kontinuous/controller"
	"github.com/AcalephStorage/kontinuous/model"
	"github.com/AcalephStorage/kontinuous/scm"
	"github.com/emicklei/go-restful"
)

// RepositoryResource defines the endpoints to access the git repositories
type RepositoryResource struct {
	*AuthFilter
	*controller.RepositoryController
}

// Register registers the endpoints to the container
func (r *RepositoryResource) Register(container *restful.Container) {
	ws := new(restful.WebService)

	ws.
		Path("/api/v1/repositories").
		Doc("manage repositories").
		Filter(r.AuthFilter.requireBearerToken).
		Filter(requestLogger)

	// ws.Route(ws.GET("").To(list).
	// 	Doc("Get all github repositories for the user").
	// 	Operation("list").
	// 	Produces(restful.MIME_JSON).
	// 	Param(ws.QueryParameter("repo", "the repository type")).
	// 	Writes([]model.Repository))

	// ws.Route(ws.GET("").To(r.list).
	// 	Doc("Get all repositories accessible by the current user").
	// 	Operation("list").
	// 	Writes([]scm.Repository{}))

	// ws.Route(ws.GET("/{owner}/{name}").To(r.show).
	// 	Doc("Get repository details").
	// 	Operation("show").
	// 	Param(ws.PathParameter("owner", "repository owner name").DataType("string")).
	// 	Param(ws.PathParameter("name", "repository name").DataType("string")).
	// 	Writes(scm.Repository{}))

	container.Add(ws)
}

func (rr *RepositoryResource) list(req *restful.Request, res *restful.Response) {
	user := req.Attribute("user_id").(string)
	repoType := req.QueryParameter("repo")

	repos, err := rr.RepositoryController.List(user, model.RepositoryType(repoType))
	if err != nil {
		jsonError(res, http.StatusBadRequest, err, "unable to get list of repositories")
		return
	}
	res.WriteEntity(repos)
}

func (r *RepositoryResource) show(req *restful.Request, res *restful.Response) {
	client := newSCMClient(req)
	owner := req.PathParameter("owner")
	name := req.PathParameter("name")
	repo, ok := client.GetRepository(owner, name)
	if !ok {
		jsonError(res, http.StatusNotFound, fmt.Errorf("Repository %s/%s not found.", owner, repo), "Unable to find repo")
		return
	}

	res.WriteEntity(repo)
}
