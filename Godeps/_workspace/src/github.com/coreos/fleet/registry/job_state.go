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
	"time"

	etcd "github.com/coreos/etcd/client"

	"github.com/coreos/fleet/job"
	"github.com/coreos/fleet/unit"
)

// determineJobState decides what the State field of a Job object should
// be, based on three parameters:
//  - heartbeat should be the machine ID that is known to have recently
//    heartbeaten (see UnitHeartbeat) the Unit.
//  - tgt should be the machine ID to which the Job is currently scheduled
//  - us should be the most recent UnitState
func determineJobState(heartbeat, tgt string, us *unit.UnitState) (state job.JobState) {
	state = job.JobStateInactive

	if tgt == "" || us == nil {
		return
	}

	state = job.JobStateLoaded

	if heartbeat != tgt {
		return
	}

	state = job.JobStateLaunched
	return
}

func (r *EtcdRegistry) UnitHeartbeat(name, machID string, ttl time.Duration) error {
	key := r.jobHeartbeatPath(name)
	opts := &etcd.SetOptions{
		TTL: ttl,
	}
	_, err := r.kAPI.Set(r.ctx(), key, machID, opts)
	return err
}

func (r *EtcdRegistry) ClearUnitHeartbeat(name string) {
	key := r.jobHeartbeatPath(name)
	r.kAPI.Delete(r.ctx(), key, nil)
}

func (r *EtcdRegistry) jobHeartbeatPath(jobName string) string {
	return r.prefixed(jobPrefix, jobName, "job-state")
}
