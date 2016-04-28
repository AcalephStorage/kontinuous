package api

import (
	"bytes"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"encoding/base64"
	"encoding/json"

	"io/ioutil"
	"net/http"

	"github.com/Sirupsen/logrus"
	"github.com/emicklei/go-restful"

	ps "github.com/AcalephStorage/kontinuous/pipeline"
	"github.com/AcalephStorage/kontinuous/scm"
	"github.com/AcalephStorage/kontinuous/scm/github"
	"github.com/AcalephStorage/kontinuous/store/kv"
	"github.com/AcalephStorage/kontinuous/util"
	"github.com/dgrijalva/jwt-go"
)

var apiLogger = util.NewContextLogger("api")

// filters
var (
	ncsaCommonLogFormatLogger restful.FilterFunction = func(req *restful.Request, resp *restful.Response, chain *restful.FilterChain) {
		var username = "-"
		if req.Request.URL.User != nil {
			if name := req.Request.URL.User.Username(); name != "" {
				username = name
			}
		}
		chain.ProcessFilter(req, resp)
		logrus.Printf("%s - %s [%s] \"%s %s %s\" %d %d",
			strings.Split(req.Request.RemoteAddr, ":")[0],
			username,
			time.Now().Format("02/Jan/2006:15:04:05 -0700"),
			req.Request.Method,
			req.Request.URL.RequestURI(),
			req.Request.Proto,
			resp.StatusCode(),
			resp.ContentLength(),
		)
	}
)

type (
	Error struct {
		Code    int    `json:"Code"`
		Message string `json:"Message"`
	}

	GithubUser struct {
		ID int `json:"id"`
	}
)

// utils
func jsonError(res *restful.Response, statusCode int, err error, msg string) {
	logrus.WithError(err).Error(msg)
	res.WriteServiceError(statusCode, restful.NewError(statusCode, err.Error()))
}

func newSCMClient(req *restful.Request) scm.Client {
	// set github as default SCM provider
	client := new(github.Client)
	token := req.HeaderParameter("Authorization")
	accessToken := strings.Replace(token, "Bearer ", "", -1)

	switch {
	case req.HeaderParameter("X-Remote-Client") == "github", req.HeaderParameter("X-Github-Event") != "":
		client = new(github.Client)
	}
	client.SetAccessToken(accessToken)

	return client
}

// finders
func findPipeline(owner, repo string, kvClient kv.KVClient) (*ps.Pipeline, error) {
	pipeline, exists := ps.FindPipeline(owner, repo, kvClient)
	if !exists {
		err := fmt.Errorf("Pipeline for %s/%s not found.", owner, repo)
		return nil, err
	}

	return pipeline, nil
}

func findBuild(buildNumber string, pipeline *ps.Pipeline, kvClient kv.KVClient) (*ps.Build, error) {
	msg := fmt.Errorf("Build %s not found.", buildNumber)
	num, err := strconv.Atoi(buildNumber)
	if err != nil {
		return nil, msg
	}

	build, exists := pipeline.GetBuild(num, kvClient)
	if !exists {
		return nil, msg
	}

	return build, nil
}

func findStage(stageIndex string, build *ps.Build, kvClient kv.KVClient) (*ps.Stage, error) {
	msg := fmt.Errorf("Stage %s not found.", stageIndex)
	idx, err := strconv.Atoi(stageIndex)
	if err != nil {
		return nil, msg
	}

	stage, exists := build.GetStage(idx, kvClient)
	if !exists {
		return nil, msg
	}

	return stage, nil
}

func getScopedClient(userID string, kvClient kv.KVClient, req *restful.Request) (scm.Client, error) {
	client := newSCMClient(req)

	user, exists := ps.FindUser(userID, kvClient)
	if !exists {
		err := fmt.Errorf("User %s not found, cannot access remote source.", userID)
		return nil, err
	}

	client.SetAccessToken(user.Token)

	return client, nil
}

func CreateJWT(accessToken string, secret string) (string, error) {
	if accessToken == "" {
		return "", errors.New("Access Token is empty")
	}

	ghUser, err := GetGithubUser(accessToken)

	if err != nil {
		logrus.WithError(err).Errorln("Account doesn't exists!")
		return "", err
	}

	token := jwt.New(jwt.SigningMethodHS256)
	token.Claims["user_id"] = "github|" + strconv.Itoa(ghUser.ID)
	token.Claims["identities"] = []map[string]string{
		{"access_token": accessToken},
	}

	s, _ := base64.URLEncoding.DecodeString(secret)
	jwtToken, err := token.SignedString(s)
	if err != nil {
		logrus.WithError(err).Errorln("Unable to Create JWT")
		return "", errors.New("Unable to Create JWT")
	}

	return jwtToken, nil
}

func SendGithubRequest(token string, client *http.Client, method, endpoint string, data []byte) ([]byte, error) {
	url := "https://api.github.com" + endpoint
	req, err := http.NewRequest(method, url, bytes.NewReader(data))
	if err != nil {
		return nil, err
	}

	authReqToken := "token " + token
	req.Header.Add("Authorization", authReqToken)

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

func GetGithubUser(token string) (*GithubUser, error) {
	client := http.DefaultClient
	body, err := SendGithubRequest(token, client, "GET", "/user", nil)
	user := &GithubUser{}
	err = json.Unmarshal(body, user)
	if err != nil {
		return nil, err
	}
	return user, nil
}
