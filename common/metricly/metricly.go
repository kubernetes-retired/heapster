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
	"strconv"
)

type MetriclyConfig struct {
	ApiURL           string
	ApiKey           string
	ElementBatchSize int
}

func Config(uri *url.URL) (MetriclyConfig, error) {
	config := MetriclyConfig{
		ApiURL: "https://api.app.netuitive.com/ingest",
		ApiKey: "",
	}

	if len(uri.Host) > 0 {
		config.ApiURL = uri.Scheme + "://" + uri.Host + uri.Path
	}
	opts := uri.Query()
	if len(opts["apiKey"]) > 0 {
		config.ApiKey = opts["apiKey"][0]
	}
	if len(opts["elementBatchSize"]) > 0 {
		if ebs, err := strconv.Atoi(opts["elementBatchSize"][0]); ebs > 0 && err == nil {
			config.ElementBatchSize = ebs
		}
	}
	return config, nil
}
