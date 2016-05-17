package github

import (
	"fmt"
	"strings"

	"encoding/json"

	"github.com/google/go-github/github"
	"golang.org/x/oauth2"

	"github.com/AcalephStorage/kontinuous/scm"
)

// Client is used for making requests to GitHub
type Client struct {
	token string
}

func (gc *Client) client() *github.Client {
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: gc.token},
	)
	tc := oauth2.NewClient(oauth2.NoContext, ts)
	return github.NewClient(tc)
}

// CreateHook creates a webhook
func (gc *Client) CreateHook(owner, repo, callback string, events []string) error {
	hook := &github.Hook{
		Events: events,
		Name:   github.String("web"),
		Config: map[string]interface{}{
			"url":          callback,
			"content_type": "json",
		},
	}

	if _, _, err := gc.client().Repositories.CreateHook(owner, repo, hook); err != nil {
		return err
	}

	return nil
}

// CreateKey creates repository deploy keys
func (gc *Client) CreateKey(owner, repo, key, title string) error {
	deployKey := &github.Key{
		Key:   github.String(key),
		Title: github.String(title),
	}

	if _, _, err := gc.client().Repositories.CreateKey(owner, repo, deployKey); err != nil {
		return err
	}

	return nil
}

func (gc *Client) CreateStatus(owner, repo, ref string, stageID int, stageName, state string) error {

	context := fmt.Sprintf("kontinuous:%d", stageID)

	status := &github.RepoStatus{
		State:       &state,
		Description: &stageName,
		Context:     &context,
	}

	if _, _, err := gc.client().Repositories.CreateStatus(owner, repo, ref, status); err != nil {
		return err
	}

	return nil
}

// GetFileContent fetches a file from the given commit or branch
func (gc *Client) GetFileContent(owner, repo, path, ref string) ([]byte, bool) {
	file, _, _, err := gc.client().Repositories.GetContents(owner,
		repo,
		path,
		&github.RepositoryContentGetOptions{ref})
	if err != nil {
		return nil, false
	}

	decoded, err := file.Decode()
	if err != nil {
		return nil, false
	}

	return decoded, true
}

// GetContents gets the metadata and content of a file from the given commit or branch
func (gc *Client) GetContents(owner, repo, path, ref string) (*scm.RepositoryContent, bool) {
	file, _, _, err := gc.client().Repositories.GetContents(owner,
		repo,
		path,
		&github.RepositoryContentGetOptions{ref})
	if err != nil {
		return nil, false
	}

	return &scm.RepositoryContent{
		Content: file.Content,
		SHA:     file.SHA,
	}, true
}

// UpdateFile commits diff of a file content from a given commit ref
func (gc *Client) UpdateFile(owner, repo, path, commit string, content []byte) error {
	message := fmt.Sprintf("Update %s", path)
	_, _, err := gc.client().Repositories.UpdateFile(owner,
		repo,
		path,
		&github.RepositoryContentFileOptions{
			Message: &message,
			Content: content,
			SHA:     &commit,
		})
	if err != nil {
		return err
	}
	return nil
}

// GetRepository fetches repository details from GitHub
func (gc *Client) GetRepository(owner, name string) (*scm.Repository, bool) {
	data, _, err := gc.client().Repositories.Get(owner, name)
	if err != nil {
		return nil, false
	}

	repo := &scm.Repository{
		ID:            *data.ID,
		Owner:         *data.Owner.Login,
		Name:          *data.Name,
		FullName:      *data.FullName,
		Avatar:        *data.Owner.AvatarURL,
		CloneURL:      *data.CloneURL,
		Permissions:   *data.Permissions,
		DefaultBranch: *data.DefaultBranch,
	}

	return repo, true
}

// ListRepositories lists the repositories accessible by the current user
func (gc *Client) ListRepositories(user string) (repos []*scm.Repository, err error) {
	// thanks drone
	opts := new(github.RepositoryListOptions)
	opts.PerPage = 100
	opts.Page = 1

	// loop through user repository list
	for opts.Page > 0 {
		list, res, err := gc.client().Repositories.List(user, opts)
		if err != nil {
			return nil, err
		}

		for _, repo := range list {
			repos = append(repos, &scm.Repository{
				ID:            *repo.ID,
				Owner:         *repo.Owner.Login,
				Name:          *repo.Name,
				FullName:      *repo.FullName,
				Avatar:        *repo.Owner.AvatarURL,
				DefaultBranch: *repo.DefaultBranch,
			})
		}
		// increment the next page to retrieve
		opts.Page = res.NextPage
	}

	return repos, nil
}

// ParseHook parses the contents of a webhook to build useful data
func (gc *Client) ParseHook(body []byte, event string) (*scm.Hook, error) {
	payload := new(PushHook)
	if err := json.Unmarshal(body, payload); err != nil {
		return nil, err
	}

	hook := &scm.Hook{
		Author:   payload.Sender.Login,
		Branch:   strings.Replace(payload.Ref, "refs/heads/", "", -1),
		CloneURL: payload.Repo.CloneURL,
		Commit:   payload.Head.ID,
		Event:    event,
	}

	return hook, nil
}

// HookExists checks whether a webhook with the given callback already exists
func (gc *Client) HookExists(owner, repo, url string) bool {
	hooks, _, err := gc.client().Repositories.ListHooks(owner, repo, nil)
	if err != nil {
		return false
	}

	for _, hook := range hooks {
		if hook.Config["url"].(string) == url {
			return true
		}
	}

	return false
}

// AccessToken returns the client's access token
func (gc *Client) AccessToken() string {
	return gc.token
}

// SetAccessToken sets the client's access token
func (gc *Client) SetAccessToken(token string) {
	gc.token = token
}

// Name returns the client's remote source name
func (gc *Client) Name() string {
	return scm.RepoGithub
}
