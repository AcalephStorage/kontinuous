package model

type UserType string

const (
	GithubUser UserType = "github"
)

type User struct {
	Name   string              `json:"name,required"`
	Emails []string            `json:"emails,required"`
	UUID   string              `json:"uuid,required"`
	Keys   map[UserType]string `json:"keys,required"`
}
