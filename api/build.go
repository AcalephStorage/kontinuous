package api

import (
	"errors"
	"fmt"
	"strconv"
	"time"

	"encoding/json"
	"io/ioutil"
	"net/http"

	"github.com/emicklei/go-restful"

	ps "github.com/AcalephStorage/kontinuous/pipeline"
	"github.com/AcalephStorage/kontinuous/scm"
	"github.com/AcalephStorage/kontinuous/store/kv"
	"github.com/AcalephStorage/kontinuous/store/mc"
)

// BuildResource defines the endpoints for builds
type BuildResource struct {
	kv.KVClient
	*mc.MinioClient
}

// DashboardPayload contains the data expected from a build hook coming from the dashboard
type DashboardPayload struct {
	Author string `json:"author"`
}

func (b *BuildResource) extend(ws *restful.WebService) {

	ws.Route(ws.GET("/{owner}/{repo}/builds").To(b.list).
		Doc("Get pipelines for repo").
		Operation("list").
		Param(ws.PathParameter("owner", "repository owner name").DataType("string")).
		Param(ws.PathParameter("repo", "repository name").DataType("string")).
		Writes([]ps.Build{}).
		Filter(requireAccessToken))

	ws.Route(ws.POST("/{owner}/{repo}/builds").To(b.create).
		Doc("Create build details").
		Operation("create").
		Param(ws.PathParameter("owner", "repository owner name").DataType("string")).
		Param(ws.PathParameter("repo", "repository name").DataType("string")).
		Param(ws.HeaderParameter("X-Custom-Event", "specifies a custom event, supports: dashboard, cli").DataType("string")).
		Reads(DashboardPayload{}).
		Writes(ps.Build{}).
		Filter(requireAccessToken))

	ws.Route(ws.GET("/{owner}/{repo}/builds/{buildNumber}").To(b.show).
		Doc("Show build details").
		Operation("show").
		Param(ws.PathParameter("owner", "repository owner name").DataType("string")).
		Param(ws.PathParameter("repo", "repository name").DataType("string")).
		Param(ws.PathParameter("buildNumber", "build number").DataType("int")).
		Writes(ps.Build{}).
		Filter(requireAccessToken))
}

func (b *BuildResource) create(req *restful.Request, res *restful.Response) {
	owner := req.PathParameter("owner")
	repo := req.PathParameter("repo")
	pipeline, err := findPipeline(owner, repo, b.KVClient)
	if err != nil {
		jsonError(res, http.StatusNotFound, err, fmt.Sprintf("Unable to find pipeline %s/%s", owner, repo))
		return
	}

	// let ping pass
	if b.isPing(req.Request) {
		return
	}

	// parse hook details
	client := newSCMClient(req)
	body, _ := ioutil.ReadAll(req.Request.Body)
	hook := new(scm.Hook)

	switch {
	case b.isRemoteEvent(&req.Request.Header):
		client, err := getScopedClient(pipeline.Login, b.KVClient, req)
		if err != nil {
			jsonError(res, http.StatusBadRequest, err, "Unable to retrieve remote user")
			return
		}

		hook, err = client.ParseHook(body, req.HeaderParameter("X-Github-Event"))
	case b.isCustomEvent(&req.Request.Header):
		client.SetAccessToken(req.HeaderParameter("Authorization"))
		hook, err = b.parseCustomHook(owner, repo, body, req.HeaderParameter("X-Custom-Event"), client)
	default:
		jsonError(res, http.StatusUnauthorized, errors.New("Unknown event trigger"), "Hook source unknown")
		return
	}

	if err != nil {
		jsonError(res, http.StatusNotFound, err, "Unable to parse hook")
		return
	}

	// persist build
	build := &ps.Build{
		Author:   hook.Author,
		Branch:   hook.Branch,
		CloneURL: hook.CloneURL,
		Commit:   hook.Commit,
		Event:    hook.Event,
	}

	if err = pipeline.CreateBuild(build, []*ps.Stage{}, b.KVClient, client); err != nil {
		jsonError(res, http.StatusInternalServerError, err, "Unable to create build")
		return
	}

	info := &ps.NextJobInfo{hook.Commit, build.Number, 1}
	definition, jobInfo, err := pipeline.PrepareBuildStage(info, client)
	if err != nil {
		msg := fmt.Sprintf("Unable to get stage details %s/%s/builds/%s/stages/%d", owner, repo, build.Number, 1)
		jsonError(res, http.StatusInternalServerError, err, msg)
		return
	}

	//update details in pipeline
	pipeline.UpdatePipeline(definition, b.KVClient)

	// save stage details
	build.Stages = definition.GetStages()
	if err := build.CreateStages(b.KVClient); err != nil {
		msg := fmt.Sprintf("Unable to save stage details %s/%s/builds/%s", owner, repo, build.Number)
		jsonError(res, http.StatusInternalServerError, err, msg)
		return
	}

	stageStatus := &ps.StatusUpdate{
		Status:    ps.BuildFailure,
		Timestamp: strconv.FormatInt(time.Now().UnixNano(), 10),
	}
	stage, err := findStage("1", build, b.KVClient)
	if err != nil {
		jsonError(res, http.StatusInternalServerError, err, "Stage not found")
		return
	}

	switch stage.Type {
	case "deploy":
		stageStatus.Status = ps.BuildRunning
		stage.UpdateStatus(stageStatus, pipeline, build, b.KVClient, client)

		err := stage.Deploy(pipeline, build, client)
		if err != nil {
			stageStatus.Status = ps.BuildFailure
			stage.UpdateStatus(stageStatus, pipeline, build, b.KVClient, client)
			msg := fmt.Sprintf("Unable to deploy resouce to kubernetes for %s/%s/builds/%d/stages/%d", pipeline.Owner, pipeline.Repo, build.Number, stage.Index)
			jsonError(res, http.StatusInternalServerError, err, msg)
			return
		}
		stageStatus.Status = ps.BuildSuccess
		stage.UpdateStatus(stageStatus, pipeline, build, b.KVClient, client)

	default:
		if _, err := ps.CreateJob(definition, jobInfo); err != nil {
			stage.UpdateStatus(stageStatus, pipeline, build, b.KVClient, client)
			msg := fmt.Sprintf("Unable to create job for %s/%s/builds/%s/stages/%d", owner, repo, build.Number, 1)
			jsonError(res, http.StatusInternalServerError, err, msg)
			return
		}

	}

	res.WriteEntity(build)
}

func (b *BuildResource) list(req *restful.Request, res *restful.Response) {
	owner := req.PathParameter("owner")
	repo := req.PathParameter("repo")
	pipeline, err := findPipeline(owner, repo, b.KVClient)
	if err != nil {
		jsonError(res, http.StatusNotFound, err, fmt.Sprintf("Unable to find pipeline %s/%s", owner, repo))
		return
	}

	builds, err := pipeline.GetBuilds(b.KVClient)
	if err != nil {
		jsonError(res, http.StatusInternalServerError, err, fmt.Sprintf("Unable to list builds for %s/%s", owner, repo))
		return
	}

	res.WriteEntity(builds)
}

func (b *BuildResource) show(req *restful.Request, res *restful.Response) {
	owner := req.PathParameter("owner")
	repo := req.PathParameter("repo")
	buildNumber := req.PathParameter("buildNumber")
	pipeline, err := findPipeline(owner, repo, b.KVClient)
	if err != nil {
		jsonError(res, http.StatusNotFound, err, fmt.Sprintf("Unable to find pipeline %s/%s", owner, repo))
		return
	}

	build, err := findBuild(buildNumber, pipeline, b.KVClient)
	if err != nil {
		jsonError(res, http.StatusNotFound, err, fmt.Sprintf("Unable to find build %s for %s/%s", buildNumber, owner, repo))
		return
	}

	res.WriteEntity(build)
}

func (b *BuildResource) isPing(req *http.Request) bool {
	// add other ping checks here
	return req.Header.Get("X-Github-Event") == scm.EventPing
}

func (b *BuildResource) isRemoteEvent(h *http.Header) bool {
	switch {
	case h.Get("X-Github-Event") != "":
		return true
	default:
		return false
	}
}

func (b *BuildResource) isCustomEvent(h *http.Header) bool {
	return h.Get("X-Custom-Event") == scm.EventDashboard || h.Get("X-Custom-Event") == scm.EventCLI
}

func (b *BuildResource) parseCustomHook(owner, repo string, body []byte, event string, scmClient scm.Client) (*scm.Hook, error) {
	var payload map[string]string
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, err
	}

	source, exists := scmClient.GetRepository(owner, repo)
	if !exists {
		return nil, fmt.Errorf("Repository has no remote source from %s.", scmClient.Name())
	}

	hook := &scm.Hook{
		Author:   payload["author"],
		Branch:   source.DefaultBranch,
		CloneURL: source.CloneURL,
		Commit:   source.DefaultBranch,
		Event:    event,
	}

	return hook, nil
}
