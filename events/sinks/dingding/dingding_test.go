package dingding

import (
	"testing"
	"time"

	kube_api "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/heapster/events/core"
)

func TestNewDingdingSink(t *testing.T) {
	now := time.Now()
	event := kube_api.Event{
		Message:        "Event Message",
		Count:          100,
		LastTimestamp:  metav1.NewTime(now),
		FirstTimestamp: metav1.NewTime(now),
	}
	batch := core.EventBatch{
		Timestamp: now,
		Events:    []*kube_api.Event{&event},
	}

	t.Log(batch)
}
