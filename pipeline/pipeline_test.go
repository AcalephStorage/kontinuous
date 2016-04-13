package pipeline

import (
	"fmt"
	"testing"
)

func TestCreateValidPipeline(t *testing.T) {
	kvc := setupStore()

	git := MockSCMClient{name: "github", success: true}

	p := &Pipeline{
		Owner:  "SampleOwner",
		Repo:   "SampleRepo",
		Login:  "github-user",
		Source: "github",
		Events: []string{"push"},
	}
	err := CreatePipeline(p, git, kvc)
	if err != nil {
		t.Errorf("Expected pipeline creation to succeed, got error: ", err)
	}
}

func TestCreateExistingPipeline(t *testing.T) {
	kvc := setupStoreWithSampleRepo()

	git := MockSCMClient{name: "github", success: true}

	p := &Pipeline{
		Owner:  "SampleOwner",
		Repo:   "SampleRepo",
		Login:  "github-user",
		Source: "github",
		Events: []string{"push"},
	}
	actual := CreatePipeline(p, git, kvc)

	expected := fmt.Errorf("Pipeline %s/%s already exists!", p.Owner, p.Repo).Error()
	if actual == nil {
		t.Errorf("Expected to get error `%s`", expected)
	} else if actual.Error() != expected {
		t.Errorf("Expected pipeline creation to fail with error `%s`, but got `%s` ", expected, actual.Error())
	}
}

func TestFindExistingPipeline(t *testing.T) {
	kvc := setupStoreWithSampleRepo()

	_, exists := FindPipeline("SampleOwner", "SampleRepo", kvc)
	if !exists {
		t.Error("Expected to find pipeline.")
	}
}

func TestFindNonExistingPipeline(t *testing.T) {
	kvc := setupStore()

	_, exists := FindPipeline("SampleOwner", "SampleRepo", kvc)
	if exists {
		t.Error("Expected to not find pipeline.")
	}
}

func TestFindAllPipelines(t *testing.T) {
	kvc := setupStoreWithSampleRepo()

	ps, _ := FindAllPipelines(kvc)
	if len(ps) != 1 {
		t.Errorf("Expected to get `1` pipeline, got `%d`", len(ps))
	}
}

func TestFindAllEmptyPipelines(t *testing.T) {
	kvc := setupStore()

	ps, _ := FindAllPipelines(kvc)
	if len(ps) != 0 {
		t.Errorf("Expected to get `0` pipelines, got `%d`", len(ps))
	}
}

func TestCreateBuild(t *testing.T) {
	kvc := setupStoreWithSampleRepo()

	p, _ := FindPipeline("SampleOwner", "SampleRepo", kvc)
	build := &Build{}
	def := &Definition{}
	stages := def.GetStages()

	actual := p.CreateBuild(build, stages, kvc, nil)
	if actual != nil {
		t.Errorf("Expected to create build without an error, got `%s`", actual.Error())
	}
}

func TestGetBuilds(t *testing.T) {
	kvc := setupStoreWithSampleBuild()

	p, _ := FindPipeline("SampleOwner", "SampleRepo", kvc)
	builds, _ := p.GetBuilds(kvc)

	if len(builds) != 1 {
		t.Errorf("Expected to return 1 pipeline build, got %d", len(builds))
	}
}

func TestGetBuild(t *testing.T) {
	kvc := setupStoreWithSampleBuild()

	p, _ := FindPipeline("SampleOwner", "SampleRepo", kvc)
	buildNum := 1
	build, _ := p.GetBuild(buildNum, kvc)

	if build.Number != buildNum {
		t.Errorf("Expected to get build Number `%d`, got `%d`", buildNum, build.Number)
	}
}

func TestPrepareBuildStage(t *testing.T) {
	kvc := setupStoreWithSampleBuild()
	git := MockSCMClient{name: "github", success: true}
	login := "SampleUser"
	p, _ := FindPipeline("SampleOwner", "SampleRepo", kvc)
	p.Login = login

	buildNum := 1
	build, _ := p.GetBuild(buildNum, kvc)
	info := &NextJobInfo{build.Commit, build.Number, 1}

	definition, jobInfo, _ := p.PrepareBuildStage(info, git)

	if definition == nil {
		t.Error("Expected definition to be defined!")
	}

	if jobInfo == nil {
		t.Error("Expected job info to be defined!")
	}
}
