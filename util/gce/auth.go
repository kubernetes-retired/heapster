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

// This package provides authentication utilities for Google Compute Engine (GCE)
package gce

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/GoogleCloudPlatform/gcloud-golang/compute/metadata"
)

type AuthToken struct {
	// The actual token.
	AccessToken string `json:"access_token"`

	// Number of seconds in which the token will expire
	ExpiresIn int `json:"expires_in"`

	// Type of token.
	TokenType string `json:"token_type"`
}

// Checks that the required auth scope is present.
func VerifyAuthScope(expectedScope string) error {
	scopes, err := metadata.Get("instance/service-accounts/default/scopes")
	if err != nil {
		return err
	}

	for _, scope := range strings.Fields(scopes) {
		if scope == expectedScope {
			return nil
		}
	}

	return fmt.Errorf("Current instance does not have the expected scope (%q). Actual scopes: %v", expectedScope, scopes)
}

// Get a token for performing GCE requests.
func GetAuthToken() (AuthToken, error) {
	rawToken, err := metadata.Get("instance/service-accounts/default/token")
	if err != nil {
		return AuthToken{}, err
	}

	var token AuthToken
	err = json.Unmarshal([]byte(rawToken), &token)
	if err != nil {
		return AuthToken{}, fmt.Errorf("failed to unmarshal service account token with output %q: %v", rawToken, err)
	}

	return token, err
}
