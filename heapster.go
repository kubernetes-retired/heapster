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

package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/GoogleCloudPlatform/heapster/sinks"
	"github.com/GoogleCloudPlatform/heapster/sources"
	"github.com/GoogleCloudPlatform/heapster/validate"
	"github.com/GoogleCloudPlatform/heapster/version"
	"github.com/golang/glog"
)

var (
	argPollDuration = flag.Duration("poll_duration", 10*time.Second, "Polling duration")
	argPort         = flag.Int("port", 8082, "port to listen")
	argIp           = flag.String("listen_ip", "", "IP to listen on, defaults to all IPs")
	argMaxProcs     = flag.Int("max_procs", 0, "max number of CPUs that can be used simultaneously. Less than 1 for default (number of cores).")
)

func main() {
	defer glog.Flush()
	flag.Parse()
	setMaxProcs()
	glog.Infof(strings.Join(os.Args, " "))
	glog.Infof("Heapster version %v", version.HeapsterVersion)
	glog.Infof("Flags: %s", strings.Join(getFlags(), " "))
	if err := validateFlags(); err != nil {
		glog.Fatal(err)
	}
	source, sink, err := doWork()
	if err != nil {
		glog.Error(err)
		os.Exit(1)
	}
	setupHandlers(source, sink)
	addr := fmt.Sprintf("%s:%d", *argIp, *argPort)
	glog.Infof("Starting heapster on port %d", *argPort)
	glog.Fatal(http.ListenAndServe(addr, nil))
	os.Exit(0)
}

func getFlags() []string {
	flagData := []string{}
	flag.VisitAll(func(flag *flag.Flag) {
		flagData = append(flagData, fmt.Sprintf("%s='%v'", flag.Name, flag.Value))
	})
	return flagData
}

func validateFlags() error {
	if *argPollDuration <= time.Second {
		return fmt.Errorf("poll duration is invalid '%d'. Set it to a duration greater than a second", *argPollDuration)
	}
	return nil
}
func setupHandlers(source sources.Source, sink sinks.ExternalSinkManager) {
	// Validation/Debug handler.
	http.HandleFunc(validate.ValidatePage, func(w http.ResponseWriter, r *http.Request) {
		err := validate.HandleRequest(w, source, sink)
		if err != nil {
			fmt.Fprintf(w, "%s", err)
		}
	})

	// TODO(jnagal): Add a main status page.
	http.Handle("/", http.RedirectHandler(validate.ValidatePage, http.StatusTemporaryRedirect))
}

func doWork() (sources.Source, sinks.ExternalSinkManager, error) {
	source, err := sources.NewSource(*argPollDuration)
	if err != nil {
		return nil, nil, err
	}
	sink, err := sinks.NewSink()
	if err != nil {
		return nil, nil, err
	}
	go housekeep(source, sink)
	return source, sink, nil
}

func housekeep(source sources.Source, sink sinks.ExternalSinkManager) {
	ticker := time.NewTicker(*argPollDuration)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			data, err := source.GetInfo()
			if err != nil {
				glog.Errorf("failed to get information from source - %v", err)
			}
			if err := sink.Store(data); err != nil {
				glog.Errorf("failed to push information to sink - %v", err)
			}
		}
	}

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
