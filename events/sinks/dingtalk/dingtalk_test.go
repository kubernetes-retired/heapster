package dingtalk

import (
	"testing"

	"k8s.io/api/core/v1"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"time"
)

func TestGetLevel(t *testing.T) {
	warning := getLevel(v1.EventTypeWarning)
	normal := getLevel(v1.EventTypeNormal)
	none := getLevel("")
	assert.True(t, warning > normal)
	assert.True(t, warning == WARNING)
	assert.True(t, normal == NORMAL)
	assert.True(t, 0 == none)
}

func TestCreateMsgFromEvent(t *testing.T) {
	now := time.Now()
	labels := make([]string, 0)
	event := &v1.Event{
		Message:        "some thing wrong",
		Count:          251,
		LastTimestamp:  metav1.NewTime(now),
		FirstTimestamp: metav1.NewTime(now),
	}

	msg := createMsgFromEvent(labels, event)

	assert.True(t, msg != nil)
}
