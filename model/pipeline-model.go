package model

type (
	PipelineStatus string
	RepositoryType string
)

const (
	GithubRepository RepositoryType = "github"

	PipelineCreated PipelineStatus = "created"
	PipelineReady   PipelineStatus = "ready"
)

type Pipeline struct {
	ID            string         `json:"id"`
	Name          string         `json:"name"`
	Spec          string         `json:"spec"`
	Created       int64          `json:"created"`
	Creator       string         `json:"creator"`
	Internal      bool           `json:"inline"`
	Repository    SpecRepository `json:"repository"`
	Status        PipelineStatus `json:"status"`
	StatusMessage string         `json:"statusMessage"`
}

type SpecRepository struct {
	Type     RepositoryType `json:"type"`
	Owner    string         `json:"owner"`
	Repo     string         `json:"repo"`
	URL      string         `json:"url"`
	SpecPath string         `json:"specPath"`
}
