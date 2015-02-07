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

package gcm

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/GoogleCloudPlatform/gcloud-golang/compute/metadata"
)

type authToken struct {
	// The actual token.
	AccessToken string `json:"access_token"`

	// Number of seconds in which the token will expire
	ExpiresIn int `json:"expires_in"`

	// Type of token.
	TokenType string `json:"token_type"`
}

// Get a token for performing GCM requests.
func getToken() (authToken, error) {
	rawToken, err := metadata.Get("instance/service-accounts/default/token")
	if err != nil {
		return authToken{}, err
	}

	var token authToken
	err = json.Unmarshal([]byte(rawToken), &token)
	if err != nil {
		return authToken{}, fmt.Errorf("failed to unmarshal service account token with output %q: %v", rawToken, err)
	}

	return token, err
}

// Checks that we have the required service scopes.
func checkServiceAccounts() error {
	scopes, err := metadata.Get("instance/service-accounts/default/scopes")
	if err != nil {
		return err
	}

	// Ensure it has the monitoring R/W scope.
	for _, scope := range strings.Fields(scopes) {
		if scope == "https://www.googleapis.com/auth/monitoring" {
			return nil
		}
	}

	return fmt.Errorf("current instance does not have the monitoring read-write scope")
}
