package pipeline

import (
	"errors"
	"strconv"
	"strings"

	"github.com/AcalephStorage/kontinuous/scm"
	"github.com/AcalephStorage/kontinuous/store/kv"
)

type (
	MockSCMClient struct {
		success bool
		name    string
		token   string
	}

	MockKVClient struct {
		mockError error
		data      map[string]string
	}
)

var validyamlSpec = `
apiVersion: v1alpha1
kind: Pipeline
metadata:
  name: my-pipeline
  namespace: acaleph
spec:
  selector:
    matchLabels:
      app: my-pipeline-infra # can be used as a selector for finding infra launched during a build?
  template: # taken from Job spec... needed?
    metadata:
      name: my-pipeline
      labels:
        app: my-pipeline
      image: acaleph/deploy-base # Overridable?
    stages:
    - name: Test Infra
      type: command
      params:
        command: ["test_infra.sh", "--hack-the-gibson"]
        env: # env vars
          - name: MYVAR
            value: my value
        artifact_paths: # sent to minio?
          - "logs/**/*"
          - "coverage/**/*"
        timeout: 60 # kills after X minutes?
    - name: "Have you finished testing?" # friendly name
      type: block # waits for the user to approve
    - name: Teardown Deploy local test infra
      type: deploy_cleanup # stop
      selector:
        build: my-val-stage-1 # selects build to stop
    - name: "Do you want to Publish to Production?" # friendly name
      type: block # waits for the user to approve
    - name: Deploy production
      type: deploy
      params:
        template: production.yaml # DM template
        properties: # DM properties
          external_service: true
          replicas: 3
      labels: # labels to cleanup build
        state: canary
 `

func setupStore() kv.KVClient {
	return &MockKVClient{data: map[string]string{}}
}

func setupStoreWithSampleRepo() kv.KVClient {
	key := pipelineNamespace + "SampleOwner:SampleRepo"

	kv := &MockKVClient{data: map[string]string{}}
	kv.PutDir(key)

	return kv
}

func setupStoreWithSampleBuild() kv.KVClient {
	pipeline := "SampleOwner:SampleRepo"
	key := pipelineNamespace + pipeline
	ownerKey := key + "/owner"
	repoKey := key + "/repo"
	buildsKey := pipelineNamespace + pipeline + "/builds"
	buildKey := buildsKey + "/1"
	buildNumKey := buildKey + "/number"

	kv := &MockKVClient{data: map[string]string{}}
	kv.PutDir(key)
	kv.Put(ownerKey, "SampleOwner")
	kv.Put(repoKey, "SampleRepo")
	kv.PutInt(buildNumKey, 1)

	return kv
}

func setupStoreWithSampleStage() kv.KVClient {
	pipeline := "SampleOwner:SampleRepo"
	key := pipelineNamespace + pipeline
	buildsKey := pipelineNamespace + pipeline + "/builds"
	buildKey := buildsKey + "/1"
	stagesKey := buildKey + "/stages"
	stageKey := stagesKey + "/1"

	kv := &MockKVClient{data: map[string]string{}}
	kv.PutDir(key)
	kv.Put(key+"/owner", "SampleOwner")
	kv.Put(key+"/repo", "SampleRepo")
	kv.PutInt(buildKey+"/number", 1)
	kv.Put(buildKey+"/pipeline", pipeline)
	kv.PutInt(stageKey+"/index", 1)

	return kv
}

func setupStoreWithSampleUser(id string) kv.KVClient {
	key := userNamespace + id
	remoteIDKey := key + "/remote-id"

	kv := &MockKVClient{data: map[string]string{}}
	kv.PutDir(key)
	kv.Put(remoteIDKey, id)

	return kv
}

func (s MockSCMClient) AccessToken() string {
	return s.token
}

func (s MockSCMClient) GetRepository(owner, name string) (*scm.Repository, bool) {
	if !s.success {
		return nil, false
	}

	repo := &scm.Repository{
		ID:          1,
		Owner:       owner,
		Name:        name,
		Permissions: map[string]bool{"admin": true},
	}
	return repo, true
}

// kv mocks
func (kv *MockKVClient) Put(key, value string) error {
	kv.data[key] = value
	return kv.mockError
}

func (kv *MockKVClient) Get(key string) (string, error) {
	return kv.data[key], kv.mockError
}

func (kv *MockKVClient) PutInt(key string, value int) error {
	return kv.Put(key, strconv.Itoa(value))
}

func (kv *MockKVClient) GetInt(key string) (int, error) {
	val, _ := kv.Get(key)
	return strconv.Atoi(val)
}

func (kvc *MockKVClient) GetDir(key string) ([]*kv.KVPair, error) {
	kvpair := []*kv.KVPair{}

	// review checking
	for k, v := range kvc.data {
		if strings.HasPrefix(k, key) {
			kvpair = append(kvpair, &kv.KVPair{
				Key:   k,
				Value: []byte(v),
			})
		}
	}

	if len(kvpair) == 0 {
		return kvpair, errors.New("Empty list")
	}

	return kvpair, kvc.mockError
}

func (kv *MockKVClient) PutDir(key string) error {
	return kv.Put(key, "")
}

func (kv *MockKVClient) PutIntDir(key string, value int) error {
	dirName := key + "/" + strconv.Itoa(value)
	return kv.PutDir(dirName)
}

// unimplemented mock methods
func (kv *MockKVClient) DeleteTree(key string) error {
	return nil
}

func (s MockSCMClient) SetAccessToken(string) {}

func (s MockSCMClient) Name() string {
	return s.name
}

func (s MockSCMClient) HookExists(owner, repo, url string) bool {
	return true
}

func (s MockSCMClient) CreateHook(owner, repo, callback string, events []string) error {
	return nil
}

func (s MockSCMClient) CreateKey(owner, repo, key, title string) error {
	return nil
}

func (s MockSCMClient) GetFileContent(owner, repo, path, ref string) ([]byte, bool) {
	if !s.success {
		return nil, false
	}
	return []byte(validyamlSpec), true
}

func (s MockSCMClient) GetContents(owner, repo, path, ref string) (*scm.RepositoryContent, bool) {
	if !s.success {
		return nil, false
	}
	return &scm.RepositoryContent{
		Content: &validyamlSpec,
	}, true
}

func (s MockSCMClient) CreateFile(owner, repo, path, message, branch string, content []byte) (*scm.RepositoryContent, error) {
	return &scm.RepositoryContent{}, nil
}

func (s MockSCMClient) UpdateFile(owner, repo, path, blob, message, branch string, content []byte) (*scm.RepositoryContent, error) {
	return &scm.RepositoryContent{}, nil
}

func (s MockSCMClient) ListRepositories(user string) ([]*scm.Repository, error) {
	return nil, nil
}

func (s MockSCMClient) ParseHook(payload []byte, event string) (*scm.Hook, error) {
	return nil, nil
}

func (s MockSCMClient) CreateStatus(owner, repo, sha string, stageID int, stageName, state string) error {
	return nil
}

func (s MockSCMClient) GetHead(owner, repo, branch string) (string, error) {
	return "", nil
}
func (s MockSCMClient) CreateBranch(owner, repo, branchName, baseRef string) (string, error) {
	return "", nil
}
func (s MockSCMClient) CreatePullRequest(owner, repo, baseRef, headRef, title string) error {
	return nil
}
