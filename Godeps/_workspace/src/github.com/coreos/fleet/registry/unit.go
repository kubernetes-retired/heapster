// Copyright 2014 CoreOS, Inc.
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

package registry

import (
	"strings"

	etcd "github.com/coreos/etcd/client"

	"github.com/coreos/fleet/log"
	"github.com/coreos/fleet/unit"
)

const (
	unitPrefix = "/unit/"
)

func (r *EtcdRegistry) storeOrGetUnitFile(u unit.UnitFile) (err error) {
	um := unitModel{
		Raw: u.String(),
	}

	val, err := marshal(um)
	if err != nil {
		return err
	}

	key := r.hashedUnitPath(u.Hash())
	opts := &etcd.SetOptions{
		PrevExist: etcd.PrevNoExist,
	}
	_, err = r.kAPI.Set(r.ctx(), key, val, opts)
	// unit is already stored
	if isEtcdError(err, etcd.ErrorCodeNodeExist) {
		// TODO(jonboulle): verify more here?
		err = nil
	}
	return
}

// getUnitByHash retrieves from the Registry the Unit associated with the given Hash
func (r *EtcdRegistry) getUnitByHash(hash unit.Hash) *unit.UnitFile {
	key := r.hashedUnitPath(hash)
	opts := &etcd.GetOptions{
		Recursive: true,
	}
	resp, err := r.kAPI.Get(r.ctx(), key, opts)
	if err != nil {
		if isEtcdError(err, etcd.ErrorCodeKeyNotFound) {
			err = nil
		}
		return nil
	}
	return r.unitFromEtcdNode(hash, resp.Node)
}

// getAllUnitsHashMap retrieves from the Registry all Units and returns a map of hash to UnitFile
func (r *EtcdRegistry) getAllUnitsHashMap() (map[string]*unit.UnitFile, error) {
	key := r.prefixed(unitPrefix)
	opts := &etcd.GetOptions{
		Recursive: true,
		Quorum:    true,
	}
	hashToUnit := map[string]*unit.UnitFile{}
	resp, err := r.kAPI.Get(r.ctx(), key, opts)
	if err != nil {
		return nil, err
	}

	for _, node := range resp.Node.Nodes {
		parts := strings.Split(node.Key, "/")
		if len(parts) == 0 {
			log.Errorf("key '%v' doesn't have enough parts", node.Key)
			continue
		}
		stringHash := parts[len(parts)-1]
		hash, err := unit.HashFromHexString(stringHash)
		if err != nil {
			log.Errorf("failed to get Hash for key '%v' with stringHash '%v': %v", node.Key, stringHash, err)
			continue
		}
		unit := r.unitFromEtcdNode(hash, node)
		if unit == nil {
			continue
		}
		hashToUnit[stringHash] = unit
	}

	return hashToUnit, nil
}

func (r *EtcdRegistry) unitFromEtcdNode(hash unit.Hash, etcdNode *etcd.Node) *unit.UnitFile {
	var um unitModel
	if err := unmarshal(etcdNode.Value, &um); err != nil {
		log.Errorf("error unmarshaling Unit(%s): %v", hash, err)
		return nil
	}

	u, err := unit.NewUnitFile(um.Raw)
	if err != nil {
		log.Errorf("error parsing Unit(%s): %v", hash, err)
		return nil
	}

	return u
}

func (r *EtcdRegistry) hashedUnitPath(hash unit.Hash) string {
	return r.prefixed(unitPrefix, hash.String())
}

type unitModel struct {
	Raw string
}
