package main

import (
	"flag"
	"os"

	"github.com/vishh/caggregator/sources"
	"github.com/golang/glog"
)

func main() {
	glog.Info("cAggregator is running")	
	flag.Parse()
	cadvisorSource, err := sources.NewCadvisorSource()
	if err != nil {
		glog.Error(err)
		os.Exit(1)
	}
	err = cadvisorSource.FetchData()
	if err != nil {
		glog.Error(err)
		os.Exit(1)
	}
	os.Exit(0)
}
