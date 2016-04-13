package log

import (
	"fmt"
	"os"

	"encoding/base64"
	"io/ioutil"

	"github.com/Sirupsen/logrus"

	"github.com/AcalephStorage/kontinuous/store/mc"
)

const (
	bucket          = "kontinuous"
	logPathTemplate = "pipelines/%s/builds/%s/stages/%s/logs"
)

// Log represents a log from a build stage
type Log struct {
	Filename string `json:"filename"`
	Content  string `json:"content"`
}

// FetchLogs returns a list of logs for a given stage
func FetchLogs(mc *mc.MinioClient, uuid, buildNumber, stageIndex string) ([]Log, error) {
	path := fmt.Sprintf(logPathTemplate, uuid, buildNumber, stageIndex)
	logNames, err := fetchLogNames(mc, path)
	if err != nil {
		logrus.Error(err)
		return nil, err
	}
	logs := make([]Log, len(logNames))
	for i, logName := range logNames {
		content, err := fetchContent(mc, logName)
		if err != nil {
			logrus.Error(err)
			return nil, err
		}
		logs[i] = Log{
			Filename: logName,
			Content:  content,
		}
	}
	return logs, nil
}

func fetchLogNames(mc *mc.MinioClient, path string) ([]string, error) {
	objects, err := mc.ListObjects(bucket, path)
	if err != nil {
		logrus.Error(err)
		return nil, err
	}
	logNames := make([]string, len(objects))
	for i, object := range objects {
		logNames[i] = object.Key
	}
	return logNames, nil
}

func fetchContent(mc *mc.MinioClient, log string) (string, error) {
	// create temp file
	tmpfile, err := ioutil.TempFile("/tmp", "log-")
	if err != nil {
		logrus.Error(err)
		return "", err
	}

	// copy from minio to temp file
	filename := tmpfile.Name()
	if err := mc.CopyLocally(bucket, log, filename); err != nil {
		logrus.Error(err)
		return "", nil
	}

	// read content
	content, err := ioutil.ReadFile(filename)
	if err != nil {
		return "", err
	}
	defer os.Remove(filename)

	// encrypt content
	encodedContent := base64.StdEncoding.EncodeToString(content)
	return encodedContent, nil
}
