package notif

type StageStatus struct {
	Name   string
	Status string
}

type AppNotifier interface {
	PostMessage(pipelineName string, buildNumber int, buildStatus string, statuses []StageStatus, metadata map[string]interface{}) bool
}
