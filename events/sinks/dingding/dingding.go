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

package dingding

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

type DingdingMsg struct {
	MsgType string       `json:"msgtype"`
	Text    DingdingText `json:"text"`
}

type DingdingText struct {
	Content string `json:"content"`
}

type DingdingSink struct {
	Endpoint string
	Token    string
	Level    int
}

const (
	DINGDING_SINK         = "DingdingSink"
	WARNING           int = 2
	NORMAL            int = 1
	DEFAULT_MSG_TYPE      = "text"
	CONTENT_TYPE_JSON     = "application/json"
)

func (d *DingdingSink) Name() string {
	return DINGDING_SINK
}

func (d *DingdingSink) Stop() {
	//do nothing
}

func (d *DingdingSink) ExportEvents(batch *core.EventBatch) {
	for _, event := range batch.Events {
		if d.isEventLevelDangerous(event.Type) {
			d.Ding(event)
		}
	}
}

func (d *DingdingSink) isEventLevelDangerous(level string) bool {
	score := getLevel(level)
	if score >= d.Level {
		return true
	}
	return false
}

func (d *DingdingSink) Ding(event *v1.Event) {
	msg := createMsgFromEvent(event)
	if msg == nil {
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
		glog.Errorf("failed to send msg to dingding,because of %s", err.Error())
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

func createMsgFromEvent(event *v1.Event) *DingdingMsg {
	msg := &DingdingMsg{}
	msg.MsgType = DEFAULT_MSG_TYPE
	msg.Text = DingdingText{
		Content: fmt.Sprintf("Level:%s \nNamespace:%s \nName:%s \nMessage:%s \nReason:%s \nTimestamp:%s", event.Type, event.Namespace, event.Name, event.Message, event.Reason, event.LastTimestamp.Local()),
	}
	return msg
}

func NewDingdingSink(uri *url.URL) (*DingdingSink, error) {
	d := &DingdingSink{
		Level: NORMAL,
	}
	if len(uri.Host) > 0 {
		d.Endpoint = uri.Host + uri.Path
	}
	opts := uri.Query()

	if len(opts["access_token"]) >= 1 {
		d.Token = opts["access_token"][0]
	} else {
		return nil, fmt.Errorf("you must provide dingding bot access_token")
	}

	if len(opts["level"]) >= 1 {
		d.Level = getLevel(opts["level"][0])
	}
	return d, nil
}
