package api

import (
	"strings"

	log "github.com/Sirupsen/logrus"
	"github.com/emicklei/go-restful"
)

func requestLogger(req *restful.Request, res *restful.Response, chain *restful.FilterChain) {
	var user string
	if req.Attribute("user_id") != nil {
		user = req.Attribute("user_id").(string)
	}
	protocol := req.Request.Proto
	method := req.Request.Method
	uri := req.Request.URL.RequestURI()
	query := req.Request.URL.RawQuery
	filteredURI := strings.Replace(uri, query, "", -1)
	log.Debugf("--- %s - %s %s %s", user, method, filteredURI, protocol)
	chain.ProcessFilter(req, res)
	log.Debugf("--- status-code: %d, content-length: %d", res.StatusCode(), res.ContentLength())
}

// utils
func jsonError(res *restful.Response, statusCode int, err error, msg string) {
	log.WithError(err).Error(msg)
	res.WriteServiceError(statusCode, restful.NewError(statusCode, err.Error()))
}
