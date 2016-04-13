package api

import (
	"errors"
	"fmt"
	"reflect"
	"testing"

	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"

	"github.com/dgrijalva/jwt-go"
)

type MockClient struct {
	BaseURL    string
	HTTPClient *http.Client
}

func assertDeepEqual(t *testing.T, actual interface{}, expected interface{}) {
	if !reflect.DeepEqual(actual, expected) {
		t.Errorf("Expected %#v - Got %#v", expected, actual)
	}
}

func newServer(code int, body string) (*httptest.Server, *MockClient) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(code)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, body)
	}))

	transport := &http.Transport{
		Proxy: func(req *http.Request) (*url.URL, error) {
			return url.Parse(server.URL)
		},
	}

	httpClient := &http.Client{Transport: transport}
	client := &MockClient{server.URL, httpClient}

	return server, client
}

func TestGetPipelines(t *testing.T) {
	body := `[{"id":"1","owner":"gh-user","repo":"gh-repo","events":["push"],"builds":[]}]`
	server, client := newServer(200, body)
	defer server.Close()

	config := &Config{Host: server.URL, Token: "sampletoken"}
	actual, _ := config.GetPipelines(client.HTTPClient, "")
	expected := []*PipelineData{}
	json.Unmarshal([]byte(body), &expected)

	assertDeepEqual(t, len(actual), 1)
	assertDeepEqual(t, actual, expected)
}

func TestGetPipelinesWithName(t *testing.T) {
	body := `[{"id":"1","owner":"gh-user","repo":"gh-repo","events":["push"],"builds":[]},{"id":"2","owner":"gh-user","repo":"gh-repo2","events":["push"],"builds":[]}]`
	server, client := newServer(200, body)
	defer server.Close()

	config := &Config{Host: server.URL, Token: "sampletoken"}
	actual, _ := config.GetPipelines(client.HTTPClient, "gh-user/gh-repo")
	expected := []*PipelineData{}
	json.Unmarshal([]byte(`[{"id":"1","owner":"gh-user","repo":"gh-repo","events":["push"],"builds":[]}]`), &expected)

	assertDeepEqual(t, len(actual), 1)
	assertDeepEqual(t, actual, expected)
}

func TestGetPipelinesWithInvalidName(t *testing.T) {
	body := `[{"id":"1","owner":"gh-user","repo":"gh-repo","events":["push"],"builds":[]}]`
	server, client := newServer(200, body)
	defer server.Close()

	config := &Config{Host: server.URL, Token: "sampletoken"}
	_, err := config.GetPipelines(client.HTTPClient, "gh-user/norepo")

	assertDeepEqual(t, err, errors.New("Pipeline for `gh-user/norepo` not found"))
}

func TestGetRepos(t *testing.T) {
	body := `[{"owner":"gh-user","repo":"gh-repo"}]`
	server, client := newServer(200, body)
	defer server.Close()

	config := &Config{Host: server.URL, Token: "sampletoken"}
	actual, _ := config.GetRepos(client.HTTPClient)
	expected := []*RepoData{}
	json.Unmarshal([]byte(body), &expected)

	assertDeepEqual(t, len(actual), 1)
	assertDeepEqual(t, actual, expected)
}

func TestGetBuilds(t *testing.T) {
	body := `[{"number":1,"status":"PENDING","event":"push","author":"gh-user","stages":[]}]`
	server, client := newServer(200, body)
	defer server.Close()

	config := &Config{Host: server.URL, Token: "sampletoken"}
	actual, _ := config.GetBuilds(client.HTTPClient, "", "")
	expected := []*BuildData{}
	json.Unmarshal([]byte(body), &expected)

	assertDeepEqual(t, len(actual), 1)
	assertDeepEqual(t, actual, expected)
}

func TestGetStages(t *testing.T) {
	body := `[{"index":1,"name":"stager","type":"command","status":"PENDING"}]`
	server, client := newServer(200, body)
	defer server.Close()

	config := &Config{Host: server.URL, Token: "sampletoken"}
	actual, _ := config.GetStages(client.HTTPClient, "", "", 0)
	expected := []*StageData{}
	json.Unmarshal([]byte(body), &expected)

	assertDeepEqual(t, len(actual), 1)
	assertDeepEqual(t, actual, expected)
}

func TestCreateJWTfromValidAccessToken(t *testing.T) {
	sampleGithubAccessToken := "validToken"
	sampleSecret := "YTRjNjlkYjU4ZTRkNWM2YjU0NTk3Njg5ZjE2OWM4NTQK"

	jwtToken, err := createJWT(sampleGithubAccessToken, sampleSecret)

	if err != nil {
		t.Fatal("Unable to create JWT")
	} else {
		s, _ := base64.URLEncoding.DecodeString(sampleSecret)
		tokenDecoded, err := jwt.Parse(jwtToken, func(token *jwt.Token) (interface{}, error) { return []byte(s), nil })
		if err == nil && tokenDecoded.Valid {
			accessToken := tokenDecoded.Claims["identities"].([]interface{})[0].(map[string]interface{})["access_token"].(string)
			assertDeepEqual(t, sampleGithubAccessToken, accessToken)
		} else {
			t.Fatal(err)
		}
	}
}

func TestCreateJWTfromInvalidAccessToken(t *testing.T) {
	sampleGithubAccessToken := ""
	sampleSecret := "YTRjNjlkYjU4ZTRkNWM2YjU0NTk3Njg5ZjE2OWM4NTQK"

	jwt, _ := createJWT(sampleGithubAccessToken, sampleSecret)

	assertDeepEqual(t, jwt, sampleGithubAccessToken)
}
