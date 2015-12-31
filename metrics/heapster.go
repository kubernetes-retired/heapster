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

//go:generate ./hooks/run_extpoints.sh

package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/golang/glog"
	"k8s.io/heapster/metrics/core"
	"k8s.io/heapster/metrics/manager"
	"k8s.io/heapster/metrics/processors"
	"k8s.io/heapster/metrics/sinks"
	"k8s.io/heapster/metrics/sources"
	"k8s.io/heapster/version"
)

var (
	argMetricResolution = flag.Duration("metric_resolution", 30*time.Second, "The resolution at which heapster will retain metrics.")
	argPort             = flag.Int("port", 8082, "port to listen to")
	argIp               = flag.String("listen_ip", "", "IP to listen on, defaults to all IPs")
	argMaxProcs         = flag.Int("max_procs", 0, "max number of CPUs that can be used simultaneously. Less than 1 for default (number of cores)")
	argTLSCertFile      = flag.String("tls_cert", "", "file containing TLS certificate")
	argTLSKeyFile       = flag.String("tls_key", "", "file containing TLS key")
	argTLSClientCAFile  = flag.String("tls_client_ca", "", "file containing TLS client CA for client cert validation")
	argAllowedUsers     = flag.String("allowed_users", "", "comma-separated list of allowed users")
	argSources          manager.Uris
	argSinks            manager.Uris
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

	// sources
	if len(argSources) != 1 {
		glog.Fatal("wrong number of sources specified")
	}
	sourceFactory := sources.NewSourceFactory()
	sourceProvider, err := sourceFactory.BuildAll(argSources)
	if err != nil {
		glog.Fatal(err)
	}
	sourceManager, err := sources.NewSourceManager(sourceProvider, sources.DefaultMetricsScrapeTimeout)
	if err != nil {
		glog.Fatal(err)
	}

	// sinks
	sinksFactory := sinks.NewSinkFactory()
	metricSink, sinkList := sinksFactory.BuildAll(argSinks)
	if metricSink == nil {
		glog.Fatalf("Failed to create metric sink")
	}
	for _, sink := range sinkList {
		glog.Infof("Starting with %s sink", sink.Name())
	}
	sinkManager, err := sinks.NewDataSinkManager(sinkList, sinks.DefaultSinkExportDataTimeout, sinks.DefaultSinkStopTimeout)
	if err != nil {
		glog.Fatal(err)
	}

	// main manager
	manager, err := manager.NewManager(sourceManager, []core.DataProcessor{&processors.PodAggregator{}}, sinkManager, *argMetricResolution,
		manager.DefaultScrapeOffset, manager.DefaultMaxParallelism)
	if err != nil {
		glog.Fatal(err)
	}
	manager.Start()

	handler := setupHandlers(metricSink)
	addr := fmt.Sprintf("%s:%d", *argIp, *argPort)
	glog.Infof("Starting heapster on port %d", *argPort)

	mux := http.NewServeMux()
	if len(*argTLSCertFile) > 0 && len(*argTLSKeyFile) > 0 {
		promeHandler := core.PrometheusHandler
		if len(*argTLSClientCAFile) > 0 {
			authPprofHandler, err := newAuthHandler(handler)
			if err != nil {
				glog.Fatal(err)
			}
			handler = authPprofHandler

			authPromeHandler, err := newAuthHandler(core.PrometheusHandler)
			if err != nil {
				glog.Fatal(err)
			}
			promeHandler = authPromeHandler
		}
		mux.Handle("/", handler)
		mux.Handle(core.PrometheusPath, promeHandler)
		glog.Fatal(http.ListenAndServeTLS(addr, *argTLSCertFile, *argTLSKeyFile, mux))
	} else {
		mux.Handle("/", handler)
		mux.Handle(core.PrometheusPath, core.PrometheusHandler)
		glog.Fatal(http.ListenAndServe(addr, mux))
	}

}

func validateFlags() error {
	if *argMetricResolution < 5*time.Second {
		return fmt.Errorf("metric resolution needs to be greater than 5 seconds - %d", *argMetricResolution)
	}
	if (len(*argTLSCertFile) > 0 && len(*argTLSKeyFile) == 0) || (len(*argTLSCertFile) == 0 && len(*argTLSKeyFile) > 0) {
		return fmt.Errorf("both TLS certificate & key are required to enable TLS serving")
	}
	if len(*argTLSClientCAFile) > 0 && len(*argTLSCertFile) == 0 {
		return fmt.Errorf("client cert authentication requires TLS certificate & key")
	}
	return nil
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
