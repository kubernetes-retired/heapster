package core


import (
	"k8s.io/kubernetes/pkg/api/unversioned"
	"k8s.io/kubernetes/pkg/types"
	kube_api "k8s.io/kubernetes/pkg/api"
)

type NewEvent struct {
	metadataName string `json:"metadataName,omitempty"`
	metadataGenerateName string `json:"metadataGenerateName,omitempty"`
	metadataNamespace string `json:"metadataNamespace,omitempty"`
	metadataSelfLink string `json:"metadataSelfLink,omitempty"`
	metadataUID types.UID `json:"metadataUid,omitempty"`
	metadataResourceVersion string `json:"metadataResourceVersion,omitempty"`
	metadataGeneration int64 `json:"metadataGeneration,omitempty"`

	involvedObjectKind            string    `json:"metadataKind,omitempty"`
	involvedObjectNamespace       string    `json:"metadataNamespace,omitempty"`
	involvedObjectName            string    `json:"metadataName,omitempty"`
	involvedObjectUID             types.UID `json:"metadataUid,omitempty"`
	involvedObjectAPIVersion      string    `json:"metadataApiVersion,omitempty"`
	involvedObjectResourceVersion string    `json:"metadataResourceVersion,omitempty"`
	involvedObjectFieldPath string `json:"metadataFieldPath,omitempty"`

	Reason string `json:"reason,omitempty"`

	Message string `json:"message,omitempty"`

	Component string `json:"component,omitempty"`
	Host string `json:"host,omitempty"`
	FirstTimestamp unversioned.Time `json:"firstTimestamp,omitempty"`
	LastTimestamp unversioned.Time `json:"lastTimestamp,omitempty"`
	Count int32 `json:"count,omitempty"`
	Type string `json:"type,omitempty"`
}

func ConvertEvent (oldEvent *kube_api.Event) *NewEvent {
	newEvent := NewEvent{
		metadataName: oldEvent.ObjectMeta.Name,
		metadataGenerateName: oldEvent.ObjectMeta.GenerateName,
		metadataNamespace: oldEvent.ObjectMeta.Namespace,
		metadataSelfLink: oldEvent.ObjectMeta.SelfLink,
		metadataUID: oldEvent.ObjectMeta.UID,
		metadataResourceVersion: oldEvent.ObjectMeta.ResourceVersion,
		metadataGeneration: oldEvent.ObjectMeta.Generation,
		involvedObjectKind: oldEvent.InvolvedObject.Kind,
		involvedObjectNamespace: oldEvent.InvolvedObject.Namespace,
		involvedObjectName: oldEvent.InvolvedObject.Name,
		involvedObjectUID: oldEvent.InvolvedObject.UID,
		involvedObjectAPIVersion: oldEvent.InvolvedObject.APIVersion,
		involvedObjectResourceVersion: oldEvent.InvolvedObject.ResourceVersion,
		involvedObjectFieldPath: oldEvent.InvolvedObject.FieldPath,
		Reason: oldEvent.Reason,
		Message: oldEvent.Message,
		Component: oldEvent.Source.Component,
		Host: oldEvent.Source.Host,
		FirstTimestamp: oldEvent.FirstTimestamp,
		LastTimestamp: oldEvent.LastTimestamp,
		Count: oldEvent.Count,
		Type: oldEvent.Type,
	}
	return &newEvent
}
