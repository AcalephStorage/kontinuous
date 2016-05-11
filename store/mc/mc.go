package mc

import (
	"strings"

	"github.com/minio/minio-go"
)

type MinioClient struct {
	client *minio.Client
}

func NewMinioClient(host, accessKey, secretKey string) (*MinioClient, error) {
	// ewww
	h := strings.SplitAfter(host, "://")
	var fqdn string
	if len(h) == 2 {
		fqdn = h[1]
	} else {
		fqdn = h[0]
	}

	client, err := minio.NewV4(fqdn, accessKey, secretKey, true)
	if err != nil {
		return nil, err
	}
	return &MinioClient{client}, nil
}

func (mc *MinioClient) ListObjects(bucket, prefix string) ([]minio.ObjectInfo, error) {
	var result []minio.ObjectInfo

	doneCh := make(chan struct{})

	// Indicate to our routine to exit cleanly upon return.
	defer close(doneCh)

	// List all objects from a bucket-name with a matching prefix.

	for object := range mc.client.ListObjects(bucket, prefix, true, doneCh) {
		if object.Err != nil {
			return result, object.Err
		}
		//result = append(result, object.Key)
		result = append(result, object)
	}
	return result, nil
}

func (mc *MinioClient) CopyLocally(bucket, object, file string) error {
	if err := mc.client.FGetObject(bucket, object, file); err != nil {
		return err
	}
	return nil
}

func (mc *MinioClient) DeleteTree(bucket, prefix string) error {
	doneCh := make(chan struct{})
	defer close(doneCh)

	for object := range mc.client.ListObjects(bucket, prefix, true, doneCh) {
		err := mc.DeleteObject(bucket, object.Key)
		if err != nil {
			return err
		}
	}
	return nil
}

func (mc *MinioClient) DeleteObject(bucket, object string) error {
	err := mc.client.RemoveObject(bucket, object)
	if err != nil {
		return err
	}
	return nil
}
