// +build !ignore_autogenerated

/*
Copyright 2017 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// This file was autogenerated by deepcopy-gen. Do not edit it manually!

package admission

import (
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
)

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *AdmissionReview) DeepCopyInto(out *AdmissionReview) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new AdmissionReview.
func (in *AdmissionReview) DeepCopy() *AdmissionReview {
	if in == nil {
		return nil
	}
	out := new(AdmissionReview)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *AdmissionReview) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	} else {
		return nil
	}
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *AdmissionReviewSpec) DeepCopyInto(out *AdmissionReviewSpec) {
	*out = *in
	out.Kind = in.Kind
	if in.Object == nil {
		out.Object = nil
	} else {
		out.Object = in.Object.DeepCopyObject()
	}
	if in.OldObject == nil {
		out.OldObject = nil
	} else {
		out.OldObject = in.OldObject.DeepCopyObject()
	}
	out.Resource = in.Resource
	in.UserInfo.DeepCopyInto(&out.UserInfo)
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new AdmissionReviewSpec.
func (in *AdmissionReviewSpec) DeepCopy() *AdmissionReviewSpec {
	if in == nil {
		return nil
	}
	out := new(AdmissionReviewSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *AdmissionReviewStatus) DeepCopyInto(out *AdmissionReviewStatus) {
	*out = *in
	if in.Result != nil {
		in, out := &in.Result, &out.Result
		if *in == nil {
			*out = nil
		} else {
			*out = new(v1.Status)
			(*in).DeepCopyInto(*out)
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new AdmissionReviewStatus.
func (in *AdmissionReviewStatus) DeepCopy() *AdmissionReviewStatus {
	if in == nil {
		return nil
	}
	out := new(AdmissionReviewStatus)
	in.DeepCopyInto(out)
	return out
}
