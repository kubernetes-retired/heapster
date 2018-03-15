// Copyright 2018 Google Inc. All Rights Reserved.
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
package metricly

import (
	"net/url"
	"testing"
)

func TestParsingMetriclyConfig(t *testing.T) {
	//given
	metriclyURI := "https://api.uat.netuitive.com/ingest?apiKey=datasourceApiKey&elementBatchSize=60&filter=label:{type:pod.*}&filter=!label:{type:pod_container}"
	uri, err := url.Parse(metriclyURI)
	if err != nil {
		t.Fatalf("Error when parsing Metricly URL: %s", err.Error())
	}
	//when
	config, err := Config(uri)
	if err != nil {
		t.Fatalf("Error when configuring Metricly URL: %s", err.Error())
	}
	//then
	if config.ApiURL != "https://api.uat.netuitive.com/ingest" {
		t.Fatalf("The Api URL is wrong, actual=%s, expected=%s", config.ApiURL, "https://api.uat.netuitive.com/ingest")
	}
	if config.ApiKey != "datasourceApiKey" {
		t.Fatalf("The Api Key is wrong, actual=%s, expected=%s", config.ApiKey, "datasourceApiKey")
	}
	if config.ElementBatchSize != 60 {
		t.Fatalf("The Element Batch Size is wrong, actual=%d, expected=%d", config.ElementBatchSize, 60)
	}
	if len(config.InclusionFilters) != 1 {
		t.Fatalf("There should be exactly 1 inclusion filter, actual=%d", len(config.InclusionFilters))
	}
	inFilter := config.InclusionFilters[0]
	if inFilter.Type != "label" || inFilter.Name != "type" || inFilter.Regex.String() != "pod.*" {
		t.Fatalf("The inclusion filter has the wrong value")
	}
	if len(config.ExclusionFilters) != 1 {
		t.Fatalf("There should be exactly 1 exclusion filter, actual=%d", len(config.ExclusionFilters))
	}
	exFilter := config.ExclusionFilters[0]
	if exFilter.Type != "label" || exFilter.Name != "type" || exFilter.Regex.String() != "pod_container" {
		t.Fatalf("The exclusion filter has the wrong value")
	}
}
