// Copyright 2015 Google Inc. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package dingtalk

import (
	"k8s.io/heapster/events/core"
	"net/url"
	"k8s.io/api/core/v1"
	"fmt"
	"net/http"
	"encoding/json"
	"bytes"
	"github.com/golang/glog"
)

const (
	DINGTALK_SINK         = "DingTalkSink"
	WARNING           int = 2
	NORMAL            int = 1
	DEFAULT_MSG_TYPE      = "text"
	CONTENT_TYPE_JSON     = "application/json"
	MSG_TEMPLATE          = "Level:%s \nNamespace:%s \nName:%s \nMessage:%s \nReason:%s \nTimestamp:%s"
	LABE_TEMPLATE         = "%s\n"
)

/**
	dingtalk msg struct
 */
type DingTalkMsg struct {
	MsgType string       `json:"msgtype"`
	Text    DingTalkText `json:"text"`
}

type DingTalkText struct {
	Content string `json:"content"`
}

/**
	dingtalk sink usage
	--sink:dingtalk:https://oapi.dingtalk.com/robot/send?access_token=[access_token]&level=Warning&label=[label]

	level: Normal or Warning. The event level greater than global level will emit.
	label: some thing unique when you want to distinguish different k8s clusters.
 */
type DingTalkSink struct {
	Endpoint string
	Token    string
	Level    int
	Labels   [] string
}

func (d *DingTalkSink) Name() string {
	return DINGTALK_SINK
}

func (d *DingTalkSink) Stop() {
	//do nothing
}

func (d *DingTalkSink) ExportEvents(batch *core.EventBatch) {
	for _, event := range batch.Events {
		if d.isEventLevelDangerous(event.Type) {
			d.Ding(event)
		}
	}
}

func (d *DingTalkSink) isEventLevelDangerous(level string) bool {
	score := getLevel(level)
	if score >= d.Level {
		return true
	}
	return false
}

func (d *DingTalkSink) Ding(event *v1.Event) {
	msg := createMsgFromEvent(d.Labels, event)
	if msg == nil {
		glog.Warningf("failed to create msg from event,because of %v", event)
		return
	}

	msg_bytes, err := json.Marshal(msg)
	if err != nil {
		glog.Warningf("failed to marshal msg %v", msg)
		return
	}

	b := bytes.NewBuffer(msg_bytes)

	_, err = http.Post(fmt.Sprintf("https://%s?access_token=%s", d.Endpoint, d.Token), CONTENT_TYPE_JSON, b)
	if err != nil {
		glog.Errorf("failed to send msg to dingtalk,because of %s", err.Error())
		return
	}
}

func getLevel(level string) int {
	score := 0
	switch level {
	case v1.EventTypeWarning:
		score += 2;
	case v1.EventTypeNormal:
		score += 1;
	default:
		//score will remain 0
	}
	return score
}

func createMsgFromEvent(labels []string, event *v1.Event) *DingTalkMsg {
	msg := &DingTalkMsg{}
	msg.MsgType = DEFAULT_MSG_TYPE
	template := MSG_TEMPLATE
	if len(labels) > 0 {
		for _, label := range labels {
			template = fmt.Sprintf(LABE_TEMPLATE, label) + template
		}
	}
	msg.Text = DingTalkText{
		Content: fmt.Sprintf(template, event.Type, event.Namespace, event.Name, event.Message, event.Reason, event.LastTimestamp),
	}
	return msg
}

func NewDingTalkSink(uri *url.URL) (*DingTalkSink, error) {
	d := &DingTalkSink{
		Level: WARNING,
	}
	if len(uri.Host) > 0 {
		d.Endpoint = uri.Host + uri.Path
	}
	opts := uri.Query()

	if len(opts["access_token"]) >= 1 {
		d.Token = opts["access_token"][0]
	} else {
		return nil, fmt.Errorf("you must provide dingtalk bot access_token")
	}

	if len(opts["level"]) >= 1 {
		d.Level = getLevel(opts["level"][0])
	}

	//add extra labels
	if len(opts["label"]) >= 1 {
		d.Labels = opts["label"]
	}

	return d, nil
}
