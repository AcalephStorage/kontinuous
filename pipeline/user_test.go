package pipeline

import (
	"testing"
)

func TestUserSave(t *testing.T) {
	kvc := setupStore()

	user := &User{"gh-user", "user001", "token"}
	result := user.Save(kvc)
	if result != nil {
		t.Error("Expected to save user without errors.")
	}

	path := userNamespace + user.RemoteID + "/remote-id"
	_, err := kvc.Get(path)
	if err != nil {
		t.Errorf("Expected to find key `%s` in KV Store.", path)
	}
}

func TestFindUser(t *testing.T) {
	id := "SampleUserID"
	kvc := setupStoreWithSampleUser(id)
	user, exists := FindUser(id, kvc)
	if !exists {
		t.Errorf("Expected to find user with remote-id `%s`", id)
	} else if user.RemoteID != id {
		t.Errorf("Expected to find User with remote-id `%s`, got `%s`", id, user.RemoteID)
	}
}
