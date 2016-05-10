package api

import (
	"fmt"
	"time"

	"net/http"

	"github.com/emicklei/go-restful"

	"github.com/AcalephStorage/kontinuous/kube"
	ps "github.com/AcalephStorage/kontinuous/pipeline"
	buildlog "github.com/AcalephStorage/kontinuous/pipeline/log"
	"github.com/AcalephStorage/kontinuous/scm"
	"github.com/AcalephStorage/kontinuous/store/kv"
	"github.com/AcalephStorage/kontinuous/store/mc"
)

// StageResource defines the endpoints for build stages
type StageResource struct {
	kv.KVClient
	*mc.MinioClient
	kube.KubeClient
}

func (s *StageResource) extend(ws *restful.WebService) {

	ws.Route(ws.GET("/{owner}/{repo}/builds/{buildNumber}/stages").To(s.list).
		Doc("Get build stage details").
		Operation("list").
		Param(ws.PathParameter("owner", "repository owner name").DataType("string")).
		Param(ws.PathParameter("repo", "repository name").DataType("string")).
		Param(ws.PathParameter("buildNumber", "build number").DataType("int")).
		Writes([]ps.Stage{}))

	ws.Route(ws.GET("/{owner}/{repo}/builds/{buildNumber}/stages/{stageIndex}").To(s.show).
		Doc("Get build stage details").
		Operation("show").
		Param(ws.PathParameter("owner", "repository owner name").DataType("string")).
		Param(ws.PathParameter("repo", "repository name").DataType("string")).
		Param(ws.PathParameter("buildNumber", "build number").DataType("int")).
		Param(ws.PathParameter("stageIndex", "stage index").DataType("int")).
		Writes(ps.Stage{}))

	ws.Route(ws.POST("/{owner}/{repo}/builds/{buildNumber}/stages/{stageIndex}").To(s.update).
		Doc("Send metadata to build stage").
		Operation("update").
		Param(ws.PathParameter("owner", "repository owner name").DataType("string")).
		Param(ws.PathParameter("repo", "repository name").DataType("string")).
		Param(ws.PathParameter("buildNumber", "build number").DataType("int")).
		Param(ws.PathParameter("stageIndex", "stage index").DataType("int")).
		Reads(ps.StatusUpdate{}).
		Writes(ps.Stage{}))

	ws.Route(ws.POST("/{owner}/{repo}/builds/{buildNumber}/stages/{stageIndex}/run").To(s.run).
		Doc("Run build at given stage").
		Operation("run").
		Param(ws.PathParameter("owner", "repository owner name").DataType("string")).
		Param(ws.PathParameter("repo", "repository name").DataType("string")).
		Param(ws.PathParameter("buildNumber", "build number").DataType("int")).
		Param(ws.PathParameter("stageIndex", "stage index").DataType("int")).
		Writes(ps.Stage{}))

	ws.Route(ws.GET("/{owner}/{repo}/builds/{buildNumber}/stages/{stageIndex}/logs").To(s.logs).
		Doc("Show build stage logs").
		Operation("logs").
		Param(ws.PathParameter("owner", "repository owner name").DataType("string")).
		Param(ws.PathParameter("repo", "repository name").DataType("string")).
		Param(ws.PathParameter("buildNumber", "build number").DataType("int")).
		Param(ws.PathParameter("stageIndex", "stage index").DataType("int")).
		Writes([]buildlog.Log{}))
}

func (s *StageResource) run(req *restful.Request, res *restful.Response) {
	owner := req.PathParameter("owner")
	repo := req.PathParameter("repo")
	buildNumber := req.PathParameter("buildNumber")
	stageIndex := req.PathParameter("stageIndex")

	pipeline, build, stage, err := s.fetchResources(owner, repo, buildNumber, stageIndex)
	if err != nil {
		jsonError(res, http.StatusNotFound, err, "Unable to find resource")
		return
	}

	client, err := getScopedClient(pipeline.Login, s.KVClient, req)
	if err != nil {
		jsonError(res, http.StatusBadRequest, err, "Unable to retrieve remote user")
		return
	}

	if err, msg := s.runStage(pipeline, build, stage, client); err != nil {
		jsonError(res, http.StatusInternalServerError, err, msg)
	}

	res.WriteEntity(stage)
}

func (s *StageResource) logs(req *restful.Request, res *restful.Response) {
	owner := req.PathParameter("owner")
	repo := req.PathParameter("repo")
	buildNumber := req.PathParameter("buildNumber")
	stageIndex := req.PathParameter("stageIndex")

	pipeline, build, stage, err := s.fetchResources(owner, repo, buildNumber, stageIndex)
	if err != nil {
		jsonError(res, http.StatusNotFound, err, "Unable to find resource")
		return
	}

	var logs []buildlog.Log
	if stage.Status == ps.BuildRunning {
		// where to get ref?
		ref := build.Commit
		client, err := getScopedClient(pipeline.Login, s.KVClient, req)
		if err != nil {
			jsonError(res, http.StatusInternalServerError, err, "unable to create scm client")
			return
		}
		definition, err := pipeline.Definition(ref, client)
		if err != nil {
			jsonError(res, http.StatusInternalServerError, err, "unable to get pipeline definition")
		}
		namespace := definition.Metadata["namespace"].(string)
		if namespace == "" {
			namespace = "default"
		}
		logs, err = buildlog.FetchRunningLogs(s.KubeClient, namespace, pipeline.ID, buildNumber, stageIndex)
	} else {
		logs, err = buildlog.FetchLogs(s.MinioClient, pipeline.ID, buildNumber, stageIndex)
	}

	if err != nil {
		msg := fmt.Sprintf("Unable to find logs for %s/%s/builds/%s/stages/%s", owner, repo, buildNumber, stageIndex)
		jsonError(res, http.StatusNotFound, err, msg)
		return
	}

	res.WriteEntity(logs)
}

func (s *StageResource) list(req *restful.Request, res *restful.Response) {
	owner := req.PathParameter("owner")
	repo := req.PathParameter("repo")
	buildNumber := req.PathParameter("buildNumber")
	pipeline, err := findPipeline(owner, repo, s.KVClient)
	if err != nil {
		jsonError(res, http.StatusNotFound, err, fmt.Sprintf("Unable to find pipeline %s/%s", owner, repo))
		return
	}

	build, err := findBuild(buildNumber, pipeline, s.KVClient)
	if err != nil {
		jsonError(res, http.StatusNotFound, err, fmt.Sprintf("Unable to find build %s for %s/%s", buildNumber, owner, repo))
		return
	}

	stages, err := build.GetStages(s.KVClient)
	if err != nil {
		jsonError(res, http.StatusNotFound, err, fmt.Sprintf("Unable to find stages for %s/%s/builds/%s", owner, repo, buildNumber))
		return
	}

	res.WriteEntity(stages)
}

func (s *StageResource) show(req *restful.Request, res *restful.Response) {
	owner := req.PathParameter("owner")
	repo := req.PathParameter("repo")
	buildNumber := req.PathParameter("buildNumber")
	stageIndex := req.PathParameter("stageIndex")

	_, _, stage, err := s.fetchResources(owner, repo, buildNumber, stageIndex)
	if err != nil {
		jsonError(res, http.StatusNotFound, err, "Unable to find resource")
		return
	}

	res.WriteEntity(stage)
}

func (s *StageResource) update(req *restful.Request, res *restful.Response) {
	owner := req.PathParameter("owner")
	repo := req.PathParameter("repo")
	buildNumber := req.PathParameter("buildNumber")
	stageIndex := req.PathParameter("stageIndex")

	pipeline, build, stage, err := s.fetchResources(owner, repo, buildNumber, stageIndex)
	if err != nil {
		jsonError(res, http.StatusNotFound, err, "Unable to find resource")
		return
	}

	status := new(ps.StatusUpdate)
	err = req.ReadEntity(status)
	if err != nil {
		res.WriteHeader(http.StatusBadRequest)
		return
	}

	client, err := getScopedClient(pipeline.Login, s.KVClient, req)
	if err != nil {
		jsonError(res, http.StatusBadRequest, err, "Unable to retrieve remote user")
		return
	}

	nextStage, err := stage.UpdateStatus(status, pipeline, build, s.KVClient, client)
	if err != nil {
		jsonError(res, http.StatusBadRequest, err, "Unable to update stage status")
		return
	}

	if nextStage != nil {
		if err, msg := s.runStage(pipeline, build, nextStage, client); err != nil {
			jsonError(res, http.StatusInternalServerError, err, msg)
		}
	}

	res.WriteHeader(http.StatusOK)
}

func (s *StageResource) fetchResources(owner, repo, buildNumber, stageIndex string) (*ps.Pipeline, *ps.Build, *ps.Stage, error) {

	pipeline, err := findPipeline(owner, repo, s.KVClient)
	if err != nil {
		return nil, nil, nil, err
	}

	build, err := findBuild(buildNumber, pipeline, s.KVClient)
	if err != nil {
		return pipeline, nil, nil, err
	}

	stage, err := findStage(stageIndex, build, s.KVClient)
	if err != nil {
		return pipeline, build, nil, err
	}

	return pipeline, build, stage, nil
}

func (s *StageResource) runStage(pipeline *ps.Pipeline, build *ps.Build, stage *ps.Stage, scmClient scm.Client) (error, string) {
	info := &ps.NextJobInfo{build.Commit, build.Number, stage.Index}
	definition, jobInfo, err := pipeline.PrepareBuildStage(info, scmClient)
	if err != nil {
		msg := fmt.Sprintf("Unable to get stage details %s/%s/builds/%d/stages/%d", pipeline.Owner, pipeline.Repo, build.Number, stage.Index)
		return err, msg
	}

	stageStatus := &ps.StatusUpdate{
		Status:    ps.BuildFailure,
		Timestamp: time.Now().UnixNano(),
	}

	switch stage.Type {
	case "deploy":
		stageStatus.Status = ps.BuildRunning
		stage.UpdateStatus(stageStatus, pipeline, build, s.KVClient, scmClient)
		err := stage.Deploy(pipeline, build, scmClient)
		if err != nil {
			stageStatus.Status = ps.BuildFailure
			stage.UpdateStatus(stageStatus, pipeline, build, s.KVClient, scmClient)
			msg := fmt.Sprintf("Unable to deploy resouce to kubernetes for %s/%s/builds/%s/stages/%d", pipeline.Owner, pipeline.Repo, build.Number, stage.Index)
			return err, msg
		}
		stageStatus.Status = ps.BuildSuccess
		stage.UpdateStatus(stageStatus, pipeline, build, s.KVClient, scmClient)

	default:
		if _, err := ps.CreateJob(definition, jobInfo); err != nil {
			stage.UpdateStatus(stageStatus, pipeline, build, s.KVClient, scmClient)
			msg := fmt.Sprintf("Unable to create job for %s/%s/builds/%s/stages/%d", pipeline.Owner, pipeline.Repo, build.Number, stage.Index)
			return err, msg
		}
	}

	return nil, ""
}
