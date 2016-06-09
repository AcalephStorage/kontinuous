package api

import (
	"net/http"

	log "github.com/Sirupsen/logrus"
	"github.com/emicklei/go-restful"

	"github.com/AcalephStorage/kontinuous/controller"
)

// AuthResource identifies the Auth API
type AuthResource struct {
	*controller.AuthController
	*AuthFilter
}

// AuthResponse is the response when logging in
type AuthResponse struct {
	JWT    string `json:"jwt"`
	UserID string `json:"user_id"`
}

// Register registers the auth endpoints to the restful container
func (a *AuthResource) Register(container *restful.Container) {
	ws := new(restful.WebService)

	ws.
		Path("/login").
		Produces(restful.MIME_JSON).
		Filter(ncsaCommonLogFormatLogger)

	ws.Route(ws.POST("github").To(a.githubLogin).
		Writes(AuthResponse{}).
		Doc("Generate JWT for API authentication").
		Operation("authorize"))

	container.Add(ws)
}

func (a *AuthResource) githubLogin(req *restful.Request, res *restful.Response) {
	log.Infoln("github login requested")

	authCode := req.QueryParameter("code")
	state := req.QueryParameter("state")

	user, jwt, err := a.AuthController.GithubLogin(authCode, state)
	if err != nil {
		jsonError(res, http.StatusUnauthorized, err, "unable to login to github")
		log.Infoln("user unauthorized")
		return
	}

	entity := &AuthResponse{JWT: jwt, UserID: user}
	res.WriteEntity(entity)
	log.Infoln("login successful")
}

// AuthFilter is a struct encapsulating an Authorization filter. This allows the filter to use the auth controller
type AuthFilter struct {
	*controller.AuthController
}

func (af *AuthFilter) requireBearerToken(req *restful.Request, res *restful.Response, chain *restful.FilterChain) {
	authorization := req.HeaderParameter("Authorization")

	valid, err := af.AuthController.ValidateHeaderToken(authorization)
	if err != nil {
		jsonError(res, http.StatusUnauthorized, err, "error while validating token")
		return
	}
	if !valid {
		serviceError := restful.ServiceError{Code: http.StatusUnauthorized, Message: "Unauthorized request"}
		res.WriteServiceError(http.StatusUnauthorized, serviceError)
		return
	}
	chain.ProcessFilter(req, res)
}

func (af *AuthFilter) requireIDToken(req *restful.Request, res *restful.Response, chain *restful.FilterChain) {
	idToken := req.QueryParameter("id_token")

	valid, err := af.AuthController.ValidateJWT(idToken)
	if err != nil {
		jsonError(res, http.StatusUnauthorized, err, "error while validating token")
		return
	}
	if !valid {
		serviceError := restful.ServiceError{Code: http.StatusUnauthorized, Message: "Unauthorized request"}
		res.WriteServiceError(http.StatusUnauthorized, serviceError)
		return
	}
	chain.ProcessFilter(req, res)
}
