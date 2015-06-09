// Copyright 2014 Google Inc. All Rights Reserved.
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

//go:generate go-extpoints
package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/GoogleCloudPlatform/heapster/manager"
	"github.com/GoogleCloudPlatform/heapster/sinks"
	"github.com/GoogleCloudPlatform/heapster/sources/api"
	"github.com/GoogleCloudPlatform/heapster/version"
	"github.com/GoogleCloudPlatform/kubernetes/pkg/util"
	"github.com/golang/glog"
)

var (
	argPollDuration    = flag.Duration("poll_duration", 10*time.Second, "The frequency at which heapster will poll for stats")
	argStatsResolution = flag.Duration("stats_resolution", 5*time.Second, "The resolution at which heapster will retain stats. Acceptable values are in the range [1 second, 'poll_duration')")
	argPort            = flag.Int("port", 8082, "port to listen to")
	argIp              = flag.String("listen_ip", "", "IP to listen on, defaults to all IPs")
	argMaxProcs        = flag.Int("max_procs", 0, "max number of CPUs that can be used simultaneously. Less than 1 for default (number of cores).")
	argCacheDuration   = flag.Duration("cache_duration", 10*time.Minute, "The total duration of the historical data that will be cached by heapster.")
	argSources         Uris
	argSinks           Uris
)

func main() {
	defer glog.Flush()
	flag.Var(&argSources, "source", "source(s) to watch")
	flag.Var(&argSinks, "sink", "external sink(s) that receive data")
	flag.Parse()
	setMaxProcs()
	glog.Infof(strings.Join(os.Args, " "))
	glog.Infof("Heapster version %v", version.HeapsterVersion)
	if err := validateFlags(); err != nil {
		glog.Fatal(err)
	}
	sources, sink, manager, err := doWork()
	if err != nil {
		glog.Error(err)
		os.Exit(1)
	}
	handler := setupHandlers(sources, sink, manager)
	addr := fmt.Sprintf("%s:%d", *argIp, *argPort)
	glog.Infof("Starting heapster on port %d", *argPort)
	glog.Fatal(http.ListenAndServe(addr, handler))
	os.Exit(0)
}

func validateFlags() error {
	if *argPollDuration <= time.Second {
		return fmt.Errorf("poll duration is invalid '%d'. Set it to a duration greater than a second", *argPollDuration)
	}
	if *argStatsResolution < time.Second {
		return fmt.Errorf("stats resolution needs to be greater than a second - %d", *argStatsResolution)
	}
	if *argStatsResolution >= *argPollDuration {
		return fmt.Errorf("stats resolution '%d' is not less than poll duration '%d'", *argStatsResolution, *argPollDuration)
	}

	return nil
}

func doWork() ([]api.Source, sinks.ExternalSinkManager, manager.Manager, error) {
	sources, err := newSources()
	if err != nil {
		return nil, nil, nil, err
	}
	externalSinks, err := newSinks()
	if err != nil {
		return nil, nil, nil, err
	}
	sinkManager, err := sinks.NewExternalSinkManager(externalSinks)
	if err != nil {
		return nil, nil, nil, err
	}
	manager, err := manager.NewManager(sources, sinkManager, *argStatsResolution, *argCacheDuration)
	if err != nil {
		return nil, nil, nil, err
	}
	go util.Until(manager.Housekeep, *argPollDuration, util.NeverStop)
	return sources, sinkManager, manager, nil
}

func setMaxProcs() {
	// Allow as many threads as we have cores unless the user specified a value.
	var numProcs int
	if *argMaxProcs < 1 {
		numProcs = runtime.NumCPU()
	} else {
		numProcs = *argMaxProcs
	}
	runtime.GOMAXPROCS(numProcs)

	// Check if the setting was successful.
	actualNumProcs := runtime.GOMAXPROCS(0)
	if actualNumProcs != numProcs {
		glog.Warningf("Specified max procs of %d but using %d", numProcs, actualNumProcs)
	}
}
