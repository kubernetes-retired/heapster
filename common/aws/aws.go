// Copyright 2016 Google Inc. All Rights Reserved.
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

package aws

import (
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go/aws/ec2metadata"
	"github.com/aws/aws-sdk-go/aws/session"
)

const (
	envVar = "GCP_PROJECT_ID"
)

var metadata = ec2metadata.New(session.New())

func IsOnAWS() bool {
	return metadata.Available()
}

func ProjectID() (string, error) {
	if projectId := os.Getenv(envVar); projectId != "" {
		return projectId, nil
	} else {
		return "", fmt.Errorf("no value found for: %v", envVar)
	}
}

func Zone() (string, error) {
	if ec2data, err := metadata.GetInstanceIdentityDocument(); err != nil {
		return "", err
	} else {
		return ec2data.AvailabilityZone, nil
	}
}
