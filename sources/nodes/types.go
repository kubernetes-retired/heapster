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

type Node struct {
	// Hostname
	Name string `json:"name,omitempty"`
	// HostIP
	IP string `json:"ip,omitempty"`
}

type Empty struct{}

// NodeList contains the nodes that an instance of heapster is required to
// monitor.
type NodeList struct {
	Items map[Node]Empty
}

type ExternalNodeList struct {
	Items []Node `json:"items,omitempty"`
}

type NodesApi interface {
	// Returns a list of nodes that needs to be monitores or error on failure.
	List() (*NodeList, error)

	// Returns a string that contains internal debug information.
	DebugInfo() string
}
