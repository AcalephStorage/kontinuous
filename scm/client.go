package scm

const (
	// EventDashboard indicates a dashboard event
	EventDashboard = "dashboard"

	// EventDashboard indicates a CLI event
	EventCLI = "cli"

	// EventPing indicates a ping event
	EventPing = "ping"

	// EventPush indicates a push event
	EventPush = "push"

	// EventPullRequest indicates a pull request
	EventPullRequest = "pull_request"

	// EventDeployment indicates a deployment
	EventDeployment = "deployment"

	// RepoGithub represents GitHub
	RepoGithub = "github"

	// StatePending represents a pending build/stage state
	StatePending = "pending"

	// StateSuccess represents a successful build/stage state
	StateSuccess = "success"

	// StateError represents an error build/stage state
	StateError = "error"

	// StateFailure represents a failed build/stage state
	StateFailure = "failure"
)

// Client is an interface for accessing remote SCMs
type Client interface {
	AccessToken() string
	SetAccessToken(string)
	Name() string
	HookExists(owner, repo, url string) bool
	CreateHook(owner, repo, callback string, events []string) error
	CreateKey(owner, repo, key, title string) error
	CreateStatus(owner, repo, sha string, stageId int, stageName, state string) error
	GetContents(owner, repo, content, ref string) ([]byte, bool)
	GetRepository(owner, repo string) (*Repository, bool)
	ListRepositories(user string) ([]*Repository, error)
	ParseHook(payload []byte, event string) (*Hook, error)
}

// Repository holds common repository details from SCMs
type Repository struct {
	ID            int             `json:"id"`
	Owner         string          `json:"owner"`
	Name          string          `json:"name"`
	FullName      string          `json:"full_name"`
	Avatar        string          `json:"avatar_url"`
	CloneURL      string          `json:"clone_url,omitempty"`
	DefaultBranch string          `json:"default_branch"`
	Permissions   map[string]bool `json:"-"`
}

// IsAdmin determines if the scoped user has admin rights for the repository
func (r *Repository) IsAdmin() bool {
	return r.Permissions["admin"]
}

// Hook contains the common details to be extracted from webhooks
type Hook struct {
	Author   string
	Branch   string
	CloneURL string
	Commit   string
	Event    string
}
