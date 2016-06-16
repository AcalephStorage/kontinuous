package api

import (
	"net/http"

	log "github.com/Sirupsen/logrus"
	"github.com/emicklei/go-restful"

	"github.com/AcalephStorage/kontinuous/controller"
	"github.com/AcalephStorage/kontinuous/model"
)

// PipelineResource defines the endpoints of a Pipeline
type PipelineResource struct {
	*AuthFilter
	*controller.PipelineController
}

// Register registers the endpoint of this resource to the container
func (p *PipelineResource) Register(container *restful.Container) {

	ws := new(restful.WebService)
	ws.
		Path("/api/v1/pipelines").
		Doc("manage pipelines")

	// -- GET /api/v1/pipelines
	ws.Route(ws.GET("").To(p.list).
		Doc("Get all pipelines").
		Operation("list").
		Produces(restful.MIME_JSON).
		Writes([]model.Pipeline{}).
		Filter(p.AuthFilter.requireBearerToken).
		Filter(requestLogger))

	// -- POST /api/v1/pipelines
	ws.Route(ws.POST("").To(p.create).
		Doc("Create new pipeline").
		Operation("create").
		Consumes(restful.MIME_JSON).
		Reads(model.Pipeline{}).
		Filter(p.AuthFilter.requireBearerToken).
		Filter(requestLogger))

	// -- GET /api/v1/pipelines/{pipelineName}
	ws.Route(ws.GET("/{pipelineName}").To(p.show).
		Doc("Show pipeline details").
		Operation("show").
		Produces(restful.MIME_JSON).
		Param(ws.PathParameter("pipelineName", "pipeline name").DataType("string")).
		Writes(model.Pipeline{}).
		Filter(p.AuthFilter.requireBearerToken).
		Filter(requestLogger))

	// -- POST /api/v1/pipelines/{pipelineName}
	ws.Route(ws.POST("/{pipelineName}").To(p.update).
		Doc("update pipeline details").
		Operation("update").
		Consumes(restful.MIME_JSON).
		Param(ws.PathParameter("pipelineName", "pipeline name").DataType("string")).
		Reads(model.Pipeline{}).
		Filter(p.AuthFilter.requireBearerToken).
		Filter(requestLogger))

	// -- DELETE /api/v1/pipelines/{pipelineName}
	ws.Route(ws.DELETE("/{pipelineName}").To(p.delete).
		Doc("Delete pipeline").
		Operation("delete").
		Param(ws.PathParameter("pipelineName", "pipeline name").DataType("string")).
		Filter(p.AuthFilter.requireBearerToken).
		Filter(requestLogger))

	// // -- GET /apiFIXME: i don't we need this???
	// ws.Route(ws.GET("/{owner}/{repo}/definition").To(p.definition).
	// 	Doc("Get pipeline details of the repository").
	// 	Operation("definition").
	// 	Param(ws.PathParameter("owner", "repository owner name").DataType("string")).
	// 	Param(ws.PathParameter("repo", "repository name").DataType("string")).
	// 	Writes(ps.DefinitionFile{}))
	// // FIXME: fix filters
	// // Writes(ps.DefinitionFile{}).
	// // Filter(requireAccessToken))

	// ws.Route(ws.GET("/{owner}/{repo}/definition/{ref}").To(p.definition).
	// 	Doc("Get pipeline details of the repository").
	// 	Operation("definition").
	// 	Param(ws.PathParameter("owner", "repository owner name").DataType("string")).
	// 	Param(ws.PathParameter("repo", "repository name").DataType("string")).
	// 	Param(ws.PathParameter("ref", "commit or branch").DataType("string")).
	// 	Writes(ps.DefinitionFile{}))
	// // FIXME: fix filters
	// // Writes(ps.DefinitionFile{}).
	// // Filter(requireAccessToken))

	// ws.Route(ws.POST("/{owner}/{repo}/definition").To(p.updateDefinition).
	// 	Doc("Update definition file of the pipeline, creates one if it does not exist").
	// 	Operation("updateDefinition").
	// 	Param(ws.PathParameter("owner", "repository owner name").DataType("string")).
	// 	Param(ws.PathParameter("repo", "repository name").DataType("string")).
	// 	Writes(ps.DefinitionFile{}))
	// // FIXME: fix filters
	// // Writes(ps.DefinitionFile{}).
	// // Filter(requireAccessToken))

	// FIXME -- how to do this?
	// buildResource := &BuildResource{
	// 	Client:      p.Client,
	// 	MinioClient: p.MinioClient,
	// }
	// stageResource := &StageResource{
	// 	Client:      p.Client,
	// 	MinioClient: p.MinioClient,
	// 	KubeClient:  p.KubeClient,
	// }

	// buildResource.extend(ws)
	// stageResource.extend(ws)
	container.Add(ws)
}

// -- GET /api/v1/pipelines
func (p *PipelineResource) list(req *restful.Request, res *restful.Response) {
	log.Info("get pipelines")
	pipelines, err := p.PipelineController.ListPipelines()
	if err != nil {
		jsonError(res, http.StatusBadRequest, err, "Unable to list pipelines")
		return
	}
	res.WriteEntity(pipelines)
	log.Info("returned pipelines")
}

// -- POST /api/v1/pipelines
func (p *PipelineResource) create(req *restful.Request, res *restful.Response) {
	log.Info("create pipeline requested")

	user := req.Attribute("user_id").(string)
	pipeline := &model.Pipeline{}
	if err := req.ReadEntity(pipeline); err != nil {
		jsonError(res, http.StatusBadRequest, err, "invalid create pipeline data")
		return
	}

	if err := p.PipelineController.CreatePipeline(user, pipeline); err != nil {
		jsonError(res, http.StatusInternalServerError, err, "unable to create pipeline")
		return
	}

	log.Info("pipeline created")
}

// GET /api/v1/pipelines/{pipelineName}
func (p *PipelineResource) show(req *restful.Request, res *restful.Response) {
	log.Info("get pipeline requested")
	pipelineName := req.PathParameter("pipelineName")

	pipeline, err := p.PipelineController.GetPipeline(pipelineName)
	if err != nil {
		jsonError(res, http.StatusBadRequest, err, "unable to get pipeline")
		return
	}
	log.Info("pipeline retrieved")
	res.WriteEntity(pipeline)
}

func (p *PipelineResource) update(req *restful.Request, res *restful.Response) {
	log.Info("update pipeline request")
	pipelineName := req.PathParameter("pipelineName")
	pipeline := &model.Pipeline{}
	if err := req.ReadEntity(pipeline); err != nil {
		jsonError(res, http.StatusBadRequest, err, "invalid create pipeline data")
		return
	}

	err := p.PipelineController.UpdatePipeline(pipelineName, pipeline)
	if err != nil {
		jsonError(res, http.StatusBadRequest, err, "unable to update pipeline")
	}
	log.Info("pipeline updated")
}

func (p *PipelineResource) delete(req *restful.Request, res *restful.Response) {
	log.Info("delete pipeline request")
	pipelineName := req.PathParameter("pipelineName")

	err := p.PipelineController.DeletePipeline(pipelineName)
	if err != nil {
		jsonError(res, http.StatusBadRequest, err, "unable to delete pipeline")
		return
	}
	log.Info("pipeline deleted")
}

// func (p *PipelineResource) definition(req *restful.Request, res *restful.Response) {
// 	client := newSCMClient(req)
// 	owner := req.PathParameter("owner")
// 	repo := req.PathParameter("repo")
// 	ref := req.PathParameter("ref")

// 	pipeline, err := findPipeline(owner, repo, p.Client)
// 	if err != nil {
// 		jsonError(res, http.StatusNotFound, err, fmt.Sprintf("Unable to find pipeline %s/%s", owner, repo))
// 		return
// 	}

// 	file, exists := pipeline.GetDefinitionFile(client, ref)
// 	if !exists {
// 		err = fmt.Errorf("Definition file for %s/%s not found.", owner, repo)
// 		jsonError(res, http.StatusNotFound, err, fmt.Sprintf("Unable to fetch definition for %s/%s", owner, repo))
// 		return
// 	}

// 	res.WriteAsJson(file)
// }

// func (p *PipelineResource) updateDefinition(req *restful.Request, res *restful.Response) {
// 	owner := req.PathParameter("owner")
// 	repo := req.PathParameter("repo")

// 	pipeline, err := findPipeline(owner, repo, p.Client)
// 	if err != nil {
// 		jsonError(res, http.StatusNotFound, err, fmt.Sprintf("Unable to find pipeline %s/%s", owner, repo))
// 		return
// 	}

// 	client := newSCMClient(req)
// 	body, _ := ioutil.ReadAll(req.Request.Body)
// 	payload := new(struct {
// 		Definition *ps.DefinitionFile `json:"definition"`
// 		Commit     map[string]string  `json:"commit"`
// 	})
// 	if err := json.Unmarshal(body, &payload); err != nil {
// 		jsonError(res, http.StatusInternalServerError, err, "Unable to read request payload")
// 		return
// 	}

// 	file, err := pipeline.UpdateDefinitionFile(client, payload.Definition, payload.Commit)
// 	if err != nil {
// 		jsonError(res, http.StatusInternalServerError, err, fmt.Sprintf("Unable to update definition file for %s/%s", pipeline.Owner, pipeline.Repo))
// 		return
// 	}
// 	res.WriteAsJson(file)
// }
