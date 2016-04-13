package pipeline

import (
	// "fmt"
	"testing"

	"github.com/AcalephStorage/kontinuous/store/kv"
)

func getUpdateStatusResources(status string) (*StatusUpdate, *Pipeline, *Build, *Stage, kv.KVClient, MockSCMClient) {
	kvc := setupStoreWithSampleStage()
	git := MockSCMClient{name: "github", success: true}

	p, _ := FindPipeline("SampleOwner", "SampleRepo", kvc)
	buildNum := 1
	b, _ := p.GetBuild(buildNum, kvc)
	b.Status = BuildPending
	stageIdx := 1
	s, _ := b.GetStage(stageIdx, kvc)
	s.Status = BuildPending

	u := &StatusUpdate{
		Status:    status,
		Timestamp: "1460183953",
	}

	return u, p, b, s, kvc, git
}

func TestUpdateRunningStatus(t *testing.T) {
	u, p, b, s, kvc, git := getUpdateStatusResources(BuildRunning)

	s.UpdateStatus(u, p, b, kvc, git)

	if b.Status != BuildRunning {
		t.Errorf("Expected build status to be %s", BuildRunning)
	}

	if b.Started == 0 {
		t.Error("Expected build started to be updated")
	}

	if b.Finished != 0 {
		t.Error("Expected build finished to not be updated")
	}

	if s.Status != BuildRunning {
		t.Errorf("Expected stage status to be %s", BuildRunning)
	}

	updatedStage, _ := b.GetStage(s.Index, kvc)

	if updatedStage.Status != BuildRunning {
		t.Errorf("Expected updated stage status to be %s", BuildRunning)
	}
}

func TestUpdateFailureStatus(t *testing.T) {
	u, p, b, s, kvc, git := getUpdateStatusResources(BuildFailure)

	s.UpdateStatus(u, p, b, kvc, git)

	if b.Status != BuildFailure {
		t.Errorf("Expected build status to be %s", BuildFailure)
	}

	if b.Finished == 0 {
		t.Error("Expected build finished to be updated")
	}

	if s.Status != BuildFailure {
		t.Errorf("Expected stage status to be %s", BuildFailure)
	}

	updatedStage, _ := b.GetStage(s.Index, kvc)

	if updatedStage.Status != BuildFailure {
		t.Errorf("Expected updated stage status to be %s", BuildFailure)
	}
}

// no next stage
func TestUpdateSuccessStatus(t *testing.T) {
	u, p, b, s, kvc, git := getUpdateStatusResources(BuildSuccess)

	s.UpdateStatus(u, p, b, kvc, git)

	if b.Status != BuildSuccess {
		t.Errorf("Expected build status to be %s", BuildSuccess)
	}

	if b.Finished == 0 {
		t.Error("Expected build finished to be updated", BuildSuccess)
	}

	if s.Status != BuildSuccess {
		t.Errorf("Expected stage status to be %s")
	}

	updatedStage, _ := b.GetStage(s.Index, kvc)

	if updatedStage.Status != BuildSuccess {
		t.Errorf("Expected updated stage status to be %s", BuildSuccess)
	}
}
