package api

import (
	"errors"
	"os"
	"strings"

	"encoding/base64"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/url"

	"github.com/dgrijalva/jwt-go"
	"github.com/emicklei/go-restful"
)

type GithubAuthResponse struct {
	AccessToken string `json:"access_token"`
}

// JWTClaims contains the claims from the jwt
type JWTClaims struct {
	GithubAccessToken string
}

type AuthResource struct {
	JWTClaims
}

var (
	claims JWTClaims

	authenticate restful.FilterFunction = func(req *restful.Request, resp *restful.Response, chain *restful.FilterChain) {
		authToken := parseToken(req)

		if authToken == "" {
			resp.WriteServiceError(http.StatusUnauthorized, restful.ServiceError{Message: "Missing Access Token!"})
			return
		}

		dsecret, _ := base64.URLEncoding.DecodeString(os.Getenv("AUTH_SECRET"))
		token, err := jwt.Parse(
			authToken,
			func(token *jwt.Token) (interface{}, error) {
				return []byte(dsecret), nil
			})

		if err == nil && token.Valid {
			claims.GithubAccessToken = ""

			if token.Claims["identities"] != nil {
				claims.GithubAccessToken = token.Claims["identities"].([]interface{})[0].(map[string]interface{})["access_token"].(string)
			}
			chain.ProcessFilter(req, resp)
		} else {
			jsonError(resp, http.StatusUnauthorized, errors.New("Unauthorized!"), "Unauthorized request")
		}
	}

	requireAccessToken restful.FilterFunction = func(req *restful.Request, resp *restful.Response, chain *restful.FilterChain) {
		if len(claims.GithubAccessToken) == 0 {
			jsonError(resp, http.StatusBadRequest, errors.New("Missing Access Token!"), "Unable to find access token")
			return
		}

		req.Request.Header.Set("Authorization", claims.GithubAccessToken)
		chain.ProcessFilter(req, resp)
	}
)

func (a *AuthResource) Register(container *restful.Container) {
	ws := new(restful.WebService)

	ws.
		Path("/authorize").
		Consumes(restful.MIME_JSON).
		Produces(restful.MIME_JSON).
		Produces(restful.MIME_JSON).
		Filter(ncsaCommonLogFormatLogger)

	ws.Route(ws.POST("").To(a.authorize).
		Doc("Generate JWT for API authentication").
		Operation("authorize"))

	container.Add(ws)
}

func (a *AuthResource) authorize(req *restful.Request, res *restful.Response) {
	dsecret, _ := base64.URLEncoding.DecodeString(os.Getenv("AUTH_SECRET"))
	authCode := req.QueryParameter("code")
	state := req.QueryParameter("state")

	if len(authCode) == 0 {
		jsonError(res, http.StatusUnauthorized, errors.New("Missing Authorization Code"), "No authorization code provided")
		return
	}

	// request url
	reqUrl := url.URL{
		Scheme: "https",
		Host:   "github.com",
		Path:   "login/oauth/access_token",
	}
	q := reqUrl.Query()
	q.Set("client_id", os.Getenv("GH_CLIENT_ID"))
	q.Set("client_secret", os.Getenv("GH_CLIENT_SECRET"))
	q.Set("code", authCode)
	q.Set("state", state)
	reqUrl.RawQuery = q.Encode()

	client := &http.Client{}

	r, err := http.NewRequest("POST", reqUrl.RequestURI(), nil)
	if err != nil {
		jsonError(res, http.StatusUnauthorized, err, "Error creating auth request")
		return
	}
	r.Header.Add("Accepts", "application/json")

	authRes, err := client.Do(r)
	if err != nil {
		jsonError(res, http.StatusUnauthorized, err, "Error requesting authorization token")
		return
	}
	defer authRes.Body.Close()

	body, err := ioutil.ReadAll(authRes.Body)
	if err != nil {
		jsonError(res, http.StatusUnauthorized, err, "Error reading response body")
		return
	}

	var ghRes GithubAuthResponse
	if err := json.Unmarshal(body, &ghRes); err != nil {
		jsonError(res, http.StatusUnauthorized, err, "Error reading json body")
		return
	}

	accessToken := ghRes.AccessToken

	jwtToken, err := CreateJWT(accessToken, string(dsecret))

	if err != nil {
		jsonError(res, http.StatusInternalServerError, err, "Unable to create jwt for user")
		return
	}

	res.WriteHeader(http.StatusCreated)
	res.Write([]byte(jwtToken))
}

func parseToken(req *restful.Request) string {
	// apply the same checking as jwt.ParseFromRequest
	if ah := req.HeaderParameter("Authorization"); ah != "" {
		if len(ah) > 6 && strings.EqualFold(ah[0:7], "BEARER ") {
			return strings.TrimSpace(ah[7:])
		}
	}
	if idt := req.QueryParameter("id_token"); idt != "" {
		return strings.TrimSpace(idt)
	}

	return ""
}
