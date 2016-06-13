package api

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"encoding/json"
	"io/ioutil"
	"net/http"

	"github.com/AcalephStorage/kontinuous/api"
	scm "github.com/AcalephStorage/kontinuous/scm"
	"github.com/Sirupsen/logrus"
	"github.com/spf13/viper"
)

type (
	Config struct {
		Host   string
		Token  string
		Secret string
	}

	Error struct {
		Code    int    `json:"Code"`
		Message string `json:"Message"`
	}

	PipelineData struct {
		ID          string     `json:"id"`
		Owner       string     `json:"owner"`
		Repo        string     `json:"repo"`
		Events      []string   `json:"events"`
		Login       string     `json:"login"`
		LatestBuild *BuildData `json:"latest_build"`
	}

	RepoData struct {
		Owner string `json:"owner"`
		Name  string `json:"name"`
	}

	BuildData struct {
		Number   int          `json:"number"`
		Status   string       `json:"status"`
		Created  int64        `json:"created"`
		Finished int64        `json:"finished"`
		Event    string       `json:"event"`
		Author   string       `json:"author"`
		Commit   string       `json:"commit"`
		Stages   []*StageData `json:"stages"`
	}

	StageData struct {
		Index     int    `json:"index"`
		Name      string `json:"name"`
		Type      string `json:"type"`
		Status    string `json:"status"`
		Started   int64  `json:"start-time"`
		Finished  int64  `json:"end-time"`
		JobName   string `json:"job_name"`
		Namespace string `json:"namespace"`
		PodName   string `json:"pod_name"`
	}
)

func GetConfigFromFile(file string) (*Config, error) {
	_, err := os.Stat(file)
	if err != nil {
		logrus.WithError(err).Errorln("Unable to read config file")
		return nil, err
	}

	v := viper.New()
	v.SetConfigType("yaml")
	v.SetConfigFile(file)
	v.ReadInConfig()

	config := &Config{}
	err = v.Unmarshal(&config)
	if err != nil {
		logrus.WithError(err).Errorln("Unable to read config file")
		return nil, err
	}

	err = config.validate()
	if err != nil {
		logrus.WithError(err).Errorln("Invalid config file")
		return nil, err
	}

	return config, nil
}

func (c *Config) GetPipelines(client *http.Client, pipelineName string) ([]*PipelineData, error) {
	body, err := c.sendAPIRequest(client, "GET", "/api/v1/pipelines", nil)
	if err != nil {
		return nil, err
	}
	list := []*PipelineData{}
	err = json.Unmarshal(body, &list)
	if err != nil {
		return nil, err
	}
	if len(pipelineName) > 0 {
		for _, p := range list {
			pname := strings.Join([]string{p.Owner, p.Repo}, "/")
			if pname == pipelineName {
				return []*PipelineData{p}, nil
			}
		}
		return nil, fmt.Errorf("Pipeline for `%s` not found", pipelineName)
	}
	return list, nil
}

func (c *Config) GetPipeline(client *http.Client, pipelineName string) (*PipelineData, error) {
	endpoint := fmt.Sprintf("/api/v1/pipelines/%s", pipelineName)
	body, err := c.sendAPIRequest(client, "GET", endpoint, nil)
	if err != nil {
		return nil, err
	}
	item := new(PipelineData)
	err = json.Unmarshal(body, &item)
	if err != nil {
		return nil, err
	}
	return item, nil
}

func (c *Config) GetRepos(client *http.Client) ([]*RepoData, error) {
	body, err := c.sendAPIRequest(client, "GET", "/api/v1/repositories", nil)
	if err != nil {
		return nil, err
	}
	list := []*RepoData{}
	err = json.Unmarshal(body, &list)
	if err != nil {
		return nil, err
	}

	return list, nil
}

func (c *Config) GetBuilds(client *http.Client, owner, repo string) ([]*BuildData, error) {
	endpoint := fmt.Sprintf("/api/v1/pipelines/%s/%s/builds", owner, repo)
	body, err := c.sendAPIRequest(client, "GET", endpoint, nil)
	if err != nil {
		return nil, err
	}
	list := []*BuildData{}
	err = json.Unmarshal(body, &list)
	if err != nil {
		return nil, err
	}
	return list, nil
}

func (c *Config) GetBuild(client *http.Client, owner, repo string, buildNumber int) (*BuildData, error) {
	endpoint := fmt.Sprintf("/api/v1/pipelines/%s/%s/builds/%d", owner, repo, buildNumber)
	body, err := c.sendAPIRequest(client, "GET", endpoint, nil)
	if err != nil {
		return nil, err
	}
	buildData := &BuildData{}
	err = json.Unmarshal(body, &buildData)
	if err != nil {
		return nil, err
	}
	return buildData, nil
}

func (c *Config) GetStages(client *http.Client, owner, repo string, buildNumber int) ([]*StageData, error) {
	if buildNumber == 0 {
		pipelineName := fmt.Sprintf("%s/%s", owner, repo)
		pipeline, err := c.GetPipeline(client, pipelineName)
		if err != nil {
			return nil, err
		}
		if pipeline.LatestBuild == nil {
			return nil, errors.New("No builds for pipeline.")
		}
		buildNumber = pipeline.LatestBuild.Number
	}

	endpoint := fmt.Sprintf("/api/v1/pipelines/%s/%s/builds/%d/stages", owner, repo, buildNumber)
	body, err := c.sendAPIRequest(client, "GET", endpoint, nil)
	if err != nil {
		return nil, err
	}
	list := []*StageData{}
	err = json.Unmarshal(body, &list)
	if err != nil {
		return nil, err
	}

	return list, nil
}

func (c *Config) CreatePipeline(client *http.Client, pipeline *PipelineData) error {
	user, err := api.GetGithubUser(c.Token)
	if err != nil {
		return err
	}
	pipeline.Login = "github|" + strconv.Itoa(user.ID)

	data, _ := json.Marshal(pipeline)
	_, err = c.sendAPIRequest(client, "POST", "/api/v1/pipelines", data)
	if err != nil {
		return err
	}

	err = c.CreateBuild(client, pipeline.Owner, pipeline.Repo)
	if err != nil {
		fmt.Println("Unable to create initial build")
		return err
	}

	fmt.Println("Initializing pipeline build")
	return nil
}

func (c *Config) CreateBuild(client *http.Client, owner, repo string) error {
	user, err := api.GetGithubUser(c.Token)
	if err != nil {
		return err
	}
	login := "github|" + strconv.Itoa(user.ID)
	data := fmt.Sprintf(`{"author":"%s"}`, login)
	endpoint := fmt.Sprintf("/api/v1/pipelines/%s/%s/builds", owner, repo)
	_, err = c.sendAPIRequest(client, "POST", endpoint, []byte(data))
	if err != nil {
		return err
	}

	pipelineName := fmt.Sprintf("%s/%s", owner, repo)
	pipeline, err := c.GetPipeline(client, pipelineName)
	if err != nil {
		return err
	}

	buildNumber := pipeline.LatestBuild.Number
	fmt.Print("\nPreparing pipeline build.")
	err = c.monitorBuildStatus(client, buildNumber, owner, repo, false)

	if err != nil {
		return err
	}

	return nil
}

func (c *Config) monitorBuildStatus(client *http.Client, buildNumber int, owner, repo string, started bool) error {

	build, err := c.GetBuild(client, owner, repo, buildNumber)

	if err != nil {
		return err
	}

	buildStatus := build.Status

	switch buildStatus {

	case "FAIL":
		fmt.Println("\nBuild failed.")
		return err
	case "SUCCESS":
		fmt.Println("\nBuild successful.")
		return nil

	case "WAITING":
		fmt.Print("\nBuild waiting.")
		stages, err := c.GetStages(client, owner, repo, buildNumber)
		if err != nil {
			break
		}

		message := "\nDo you want to continue? (Y/N) "
		reader := bufio.NewReader(os.Stdin)

		for stageIdx, stage := range stages {
			if stage.Status != "WAITING" {
				continue
			}

			fmt.Printf("%s", message)
			text, _ := reader.ReadString('\n')
			text = strings.ToLower(strings.TrimSpace(text))

			switch text {
			case "y":
				fallthrough
			case "yes":
				data := fmt.Sprintf(`{"status":"%s","timestamp": %v }`, buildStatus, time.Now().UnixNano())
				endpoint := fmt.Sprintf("/api/v1/pipelines/%s/%s/builds/%d/stages/%d?continue=yes", owner, repo, buildNumber, stageIdx+1)
				_, err := c.sendAPIRequest(client, "POST", endpoint, []byte(data))
				if err != nil {
					return err
				}
				fmt.Print("Resuming build.")
				time.Sleep(2 * time.Second)
			case "n":
				fallthrough
			case "no":
				fmt.Println("Build stopped.")
				return nil
			default:
				fmt.Println("Invalid input")
			}

		}

	case "RUNNING":
		if !started {
			fmt.Print("\nBuild running.")
			started = true
		} else {
			fmt.Print(".")
			time.Sleep(2 * time.Second)
		}
	}

	return c.monitorBuildStatus(client, buildNumber, owner, repo, started)
}

func (c *Config) DeletePipeline(client *http.Client, pipelineName string) error {
	endpoint := fmt.Sprintf("/api/v1/pipelines/%s", pipelineName)
	_, err := c.sendAPIRequest(client, "DELETE", endpoint, nil)
	if err != nil {
		return err
	}
	return nil
}

func (c *Config) DeleteBuild(client *http.Client, pipelineName string, buildNumber string) error {
	endpoint := fmt.Sprintf("/api/v1/pipelines/%s/builds/%v", pipelineName, buildNumber)
	_, err := c.sendAPIRequest(client, "DELETE", endpoint, nil)
	if err != nil {
		return err
	}
	return nil
}

func (c *Config) validate() error {
	missing := []string{}
	if len(c.Host) == 0 {
		missing = append(missing, "host")
	}
	if len(c.Token) == 0 {
		missing = append(missing, "token")
	}
	if len(c.Secret) <= 0 {
		missing = append(missing, "secret")
	}
	if len(missing) > 0 {
		return fmt.Errorf("Missing configuration: [%s]", strings.Join(missing, ", "))
	}
	return nil
}

func (c *Config) sendAPIRequest(client *http.Client, method, endpoint string, data []byte) ([]byte, error) {
	url := c.Host + endpoint
	req, err := http.NewRequest(method, url, bytes.NewReader(data))
	if err != nil {
		return nil, err
	}

	jwtToken, err := api.CreateJWT(c.Token, c.Secret)
	if err != nil {
		return nil, err
	}
	auth := "Bearer " + jwtToken
	req.Header.Add("Authorization", auth)
	req.Header.Add("Accept", "application/json")
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("X-Custom-Event", scm.EventCLI)

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	body, _ := ioutil.ReadAll(resp.Body)
	resp.Body.Close()

	if resp.StatusCode >= 400 && resp.StatusCode <= 500 {
		apiError := &Error{}
		err = json.Unmarshal(body, apiError)
		if err != nil {
			return nil, errors.New(resp.Status)
		}
		return nil, errors.New(apiError.Message)
	}

	return body, nil
}
