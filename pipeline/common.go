package pipeline

import (
	"github.com/AcalephStorage/kontinuous/store/kv"
	"github.com/satori/go.uuid"
)

func generateUUID() string {
	// will I ever collide?
	return uuid.NewV4().String()
}

func generateSequentialID(namespace string, kvClient kv.KVClient) int {
	dirs, err := kvClient.GetDir(namespace)
	if err != nil {
		return 1
	}

	return len(dirs) + 1
}

func handleSaveError(namespace string, isNew bool, err error, kvClient kv.KVClient) error {
	if isNew {
		kvClient.DeleteTree(namespace)
	}

	return err
}
