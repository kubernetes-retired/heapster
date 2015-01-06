package integration

import (
	"flag"
	"strings"
	"testing"
	
	"github.com/stretchr/testify/assert"
)

var kubeVersions = flag.String("kube_versions", "0.7.2", "Comma separated list of kube versions to test against")

func TestHeapsterInfluxDBWorks(t * testing.T) {
	kubeVersionsList := strings.Split(*kubeVersions, ",")
	for _, kubeVersion := range kubeVersionsList {
		fm, err := newKubeFramework(t, kubeVersion)
		assert.NoError(t, err)
		assert.NoError(t, fm.Cleanup())
	}
}
