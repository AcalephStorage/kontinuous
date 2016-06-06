package controller

import (
	"strings"

	"encoding/base64"

	"github.com/dgrijalva/jwt-go"
)

// AuthController
type AuthController struct {
	JWTSecret          string
	GithubClientID     string
	GithubClientSecret string
}

// type GithubAuthResponse struct {
// 	AccessToken string `json:"access_token"`
// }

// a.AuthController.GithubLogin(authCode, state)

// // request url
// reqUrl := url.URL{
// 	Scheme: "https",
// 	Host:   "github.com",
// 	Path:   "login/oauth/access_token",
// }
// q := reqUrl.Query()
// q.Set("client_id", os.Getenv("GITHUB_CLIENT_ID"))
// q.Set("client_secret", os.Getenv("GITHUB_CLIENT_SECRET"))
// q.Set("code", authCode)
// q.Set("state", state)
// reqUrl.RawQuery = q.Encode()

// client := &http.Client{}

// r, err := http.NewRequest("POST", reqUrl.String(), nil)
// if err != nil {
// 	jsonError(res, http.StatusUnauthorized, err, "Error creating auth request")
// 	return
// }
// r.Header.Add("Accept", "application/json")

// authRes, err := client.Do(r)
// if err != nil {
// 	jsonError(res, http.StatusUnauthorized, err, "Error requesting authorization token")
// 	return
// }
// defer authRes.Body.Close()

// body, err := ioutil.ReadAll(authRes.Body)
// if err != nil {
// 	jsonError(res, http.StatusUnauthorized, err, "Error reading response body")
// 	return
// }

// var ghRes GithubAuthResponse
// if err := json.Unmarshal(body, &ghRes); err != nil {
// 	jsonError(res, http.StatusUnauthorized, err, "Error reading json body")
// 	return
// }

// accessToken := ghRes.AccessToken

// jwtToken, err := CreateJWT(accessToken, string(dsecret))
// if err != nil {
// 	jsonError(res, http.StatusUnauthorized, err, "Unable to create jwt for user")
// 	return
// }

// ghUser, err := GetGithubUser(accessToken)
// if err != nil {
// 	jsonError(res, http.StatusUnauthorized, err, "Unable to get github user")
// 	return
// }

// userID := fmt.Sprintf("github|%v", ghUser.ID)
// user := &pipeline.User{
// 	Name:     ghUser.Login,
// 	RemoteID: userID,
// 	Token:    accessToken,
// }
// if err := user.Save(a.KVClient); err != nil {
// 	jsonError(res, http.StatusUnauthorized, err, "Unable to register user")
// 	return
// }

// GithubLogin handles creating JWT using github credentials
func (ac *AuthController) GithubLogin(code, state string) (user, token string, err error) {
	// request access token from github
	//
	return
}

// ValidateHeaderToken validates the given token if it is authenticated
func (ac *AuthController) ValidateHeaderToken(authToken string) (ok bool, err error) {

	// bearer token
	if len(authToken) > 6 && authToken[:7] == "Bearer " {
		jwt := strings.TrimSpace(authToken[7:])
		return ac.ValidateJWT(jwt)
	}
	return
}

// ValidateJWT checks the JWT for validity
func (ac *AuthController) ValidateJWT(token string) (ok bool, err error) {
	if token == "" {
		return
	}

	token = strings.TrimSpace(token)

	secret, err := base64.URLEncoding.DecodeString(ac.JWTSecret)
	if err != nil {
		return
	}
	jwt, err := jwt.Parse(token, func(token *jwt.Token) (interface{}, error) { return []byte(secret), nil })

	// other jwt checks?
	ok = jwt.Valid
	return

}
