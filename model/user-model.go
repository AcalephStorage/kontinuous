package model

type UserType string

const (
	GithubUser UserType = "github"
)

type User struct {
	Name    string              `json:"name,required"`
	Details *UserDetails        `json:"details,required"`
	Emails  []string            `json:"emails,required"`
	UUID    string              `json:"uuid,required"`
	Keys    map[UserType]string `json:"keys,required"`
}

type UserDetails struct {
	AvatarURL string `json:"avatarURL"`
	FullName  string `json:"fullName"`
}
