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
	"strconv"

	etcd "github.com/coreos/etcd/client"
	"github.com/coreos/go-semver/semver"
)

// LatestDaemonVersion attempts to retrieve the latest version of fleetd
// that has been registered in the Registry. It returns the version if
// it can be determined (or nil otherwise), and any error encountered.
func (r *EtcdRegistry) LatestDaemonVersion() (*semver.Version, error) {
	machs, err := r.Machines()
	if err != nil {
		if isEtcdError(err, etcd.ErrorCodeKeyNotFound) {
			err = nil
		}
		return nil, err
	}
	var lv *semver.Version
	for _, m := range machs {
		v, err := semver.NewVersion(m.Version)
		if err != nil {
			continue
		} else if lv == nil || lv.LessThan(*v) {
			lv = v
		}
	}
	return lv, nil
}

// EngineVersion implements the ClusterRegistry interface
func (r *EtcdRegistry) EngineVersion() (int, error) {
	res, err := r.kAPI.Get(r.ctx(), r.engineVersionPath(), nil)
	if err != nil {
		// no big deal, either the cluster is new or is just
		// upgrading from old unversioned code
		if isEtcdError(err, etcd.ErrorCodeKeyNotFound) {
			err = nil
		}
		return 0, err
	}

	return strconv.Atoi(res.Node.Value)
}

// UpdateEngineVersion implements the ClusterRegistry interface
func (r *EtcdRegistry) UpdateEngineVersion(from, to int) error {
	key := r.engineVersionPath()

	strTo := strconv.Itoa(to)
	strFrom := strconv.Itoa(from)

	opts := &etcd.SetOptions{
		PrevValue: strFrom,
	}
	_, err := r.kAPI.Set(r.ctx(), key, strTo, opts)
	if err == nil {
		return nil
	} else if !isEtcdError(err, etcd.ErrorCodeKeyNotFound) {
		return err
	}

	opts = &etcd.SetOptions{
		PrevExist: etcd.PrevNoExist,
	}
	_, err = r.kAPI.Set(r.ctx(), key, strTo, opts)
	return err
}

func (r *EtcdRegistry) engineVersionPath() string {
	return r.prefixed("/engine/version")
}
