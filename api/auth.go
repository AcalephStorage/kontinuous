package api

import (
	"errors"
	"os"
	"strings"

	"encoding/base64"
	"net/http"

	"github.com/dgrijalva/jwt-go"
	"github.com/emicklei/go-restful"
)

// JWTClaims contains the claims from the jwt
type JWTClaims struct {
	GithubAccessToken string
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
