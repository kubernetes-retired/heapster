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

package nodes

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"
)

const tempHostsFile = "/temp_file"

func writeMarshaledData(f *os.File, v interface{}) error {
	data, err := json.Marshal(&v)
	if err != nil {
		return err
	}
	_, err = f.WriteAt(data, 0)
	return err
}

func externalizeNodes(nodeList *NodeList) *ExternalNodeList {
	externalNodeList := ExternalNodeList{}
	for node := range nodeList.Items {
		externalNodeList.Items = append(externalNodeList.Items, node)
	}
	return &externalNodeList
}

func TestExternalFile(t *testing.T) {
	f, err := ioutil.TempFile("", "")
	if err != nil {
		t.Error(err)
	}
	defer os.Remove(f.Name())
	nodesApi := externalCadvisorNodes{f.Name()}

	testData := &NodeList{
		Items: map[Node]Empty{
			Node{"host1", "1.2.3.4"}: {},
			Node{"host2", "1.2.3.4"}: {},
		},
	}
	require.NoError(t, writeMarshaledData(f, externalizeNodes(testData)))
	res, err := nodesApi.List()
	require.NoError(t, err)
	require.True(t, reflect.DeepEqual(res, testData), "failure. Expected: %+v, got: %+v", res, testData)

	testData.Items[Node{"host3", "2.2.2.2"}] = Empty{}
	require.NoError(t, writeMarshaledData(f, externalizeNodes(testData)))
	res, err = nodesApi.List()
	require.NoError(t, err)
	require.True(t, reflect.DeepEqual(res, testData), "failure. Expected: %+v, got: %+v", res, testData)
}
