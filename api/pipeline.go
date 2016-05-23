package api

import (
	"fmt"

	"encoding/json"
	"io/ioutil"
	"net/http"

	"github.com/AcalephStorage/kontinuous/kube"
	ps "github.com/AcalephStorage/kontinuous/pipeline"
	"github.com/AcalephStorage/kontinuous/store/kv"
	"github.com/AcalephStorage/kontinuous/store/mc"
	"github.com/emicklei/go-restful"
)

// PipelineResource defines the endpoints of a Pipeline
type PipelineResource struct {
	kv.KVClient
	*mc.MinioClient
	kube.KubeClient
}

// Register registers the endpoint of this resource to the container
func (p *PipelineResource) Register(container *restful.Container) {
	ws := new(restful.WebService)

	ws.
		Path("/api/v1/pipelines").
		Consumes(restful.MIME_JSON).
		Produces(restful.MIME_JSON).
		Doc("manage pipelines").
		Produces(restful.MIME_JSON).
		Filter(ncsaCommonLogFormatLogger)

	ws.Route(ws.GET("").To(p.list).
		Doc("Get all pipelines").
		Operation("list").
		Writes([]ps.Pipeline{}).
		Filter(authenticate).
		Filter(requireAccessToken))

	ws.Route(ws.POST("").To(p.create).
		Doc("Create new pipeline").
		Operation("create").
		Reads(ps.Pipeline{}).
		Writes(ps.Pipeline{}).
		Filter(authenticate).
		Filter(requireAccessToken))

	ws.Route(ws.POST("/login").To(p.login).
		Doc("Save SCM user details").
		Operation("login").
		Reads(ps.User{}).
		Writes(ps.User{}).
		Operation("login").
		Filter(authenticate))

	ws.Route(ws.GET("/{owner}/{repo}").To(p.show).
		Doc("Show pipeline details").
		Operation("show").
		Param(ws.PathParameter("owner", "repository owner name").DataType("string")).
		Param(ws.PathParameter("repo", "repository name").DataType("string")).
		Writes(ps.Pipeline{}).
		Filter(authenticate).
		Filter(requireAccessToken))

	ws.Route(ws.DELETE("/{owner}/{repo}").To(p.delete).
		Doc("Delete pipeline").
		Operation("delete").
		Param(ws.PathParameter("owner", "repository owner name").DataType("string")).
		Param(ws.PathParameter("repo", "repository name").DataType("string")).
		Writes(ps.Pipeline{}).
		Filter(authenticate).
		Filter(requireAccessToken))

	ws.Route(ws.GET("/{owner}/{repo}/definition").To(p.definition).
		Doc("Get pipeline details of the repository").
		Operation("definition").
		Param(ws.PathParameter("owner", "repository owner name").DataType("string")).
		Param(ws.PathParameter("repo", "repository name").DataType("string")).
		Writes(ps.DefinitionFile{}).
		Filter(requireAccessToken))

	ws.Route(ws.GET("/{owner}/{repo}/definition/{ref}").To(p.definition).
		Doc("Get pipeline details of the repository").
		Operation("definition").
		Param(ws.PathParameter("owner", "repository owner name").DataType("string")).
		Param(ws.PathParameter("repo", "repository name").DataType("string")).
		Param(ws.PathParameter("ref", "commit or branch").DataType("string")).
		Writes(ps.DefinitionFile{}).
		Filter(requireAccessToken))

	ws.Route(ws.POST("/{owner}/{repo}/definition").To(p.updateDefinition).
		Doc("Update definition file of the pipeline, creates one if it does not exist").
		Operation("updateDefinition").
		Param(ws.PathParameter("owner", "repository owner name").DataType("string")).
		Param(ws.PathParameter("repo", "repository name").DataType("string")).
		Writes(ps.DefinitionFile{}).
		Filter(requireAccessToken))

	buildResource := &BuildResource{
		KVClient:    p.KVClient,
		MinioClient: p.MinioClient,
	}
	stageResource := &StageResource{
		KVClient:    p.KVClient,
		MinioClient: p.MinioClient,
		KubeClient:  p.KubeClient,
	}

	buildResource.extend(ws)
	stageResource.extend(ws)
	container.Add(ws)
}

func (p *PipelineResource) create(req *restful.Request, res *restful.Response) {
	client := newSCMClient(req)
	pipeline := new(ps.Pipeline)

	if err := req.ReadEntity(pipeline); err != nil {
		jsonError(res, http.StatusInternalServerError, err, "Unable to readline pipeline from request")
		return
	}

	// save user token if not saved already (for remote access)
	if _, exists := ps.FindUser(pipeline.Login, p.KVClient); !exists {
		u := &ps.User{
			Name:     pipeline.Login,
			RemoteID: pipeline.Login,
			Token:    req.HeaderParameter("Authorization"),
		}

		if err := u.Save(p.KVClient); err != nil {
			jsonError(res, http.StatusInternalServerError, err, "Unable to save user details")
			return
		}
	}

	if err := ps.CreatePipeline(pipeline, client, p.KVClient); err != nil {
		jsonError(res, 422, err, "Unable to create pipeline")
		return
	}

	res.WriteHeaderAndEntity(http.StatusCreated, pipeline)
}

func (p *PipelineResource) delete(req *restful.Request, res *restful.Response) {
	owner := req.PathParameter("owner")
	repo := req.PathParameter("repo")
	pipeline, err := findPipeline(owner, repo, p.KVClient)
	if err != nil {
		jsonError(res, http.StatusNotFound, err, fmt.Sprintf("Unable to find pipeline %s/%s", owner, repo))
		return
	}

	if err := pipeline.DeletePipeline(p.KVClient, p.MinioClient); err != nil {
		jsonError(res, http.StatusInternalServerError, err, fmt.Sprintf("Unable to delete pipeline %s/%s", owner, repo))
		return
	}

	res.WriteHeader(http.StatusOK)

}

func (p *PipelineResource) list(req *restful.Request, res *restful.Response) {
	pipelines, err := ps.FindAllPipelines(p.KVClient)
	if err != nil {
		jsonError(res, http.StatusInternalServerError, err, "Unable to list pipelines")
		return
	}

	res.WriteEntity(pipelines)
}

func (p *PipelineResource) show(req *restful.Request, res *restful.Response) {
	owner := req.PathParameter("owner")
	repo := req.PathParameter("repo")
	pipeline, err := findPipeline(owner, repo, p.KVClient)
	if err != nil {
		jsonError(res, http.StatusNotFound, err, fmt.Sprintf("Unable to find pipeline %s/%s", owner, repo))
		return
	}

	res.WriteEntity(pipeline)
}

func (p *PipelineResource) login(req *restful.Request, res *restful.Response) {
	user := new(ps.User)
	if err := req.ReadEntity(user); err != nil {
		jsonError(res, http.StatusInternalServerError, err, "Unable to read user details from request")
		return
	}

	if err := user.Save(p.KVClient); err != nil {
		jsonError(res, http.StatusInternalServerError, err, "Unable to save user details")
		return
	}

	res.WriteHeaderAndEntity(http.StatusCreated, user)
}

func (p *PipelineResource) definition(req *restful.Request, res *restful.Response) {
	client := newSCMClient(req)
	owner := req.PathParameter("owner")
	repo := req.PathParameter("repo")
	ref := req.PathParameter("ref")

	pipeline, err := findPipeline(owner, repo, p.KVClient)
	if err != nil {
		jsonError(res, http.StatusNotFound, err, fmt.Sprintf("Unable to find pipeline %s/%s", owner, repo))
		return
	}

	file, exists := pipeline.GetDefinitionFile(client, ref)
	if !exists {
		err = fmt.Errorf("Definition file for %s/%s not found.", owner, repo)
		jsonError(res, http.StatusNotFound, err, fmt.Sprintf("Unable to fetch definition for %s/%s", owner, repo))
		return
	}

	res.WriteAsJson(file)
}

func (p *PipelineResource) updateDefinition(req *restful.Request, res *restful.Response) {
	owner := req.PathParameter("owner")
	repo := req.PathParameter("repo")

	pipeline, err := findPipeline(owner, repo, p.KVClient)
	if err != nil {
		jsonError(res, http.StatusNotFound, err, fmt.Sprintf("Unable to find pipeline %s/%s", owner, repo))
		return
	}

	client := newSCMClient(req)
	body, _ := ioutil.ReadAll(req.Request.Body)
	payload := new(struct {
		Definition *ps.DefinitionFile `json:"definition"`
		Commit     map[string]string  `json:"commit"`
	})
	if err := json.Unmarshal(body, &payload); err != nil {
		jsonError(res, http.StatusInternalServerError, err, "Unable to read request payload")
		return
	}

	file, err := pipeline.UpdateDefinitionFile(client, payload.Definition, payload.Commit)
	if err != nil {
		jsonError(res, http.StatusInternalServerError, err, fmt.Sprintf("Unable to update definition file for %s/%s", pipeline.Owner, pipeline.Repo))
		return
	}
	res.WriteAsJson(file)
}
