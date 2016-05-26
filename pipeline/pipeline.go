package pipeline

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"encoding/base64"

	"github.com/choodur/drone/shared/crypto"
	etcd "github.com/coreos/etcd/client"
	"github.com/dgrijalva/jwt-go"

	"encoding/json"
	"github.com/AcalephStorage/kontinuous/scm"
	"github.com/AcalephStorage/kontinuous/store/kv"
	"github.com/AcalephStorage/kontinuous/store/mc"
)

const (
	// PipelineYAML is the YAML file that holds the pipeline specifications
	PipelineYAML = ".pipeline.yml"

	// BuildFailure indicates that the build has failed
	BuildFailure = "FAIL"

	// BuildPending indicates that the build is pending
	BuildPending = "PENDING"

	// BuildRunning indicates that the build is running
	BuildRunning = "RUNNING"

	// BuildSuccess indicates that the build was successful
	BuildSuccess = "SUCCESS"

	claimsIssuer      = "http://kontinuous.io"
	claimsSubject     = "kontinuous"
	buildEndpoint     = "%s/api/v1/pipelines/%s/%s/builds"
	appNamespace      = "/kontinuous/"
	userNamespace     = appNamespace + "users/"
	pipelineNamespace = appNamespace + "pipelines/"
)

type (
	// Key contains the public/private keypair used for deployments
	Key struct {
		Private string
		Public  string
	}

	// NextJobInfo contains the data needed to get the details for creating a job
	NextJobInfo struct {
		Commit      string
		BuildNumber int
		StageIndex  int
	}

	Notifier struct {
		Type      string                 `json:"type"`
		Metadata  map[string]interface{} `json:"metadata, omitempty"`
		Namespace string                 `json:"-"`
	}
)

// Pipeline contains the details of a repo required for a build
type Pipeline struct {
	ID                string                 `json:"id"`
	Name              string                 `json:"-"`
	Owner             string                 `json:"owner"`
	Repo              string                 `json:"repo"`
	Events            []string               `json:"events,omitempty"`
	Builds            []*Build               `json:"builds,omitempty"`
	LatestBuildNumber int                    `json:"-"`
	LatestBuild       *BuildSummary          `json:"latest_build,omitempty"`
	Keys              Key                    `json:"-"`
	Login             string                 `json:"login"`
	Source            string                 `json:"-"`
	Notifiers         []*Notifier            `json:"notif,omitempty"`
	Secrets           []string               `json:"secrets,omitempty"`
	Vars              map[string]interface{} `json:"vars, omitempty"`
}

// CreatePipeline persists the pipeline details and setups
// the webhook and deploy keys(used for builds) to the remote SCM
func CreatePipeline(p *Pipeline, c scm.Client, k kv.KVClient) (err error) {
	// check if pipeline exists
	// TODO: add source checking
	if _, exists := FindPipeline(p.Owner, p.Repo, k); exists {
		return fmt.Errorf("Pipeline %s/%s already exists!", p.Owner, p.Repo)
	}

	// check if repo exists
	source, exists := c.GetRepository(p.Owner, p.Repo)
	if !exists {
		return fmt.Errorf("Repository has no remote source from %s.", c.Name())
	}

	// check if user has admin rights
	if !source.IsAdmin() {
		return fmt.Errorf("Admin rights for %s/%s is required to create a pipeline.",
			source.Owner,
			source.Name)
	}

	// validate
	p.ID = generateUUID()
	p.Source = c.Name()
	if err = p.Validate(); err != nil {
		return err
	}

	if err = p.generateKeys(); err != nil {
		return err
	}

	// persist pipeline
	if err = p.Save(k); err != nil {
		return err
	}

	// create hook
	secret, err := p.GenerateHookSecret(os.Getenv("AUTH_SECRET"))
	if err != nil {
		return err
	}

	deployURL := os.Getenv("KONTINUOUS_URL")
	callback := fmt.Sprintf(buildEndpoint+"?id_token=%s", deployURL, p.Owner, p.Repo, secret)

	// hook might already be created from a previous install
	// TODO: ensure hooks are unique per install
	if !c.HookExists(p.Owner, p.Repo, callback) {
		if err = c.CreateHook(p.Owner, p.Repo, callback, p.Events); err != nil {
			return err
		}
	}

	// create deploy keys for repo
	// always create a new one since we persist this with the pipeline details
	if err = c.CreateKey(p.Owner, p.Repo, p.Keys.Public, callback); err != nil {
		return err
	}

	return nil
}

// FindPipeline returns a pipeline based on the given owner & repo details
func FindPipeline(owner, repo string, kvClient kv.KVClient) (*Pipeline, bool) {
	pipelineDirs, err := kvClient.GetDir(pipelineNamespace)
	if err != nil || etcd.IsKeyNotFound(err) {
		return nil, false
	}

	for _, pair := range pipelineDirs {
		namespace := strings.TrimPrefix(pair.Key, pipelineNamespace)
		id := strings.Split(namespace, ":")
		if id[0] == owner && id[1] == repo {
			path := pipelineNamespace + namespace
			pipeline := getPipeline(path, kvClient)
			return pipeline, true
		}
	}

	return nil, false
}

// FindAllPipelines returns all the pipelines
func FindAllPipelines(kvClient kv.KVClient) ([]*Pipeline, error) {
	pipelineDirs, err := kvClient.GetDir(pipelineNamespace)
	if err != nil {
		if etcd.IsKeyNotFound(err) {
			return make([]*Pipeline, 0), nil
		}
		return nil, err
	}

	pipelines := []*Pipeline{}
	for _, pair := range pipelineDirs {
		// TODO handle errors when getting data from etcd
		namespace := strings.TrimPrefix(pair.Key, pipelineNamespace)
		path := pipelineNamespace + namespace
		pipeline := getPipeline(path, kvClient)
		pipelines = append(pipelines, pipeline)
	}

	return pipelines, nil
}

func getPipeline(path string, kvClient kv.KVClient) *Pipeline {
	p := new(Pipeline)

	keys := Key{}
	keys.Public, _ = kvClient.Get(path + "/keys/public")
	keys.Private, _ = kvClient.Get(path + "/keys/private")
	events, _ := kvClient.Get(path + "/events")
	secrets, _ := kvClient.Get(path + "/secrets")
	vars, _ := kvClient.Get(path + "/vars")

	p.ID, _ = kvClient.Get(path + "/uuid")
	p.Repo, _ = kvClient.Get(path + "/repo")
	p.Owner, _ = kvClient.Get(path + "/owner")
	p.Login, _ = kvClient.Get(path + "/login")
	p.Source, _ = kvClient.Get(path + "/source")
	p.LatestBuildNumber, _ = kvClient.GetInt(path + "/latest-build")
	p.LatestBuild, _ = p.GetBuildSummary(p.LatestBuildNumber, kvClient)
	p.Events = strings.Split(events, ",")
	p.Keys = keys
	p.Name = p.fullName()
	p.Secrets = strings.Split(secrets, ",")
	json.Unmarshal([]byte(vars), &p.Vars)

	pipelineNotifiers := []*Notifier{}
	notifiers, _ := kvClient.Get(path + "/notif/type")

	if len(notifiers) > 0 {
		notifierType := strings.Split(notifiers, " ")
		notifnamespace, _ := kvClient.Get(path + "/notif/namespace")

		for _, notifier := range notifierType {
			pipelineNotifier := &Notifier{}
			pipelineNotifier.Type = notifier
			pipelineNotifier.Namespace = notifnamespace
			pipelineNotifiers = append(pipelineNotifiers, pipelineNotifier)
		}
		p.Notifiers = pipelineNotifiers
	}
	return p
}

// GenerateHookSecret generates the secret for web hooks used for hook authentication
func (p *Pipeline) GenerateHookSecret(secret string) (string, error) {
	token := jwt.New(jwt.SigningMethodHS256)
	token.Claims = map[string]interface{}{
		"iss":   claimsIssuer,
		"sub":   claimsSubject,
		"owner": p.Owner,
		"repo":  p.Repo,
	}
	s, _ := base64.URLEncoding.DecodeString(secret)
	tokenString, err := token.SignedString(s)
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

// Save persists the pipeline details to `etcd`
func (p *Pipeline) Save(kvClient kv.KVClient) (err error) {
	p.Name = p.fullName()
	path := pipelineNamespace + p.Name
	events := strings.Join(p.Events, ",")
	pipelineSecrets := strings.Join(p.Secrets, ",")
	isNew := false

	_, err = kvClient.GetDir(path)
	if err != nil || etcd.IsKeyNotFound(err) {
		isNew = true
	}

	if err = kvClient.Put(path+"/uuid", p.ID); err != nil {
		return handleSaveError(path, isNew, err, kvClient)
	}
	if err = kvClient.Put(path+"/repo", p.Repo); err != nil {
		return handleSaveError(path, isNew, err, kvClient)
	}
	if err = kvClient.Put(path+"/owner", p.Owner); err != nil {
		return handleSaveError(path, isNew, err, kvClient)
	}
	if err = kvClient.Put(path+"/events", events); err != nil {
		return handleSaveError(path, isNew, err, kvClient)
	}
	if err = kvClient.Put(path+"/keys/public", p.Keys.Public); err != nil {
		return handleSaveError(path, isNew, err, kvClient)
	}
	if err = kvClient.Put(path+"/keys/private", p.Keys.Private); err != nil {
		return handleSaveError(path, isNew, err, kvClient)
	}
	if err = kvClient.Put(path+"/login", p.Login); err != nil {
		return handleSaveError(path, isNew, err, kvClient)
	}
	if err = kvClient.Put(path+"/source", p.Source); err != nil {
		return handleSaveError(path, isNew, err, kvClient)
	}

	vars, _ := json.Marshal(p.Vars)
	if err = kvClient.Put(path+"/vars", string(vars)); err != nil {
		return handleSaveError(path, isNew, err, kvClient)
	}

	if err = kvClient.Put(path+"/secrets", pipelineSecrets); err != nil {
		return handleSaveError(path, isNew, err, kvClient)
	}

	if !isNew {
		if err = kvClient.PutInt(path+"/latest-build", p.LatestBuildNumber); err != nil {
			return handleSaveError(path, isNew, err, kvClient)
		}
	}

	if p.Notifiers != nil && len(p.Notifiers) > 0 {

		types := make([]string, len(p.Notifiers))
		for _, notifier := range p.Notifiers {
			types = append(types, notifier.Type)
		}

		notifValue := strings.Join(types, " ")
		if err = kvClient.Put(path+"/notif/type", notifValue); err != nil {
			return handleSaveError(path, isNew, err, kvClient)
		}

		if err = kvClient.Put(path+"/notif/namespace", p.Notifiers[0].Namespace); err != nil {
			return handleSaveError(path, isNew, err, kvClient)
		}
	}

	return nil
}

func (p *Pipeline) DeletePipeline(kvClient kv.KVClient, mcClient *mc.MinioClient) (err error) {
	path := fmt.Sprintf("%s%s", pipelineNamespace, p.fullName())
	pipelinePrefix := fmt.Sprintf("pipelines/%s", p.ID)
	bucket := "kontinuous"

	if err := kvClient.DeleteTree(path); err != nil {
		return err
	}

	if err := mcClient.DeleteTree(bucket, pipelinePrefix); err != nil {
		return err
	}
	return nil

}

// Validate checks if the required values for a pipeline are present
func (p *Pipeline) Validate() error {
	if p.Owner == "" {
		return errors.New("Owner is required.")
	}
	if p.Repo == "" {
		return errors.New("Repo is required.")
	}
	if p.Login == "" {
		return errors.New("Login is required.")
	}
	if p.Source == "" {
		return errors.New("Source is required.")
	}

	optEvents := []string{}
	reqEvents := []string{scm.EventPush}
	allEvents := append(optEvents, reqEvents...)
	if len(p.Events) == 0 {
		return fmt.Errorf("Events is required. Must be any of the following: %s",
			strings.Join(allEvents, ", "))
	}

	missingReqEvents := []string{}
	for _, req := range reqEvents {
		missing := req
		for _, event := range p.Events {
			if req == event {
				missing = ""
				break
			}
		}
		if missing != "" {
			missingReqEvents = append(missingReqEvents, missing)
		}
	}

	if len(missingReqEvents) != 0 {
		return fmt.Errorf("The following events are required: %s",
			strings.Join(missingReqEvents, ", "))
	}

	return nil
}

// Definition retrieves the pipeline definition from a given reference
func (p *Pipeline) Definition(ref string, c scm.Client) (*Definition, error) {
	file, ok := c.GetFileContent(p.Owner, p.Repo, PipelineYAML, ref)
	if !ok {
		return nil, fmt.Errorf("%s not found for %s/%s on %s",
			PipelineYAML,
			p.Owner,
			p.Repo,
			ref)
	}

	// parse definition
	definition, err := GetDefinition(file)
	if err != nil {
		return nil, err
	}

	return definition, nil
}

// GetAllBuildsSummary fetches all summarized builds from the store
func (p *Pipeline) GetAllBuildsSummary(kvClient kv.KVClient) ([]*BuildSummary, error) {
	namespace := fmt.Sprintf("%s%s/builds", pipelineNamespace, p.fullName())
	buildDirs, err := kvClient.GetDir(namespace)
	if err != nil {
		if etcd.IsKeyNotFound(err) {
			return make([]*BuildSummary, 0), nil
		}
		return nil, err
	}

	builds := make([]*BuildSummary, len(buildDirs))
	for i, pair := range buildDirs {
		builds[i] = getBuildSummary(pair.Key, kvClient)
	}

	return builds, nil
}

// GetBuilds fetches all builds from the store
func (p *Pipeline) GetBuilds(kvClient kv.KVClient) ([]*Build, error) {
	namespace := fmt.Sprintf("%s%s/builds", pipelineNamespace, p.fullName())
	buildDirs, err := kvClient.GetDir(namespace)
	if err != nil {
		if etcd.IsKeyNotFound(err) {
			return make([]*Build, 0), nil
		}
		return nil, err
	}

	p.Builds = make([]*Build, len(buildDirs))
	for i, pair := range buildDirs {
		p.Builds[i] = getBuild(pair.Key, kvClient)
	}

	return p.Builds, nil
}

// GetBuild fetches a specific build by its number
func (p *Pipeline) GetBuild(num int, kvClient kv.KVClient) (*Build, bool) {
	path := fmt.Sprintf("%s%s:%s/builds/%d", pipelineNamespace, p.Owner, p.Repo, num)
	_, err := kvClient.GetDir(path)
	if err != nil || etcd.IsKeyNotFound(err) {
		return nil, false
	}

	return getBuild(path, kvClient), true
}

// GetBuildSummary fetches a specific build by its number and returns a summarized details
func (p *Pipeline) GetBuildSummary(num int, kvClient kv.KVClient) (*BuildSummary, bool) {
	path := fmt.Sprintf("%s%s:%s/builds/%d", pipelineNamespace, p.Owner, p.Repo, num)
	_, err := kvClient.GetDir(path)
	if err != nil || etcd.IsKeyNotFound(err) {
		return nil, false
	}

	return getBuildSummary(path, kvClient), true
}

// CreateBuild persists build & stage details based on the given definition
func (p *Pipeline) CreateBuild(b *Build, stages []*Stage, kvClient kv.KVClient, scmClient scm.Client) error {
	b.Created = time.Now().UnixNano()
	b.CurrentStage = 1
	b.Status = BuildPending
	b.Pipeline = p.fullName()
	b.ID = generateUUID()
	b.Number = generateSequentialID(fmt.Sprintf("%s%s/builds", pipelineNamespace, b.Pipeline), kvClient)
	b.Stages = stages

	if err := b.Save(kvClient); err != nil {
		return err
	}

	if b.Branch != b.Commit {
		for _, stage := range b.Stages {
			if err := scmClient.CreateStatus(p.Owner, p.Repo, b.Commit, stage.Index, stage.Name, scm.StatePending); err != nil {
				return err
			}
		}
	}

	p.LatestBuildNumber = b.Number
	if err := p.Save(kvClient); err != nil {
		return err
	}

	return nil
}

// PrepareBuildStage gets the details needed to run a job
func (p *Pipeline) PrepareBuildStage(n *NextJobInfo, scmClient scm.Client) (*Definition, *JobBuildInfo, error) {
	definition, err := p.Definition(n.Commit, scmClient)
	if err != nil {
		return nil, nil, err
	}

	jobInfo := &JobBuildInfo{
		PipelineUUID: p.ID,
		Build:        strconv.Itoa(n.BuildNumber),
		Stage:        strconv.Itoa(n.StageIndex),
		Commit:       n.Commit,
		User:         scmClient.AccessToken(),
		Repo:         p.Repo,
		Owner:        p.Owner,
	}

	return definition, jobInfo, nil
}

func (p *Pipeline) fullName() string {
	return p.Owner + ":" + p.Repo
}

func (p *Pipeline) generateKeys() error {
	// generate keys
	key, err := crypto.GeneratePrivateKey()
	if err != nil {
		return err
	}

	// assign to pipeline
	p.Keys = Key{
		Public:  string(crypto.MarshalPublicKey(&key.PublicKey)),
		Private: string(crypto.MarshalPrivateKey(key)),
	}

	return nil
}

func (p *Pipeline) UpdatePipeline(definition *Definition, kvClient kv.KVClient) {

	pipelineNotifiers := []*Notifier{}

	for _, notifier := range definition.Spec.Template.Notifiers {
		namespace := "default"
		if definition.Metadata["namespace"] != "" {
			namespace = definition.Metadata["namespace"].(string)
		}
		notifier.Namespace = namespace
		pipelineNotifiers = append(pipelineNotifiers, notifier)
	}

	p.Notifiers = pipelineNotifiers
	p.Secrets = definition.Spec.Template.Secrets
	p.Vars = definition.Spec.Template.Vars

	p.Save(kvClient)

}

// GetDefinitionFile fetches the definition file (PipelineYAML) from the pipeline's repository
// returns the content (possibly encoded in base64, see scm API) and
// the SHA of the file (blob)
func (p *Pipeline) GetDefinitionFile(c scm.Client, ref string) (*DefinitionFile, bool) {
	file, exists := c.GetContents(p.Owner, p.Repo, PipelineYAML, ref)
	if !exists {
		return nil, false
	}
	return &DefinitionFile{
		Content: file.Content,
		SHA:     file.SHA,
	}, true
}

// UpdateDefinitionFile commits the changes of the definition file (PipelineYAML)
// or creates the file if it does not exist
// either directly to the default branch
// or through a pull request
func (p *Pipeline) UpdateDefinitionFile(c scm.Client, file *DefinitionFile, commit map[string]string) (*DefinitionFile, error) {
	return file.SaveToRepo(c, p.Owner, p.Repo, commit)
}
