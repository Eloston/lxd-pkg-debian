// Copyright 2017 Canonical Ltd.
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

package rafttest_test

import (
	"testing"

	"github.com/CanonicalLtd/raft-test"
	"github.com/hashicorp/raft"
	"github.com/stretchr/testify/assert"
)

// At the beginning, all nodes are disconnected and each one
// starts as follower (and possibly enters the candidate state).
func TestCluster_Default(t *testing.T) {
	rafts, control := rafttest.Cluster(t, rafttest.FSMs(3))
	defer control.Close()

	assert.Len(t, rafts, 3)

	for _, r := range rafts {
		state := r.State()
		assert.True(t, state == raft.Follower || state == raft.Candidate)
	}
}
