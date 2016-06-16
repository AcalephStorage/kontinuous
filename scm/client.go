package scm

const (
	// EventDashboard indicates a dashboard event
	EventDashboard = "dashboard"

	// EventCLI indicates a CLI event
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
	CreateStatus(owner, repo, sha string, stageID int, stageName, state string) error
	GetFileContent(owner, repo, path, ref string) ([]byte, bool)
	GetDirectoryContent(owner, repo, path, ref string) ([]interface{}, bool)
	GetContents(owner, repo, path, ref string) (*RepositoryContent, bool)
	CreateFile(owner, repo, path, message, branch string, content []byte) (*RepositoryContent, error)
	UpdateFile(owner, repo, path, blob, message, branch string, content []byte) (*RepositoryContent, error)
	GetRepository(owner, repo string) (*Repository, bool)
	ListRepositories(user string) ([]*Repository, error)
	ParseHook(payload []byte, event string) (*Hook, error)
	GetHead(owner, repo, branch string) (string, error)
	CreateBranch(owner, repo, branchName, baseRef string) (string, error)
	CreatePullRequest(owner, repo, baseRef, headRef, title string) error
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

// RepositoryContent contains metadata of a file/directory in a repository
type RepositoryContent struct {
	Content *string `json:"content,omitempty"`
	SHA     *string `json:"sha"`
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
