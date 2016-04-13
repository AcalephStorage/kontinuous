package pipeline

import (
	"testing"
)

func TestGetStages(t *testing.T) {
	kvc := setupStoreWithSampleStage()

	p, _ := FindPipeline("SampleOwner", "SampleRepo", kvc)
	buildNum := 1
	b, _ := p.GetBuild(buildNum, kvc)
	stages, _ := b.GetStages(kvc)

	if len(stages) != 1 {
		t.Errorf("Expected to return 1 pipeline build stage, got %d", len(stages))
	}
}

func TestGetExistingStage(t *testing.T) {
	kvc := setupStoreWithSampleStage()

	p, _ := FindPipeline("SampleOwner", "SampleRepo", kvc)
	buildNum := 1
	b, _ := p.GetBuild(buildNum, kvc)
	stageIdx := 1
	stage, _ := b.GetStage(stageIdx, kvc)

	if stage.Index != stageIdx {
		t.Errorf("Expected to get stageIdx `%d`, got `%d`", stageIdx, stage.Index)
	}
}

func TestGetNonExistentStage(t *testing.T) {
	kvc := setupStoreWithSampleStage()

	p, _ := FindPipeline("SampleOwner", "SampleRepo", kvc)
	buildNum := 1
	b, _ := p.GetBuild(buildNum, kvc)
	stageIdx := 2
	stage, _ := b.GetStage(stageIdx, kvc)

	if stage != nil {
		t.Errorf("Not expecting a stage but got one")
	}
}
