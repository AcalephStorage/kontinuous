package model

type UserType string

const (
	GithubUser UserType = "github"
)

type User struct {
	User    string              `json:"user,required"`
	Details *UserDetails        `json:"details,required"`
	Emails  []string            `json:"emails,required"`
	Keys    map[UserType]string `json:"keys,required"`
}

type UserDetails struct {
	AvatarURL string `json:"avatarURL"`
	FullName  string `json:"fullName"`
}
