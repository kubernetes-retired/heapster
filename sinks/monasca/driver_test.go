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

package monasca

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/heapster/extpoints"
)

// test the transformation of timeseries to monasca metrics
func TestTimeseriesTransform(t *testing.T) {
	// setup
	sut := monascaSink{}

	// do
	metrics := sut.processMetrics(testInput)

	// assert
	assert.Equal(t, expectedTransformed, metrics)
}

// test if the sink creation fails when password is not provided
func TestMissingPasswordError(t *testing.T) {
	// setup
	uri, _ := url.Parse("monasca:?keystone-url=" + testConfig.IdentityEndpoint + "&user_id=" + testUserID)

	// do
	_, err := CreateMonascaSink(uri, extpoints.HeapsterConf{})

	// assert
	assert.Error(t, err)
}

// test if the sink creation fails when keystone-url is not provided
func TestMissingKeystoneURLError(t *testing.T) {
	// setup
	uri, _ := url.Parse("monasca:?user_id=" + testUserID + "&password=" + testPassword)

	// do
	_, err := CreateMonascaSink(uri, extpoints.HeapsterConf{})

	// assert
	assert.Error(t, err)
}

// test if the sink creation fails when neither user-id nor username are provided
func TestMissingUserError(t *testing.T) {
	// setup
	uri, _ := url.Parse("monasca:?keystone-url=" + testConfig.IdentityEndpoint + "&password=" + testPassword)

	// do
	_, err := CreateMonascaSink(uri, extpoints.HeapsterConf{})

	// assert
	assert.Error(t, err)
}

// test if the sink creation fails when domain_id and domainname are missing
// and username is provided
func TestMissingDomainWhenUsernameError(t *testing.T) {
	// setup
	uri, _ := url.Parse("monasca:?keystone-url=" + testConfig.IdentityEndpoint + "&password=" +
		testPassword + "&username=" + testUsername)

	// do
	_, err := CreateMonascaSink(uri, extpoints.HeapsterConf{})

	// assert
	assert.Error(t, err)
}

// test if the sink creation fails when password is not provided
func TestWrongMonascaURLError(t *testing.T) {
	// setup
	uri, _ := url.Parse("monasca:?keystone-url=" + testConfig.IdentityEndpoint + "&password=" +
		testConfig.Password + "&user-id=" + testConfig.UserID + "&monasca-url=_malformed")

	// do
	_, err := CreateMonascaSink(uri, extpoints.HeapsterConf{})

	// assert
	assert.Error(t, err)
}

// test the successful creation of the monasca
func TestMonascaSinkCreation(t *testing.T) {
	// setup
	uri, _ := url.Parse("monasca:?keystone-url=" + testConfig.IdentityEndpoint + "&password=" +
		testConfig.Password + "&user-id=" + testConfig.UserID)

	// do
	_, err := CreateMonascaSink(uri, extpoints.HeapsterConf{})

	// assert
	assert.NoError(t, err)
}

// integration test of storing metrics
func TestStoreMetrics(t *testing.T) {
	// setup
	uri, _ := url.Parse("monasca:?keystone-url=" + testConfig.IdentityEndpoint + "&password=" +
		testConfig.Password + "&user-id=" + testConfig.UserID)
	sinkList, _ := CreateMonascaSink(uri, extpoints.HeapsterConf{})
	sut := sinkList[0]

	// do
	err := sut.StoreTimeseries(testInput)

	// assert
	assert.NoError(t, err)
}

// integration test of failure to create metrics
func TestStoreMetricsFailure(t *testing.T) {
	// setup
	ks, _ := NewKeystoneClient(testConfig)
	monURL, _ := url.Parse("http://unexisting.monasca.com")
	sut := monascaSink{client: &ClientImpl{ksClient: ks, monascaURL: monURL}}

	// do
	err := sut.StoreTimeseries(testInput)

	// assert
	assert.Error(t, err)
}
