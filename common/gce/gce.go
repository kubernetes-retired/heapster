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

package gce

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"time"

	"cloud.google.com/go/compute/metadata"
	"github.com/golang/glog"
)

const (
	waitForGCEInterval = 5 * time.Second
	waitForGCETimeout  = 3 * time.Minute
	gcpCredentialsEnv  = "GOOGLE_APPLICATION_CREDENTIALS"
	gcpProjectIdEnv    = "GOOGLE_PROJECT_ID"
)

func EnsureOnGCE() error {
	for start := time.Now(); time.Since(start) < waitForGCETimeout; time.Sleep(waitForGCEInterval) {
		glog.Infof("Waiting for GCE metadata to be available")
		if metadata.OnGCE() {
			return nil
		}
	}
	return fmt.Errorf("not running on GCE")
}

func GetProjectId() (string, error) {
	// Try the environment variable first.
	if projectId := getProjectIdFromEnv(); projectId != "" {
		glog.Infof("Using GCP project ID from environment variable: %s", projectId)
		return projectId, nil
	}

	// Try the default credentials file, if the environment variable is set.
	if file := os.Getenv(gcpCredentialsEnv); file != "" {
		projectId, err := getProjectIdFromFile(file)
		if err == nil {
			glog.Infof("Using GCP project ID from default credentials file: %s", projectId)
			return projectId, nil
		} else {
			glog.Infof("Unable to get project ID from default credentials file: %s", err)
		}
	}

	// Finally, fallback on the metadata service.
	glog.Info("Checking GCE metadata service for GCP project ID")
	if err := EnsureOnGCE(); err != nil {
		return "", err
	}
	projectId, err := metadata.ProjectID()
	if err != nil {
		return "", err
	}
	return projectId, nil
}

func getProjectIdFromEnv() string {
	return os.Getenv(gcpProjectIdEnv)
}

func getProjectIdFromFile(file string) (string, error) {
	conf, err := ioutil.ReadFile(file)
	if err != nil {
		return "", fmt.Errorf("error reading default credentials file: %s", err)
	}
	var gcpConfig struct {
		ProjectId *string `json:"project_id"`
	}
	err = json.Unmarshal(conf, &gcpConfig)
	if err != nil {
		return "", fmt.Errorf("error unmarshalling default credentials file: %s", err)
	}
	if gcpConfig.ProjectId == nil {
		return "", fmt.Errorf("field project_id not found")
	}
	return *gcpConfig.ProjectId, nil
}
