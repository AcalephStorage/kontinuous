package controller

import (
	"strings"

	"encoding/base64"

	log "github.com/Sirupsen/logrus"
	"github.com/dgrijalva/jwt-go"

	"github.com/AcalephStorage/kontinuous/model"
	"github.com/AcalephStorage/kontinuous/scm/github"
)

// AuthController handles all auth related operations
type AuthController struct {
	*UserController
	JWTSecret          string
	GithubClientID     string
	GithubClientSecret string
}

// GithubLogin handles creating JWT using github credentials
func (ac *AuthController) GithubLogin(code, state string) (username, jwt string, err error) {
	// request access token from github
	token, err := github.RequestToken(ac.GithubClientID, ac.GithubClientSecret, code, state)
	if err != nil {
		log.WithError(err).Debug("unable to request access token")
		return
	}

	// get user from github
	gc := github.NewClient(token)
	ghUser, err := gc.GetAuthenticatedUser()
	if err != nil {
		log.WithError(err).Debug("unable to get authenticated github user")
		return
	}

	// create or update user
	username = *ghUser.Login

	rawEmails, err := gc.GetAuthenticatedUserEmails()
	if err != nil {
		log.WithError(err).Debug("unable to get authenticated github user's email")
		return
	}

	emails := make([]string, len(rawEmails))
	for i, re := range rawEmails {
		emails[i] = *re.Email
	}

	user, err := ac.UserController.GetUser(model.GithubUser, username)
	if user == nil || err != nil {
		log.Info("new user login. creating new user...")
		// create user
		details := &model.UserDetails{}
		if ghUser.AvatarURL != nil {
			details.AvatarURL = *ghUser.AvatarURL
		}
		if ghUser.Name != nil {
			details.FullName = *ghUser.Name
		}
		user = &model.User{
			User:    username,
			Emails:  emails,
			Details: details,
			Keys: map[model.UserType]string{
				model.GithubUser: token,
			},
		}
		err = ac.UserController.SaveUser(model.GithubUser, username, user)
		if err != nil {
			log.WithError(err).Debug("unable to create kontinuous user")
			return
		}
		log.Info("new user created.")
	} else {
		// update user and emails
		if user.Emails == nil {
			user.Emails = []string{}
		}
		existingEmails := strings.Join(user.Emails, " ")
		for _, email := range emails {
			if !strings.Contains(existingEmails, email) {
				user.Emails = append(user.Emails, email)
			}
		}
		if user.Keys == nil {
			user.Keys = map[model.UserType]string{}
		}
		user.Keys[model.GithubUser] = token
		err = ac.UserController.SaveUser(model.GithubUser, username, user)
		if err != nil {
			log.WithError(err).Debug("unable to update kontinuous user")
			return
		}
	}

	// create JWT from user and token
	jwt, err = createJWT(user, ac.JWTSecret)
	if err != nil {
		log.WithError(err).Debug("unable to create JWT for user")
		return
	}
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

// ValidateJWT checks the JWT for validity. Empty token will return false and a nil error.
func (ac *AuthController) ValidateJWT(token string) (ok bool, err error) {
	if token == "" {
		return
	}

	token = strings.TrimSpace(token)

	secret, err := base64.URLEncoding.DecodeString(ac.JWTSecret)
	if err != nil {
		log.WithError(err).Debug("unable to base64 decode the jwt secret")
		return
	}
	jwt, err := jwt.Parse(token, func(token *jwt.Token) (interface{}, error) { return []byte(secret), nil })
	if err != nil {
		log.WithError(err).Debug("unable to parse jwt token")
		return
	}
	// other jwt checks?
	ok = jwt.Valid
	return

}

func createJWT(user *model.User, jwtSecret string) (jwtToken string, err error) {

	token := jwt.New(jwt.SigningMethodHS256)
	token.Claims["user_id"] = user.UUID

	decodedSecret, err := base64.URLEncoding.DecodeString(jwtSecret)
	if err != nil {
		log.WithError(err).Debug("unable to base64 decode jwt secret")
		return
	}

	signedToken, err := token.SignedString(decodedSecret)
	if err != nil {
		log.WithError(err).Debug("unable to sign jwt")
		return
	}

	return signedToken, nil
}
