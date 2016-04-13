package notif

import (
	"bytes"
	"encoding/json"
	"fmt"
	log "github.com/Sirupsen/logrus"
	"io/ioutil"
	"net/http"
)

type SlackDetails struct {
	Username string `json:"username"`
	Url      string `json:"url"`
	Channel  string `json:"channel"`
}

type Slack struct {
	Channel     string       `json:"channel"`
	Username    string       `json:"username"`
	Text        string       `json:"text,omitempty"`
	Emoji       string       `json:"icon_emoji"`
	MarkDown    bool         `json:"mrkdwn"`
	Icon        string       `json:"icon_url"`
	Attachments []Attachment `json:"attachments,omitempty"`
	Url         string       `json:"-"`
}

type Attachment struct {
	Color   string `json:"color"`
	Pretext string `json:"pretext,omitempty"`
	Title   string `json:"title"`
	Text    string `json:"text"`
}

func newSlackNotifier() *Slack {
	slack := &Slack{}
	slack.Attachments = []Attachment{}
	slack.MarkDown = true
	slack.Emoji = ":slack:"
	return slack

}

func (slack *Slack) updateMessageStatus(status, repo string, build int) {
	title := "*KONTINOUS* _Status_ "
	buildInfo := fmt.Sprintf("Build #%d", build)

	var msg string

	switch {
	case status == "SUCCESS":
		msg = ":tada:  *BUILD SUCCESS*"
	case status == "FAIL":
		msg = ":cry:  *BUILD FAILED*"
	}

	slack.Text = fmt.Sprintf("%s \n %s - %s \n %s ", title, repo, buildInfo, msg)
}

func (slack *Slack) addAttachment(stageName string, status string) {
	attachment := &Attachment{}
	attachment.Title = stageName

	switch {
	case status == "SUCCESS":
		attachment.Color = "good"
		attachment.Text = ":white_check_mark: SUCCESS"
	case status == "FAIL":
		attachment.Color = "danger"
		attachment.Text = ":x: FAILED"
	case status == "PENDING":
		attachment.Color = "warning"
		attachment.Text = ":warning: PENDING"
	}

	slack.Attachments = append(slack.Attachments, *attachment)
}

func buildSlackMessage(pipelineName string, buildNumber int, buildStatus string, statuses []StageStatus, metadata map[string]interface{}) *Slack {
	slack := newSlackNotifier()
	slack.updateMessageStatus(buildStatus, pipelineName, buildNumber)

	for _, stageStatus := range statuses {
		slack.addAttachment(stageStatus.Name, stageStatus.Status)
	}

	slackDetails := SlackDetails{}
	detailsJson, _ := json.Marshal(metadata)
	err := json.Unmarshal(detailsJson, &slackDetails)

	if err != nil {
		log.Println("Unable to get slack details")
		return nil
	}

	slack.Url = slackDetails.Url
	slack.Channel = slackDetails.Channel
	slack.Username = slackDetails.Username
	return slack

}

func (slack *Slack) PostMessage(pipelineName string, buildNumber int, buildStatus string, statuses []StageStatus, metadata map[string]interface{}) bool {

	slack = buildSlackMessage(pipelineName, buildNumber, buildStatus, statuses, metadata)
	data, err := json.Marshal(slack)

	if err != nil {
		log.Println("Unable to marshal payload:", err)
		return false
	}
	log.Debugf("struct = %+v, json = %s", slack, string(data))
	b := bytes.NewBuffer(data)
	if res, err := http.Post(slack.Url, "application/json", b); err != nil {
		log.Println("Unable to send data to slack:", err)
		return false
	} else {
		defer res.Body.Close()
		statusCode := res.StatusCode
		if statusCode != 200 {
			body, _ := ioutil.ReadAll(res.Body)
			log.Println("Unable to notify slack:", string(body))
			return false
		} else {
			log.Println("Slack notification sent.")
			return true
		}
	}

}
