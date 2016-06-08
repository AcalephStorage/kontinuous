package pipeline

import (
	"strings"

	etcd "github.com/coreos/etcd/client"

	"github.com/AcalephStorage/kontinuous/store/kv"
)

// User contains user details from the remote SCM
type User struct {
	Name     string `json:"name"`
	RemoteID string `json:"id"`
	Token    string `json:"access_token"`
}

// FindUser finds a User given a remote ID
func FindUser(remoteID string, kvClient kv.Client) (*User, bool) {
	userDirs, err := kvClient.GetDir(userNamespace)
	if err != nil || etcd.IsKeyNotFound(err) {
		return nil, false
	}

	for _, pair := range userDirs {
		id := strings.TrimPrefix(pair.Key, userNamespace)
		if id == remoteID {
			path := userNamespace + id
			return getUser(path, kvClient), true
		}
	}

	return nil, false
}

func getUser(path string, kvClient kv.Client) *User {
	u := new(User)
	u.Name, _ = kvClient.Get(path + "/name")
	u.RemoteID, _ = kvClient.Get(path + "/remote-id")
	u.Token, _ = kvClient.Get(path + "/token")
	return u
}

// Save persists User details to the store
func (u *User) Save(kvClient kv.Client) (err error) {
	path := userNamespace + u.RemoteID // remoteID is unique

	if err = kvClient.Put(path+"/name", u.Name); err != nil {
		kvClient.DeleteTree(path)
		return err
	}
	if err = kvClient.Put(path+"/remote-id", u.RemoteID); err != nil {
		kvClient.DeleteTree(path)
		return err
	}
	if err = kvClient.Put(path+"/token", u.Token); err != nil {
		kvClient.DeleteTree(path)
		return err
	}

	return nil
}
