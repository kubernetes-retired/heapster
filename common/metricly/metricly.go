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
	"regexp"
	"strconv"
	"strings"

	"github.com/golang/glog"
)

var (
	FilterRegex = regexp.MustCompile("\\s*(.+)\\s*:\\s*{(.+):(.*)}")
)

//Filter defines the filter type and its values
//Type: type of the filter, only label is supported currently
//Name: name of the filter
//Regex: the regex that matches the name of the filter
type Filter struct {
	Type  string
	Name  string
	Regex *regexp.Regexp
}

type MetriclyConfig struct {
	ApiURL           string
	ApiKey           string
	ElementBatchSize int
	InclusionFilters []Filter
	ExclusionFilters []Filter
}

func Config(uri *url.URL) (MetriclyConfig, error) {
	config := MetriclyConfig{
		ApiURL: "https://api.app.metricly.com/ingest",
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
	if len(opts["filter"]) > 0 {
		for _, s := range opts["filter"] {
			if strings.HasPrefix(s, "!") {
				f, err := parseFilter(s[1:])
				if err != nil {
					glog.Warningf("Failed to parse filter %s, error: %s ", s, err.Error())
					continue
				}
				config.ExclusionFilters = append(config.ExclusionFilters, f)

			} else {
				f, err := parseFilter(s)
				if err != nil {
					glog.Warningf("Failed to parsing filter %s, error: %s ", s, err.Error())
					continue
				}
				config.InclusionFilters = append(config.InclusionFilters, f)

			}
		}
	}
	return config, nil
}

func parseFilter(s string) (Filter, error) {
	matches := FilterRegex.FindStringSubmatch(s)
	re, err := regexp.Compile(matches[3])
	if err != nil {
		return Filter{}, err
	}
	filter := Filter{
		Type:  matches[1],
		Name:  matches[2],
		Regex: re,
	}
	return filter, nil
}
