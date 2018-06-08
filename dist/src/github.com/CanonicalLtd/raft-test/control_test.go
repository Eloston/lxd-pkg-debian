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
	"strconv"
	"testing"
	"time"

	"github.com/CanonicalLtd/raft-test"
	"github.com/hashicorp/raft"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Elect and depose a leader.
func TestControl_ElectAndDepose(t *testing.T) {
	rafts, control := rafttest.Cluster(t, rafttest.FSMs(3), rafttest.DiscardLogger())
	defer control.Close()

	control.Elect("0")

	r := rafts["0"]
	assert.Equal(t, raft.Leader, r.State())

	control.Depose()

	assert.NotEqual(t, raft.Leader, r.State())
}

// Depose a previously elected leader after a certain command log gets
// enqueued.
func TestControl_DeposeAfterCommandEnqueued(t *testing.T) {
	rafts, control := rafttest.Cluster(t, rafttest.FSMs(3)) // rafttest.DiscardLogger())
	defer control.Close()

	control.Elect("0").When().Command(2).Enqueued().Depose()

	r := rafts["0"]

	err := r.Apply([]byte{}, time.Second).Error()
	require.NoError(t, err)
	err = r.Apply([]byte{}, time.Second).Error()
	assert.EqualError(t, err, raft.ErrLeadershipLost.Error())
	assert.Equal(t, uint64(1), control.Commands("0"))
}

// Depose a previously elected leader after a certain command log gets
// enqueued and then elect another one.
func TestControl_DeposeAfterCommandEnqueuedThenElect(t *testing.T) {
	n := 3
	for i := 1; i < 1; i++ {
		id := raft.ServerID(strconv.Itoa(i)) // ID of the next leader
		t.Run(string(id), func(t *testing.T) {
			rafts, control := rafttest.Cluster(t, rafttest.FSMs(n), rafttest.DiscardLogger())
			defer control.Close()

			control.Elect("0").When().Command(2).Enqueued().Depose()

			r := rafts["0"]
			err := r.Apply([]byte{}, time.Second).Error()
			require.NoError(t, err)
			err = r.Apply([]byte{}, time.Second).Error()
			assert.EqualError(t, err, raft.ErrLeadershipLost.Error())

			control.Elect(id)
			r = rafts[id]
			err = r.Apply([]byte{}, time.Second).Error()
			require.NoError(t, err)
			assert.Equal(t, uint64(2), control.Commands(id))
		})
	}
}

// Depose a previously elected leader after a certain command log gets enqueued
// and then elect the same one. The recovered leader sends the command logs
// that have failed.
func TestControl_DeposeAfterCommandEnqueuedThenElectSame(t *testing.T) {
	rafts, control := rafttest.Cluster(t, rafttest.FSMs(3), rafttest.DiscardLogger())
	defer control.Close()

	control.Elect("0").When().Command(2).Enqueued().Depose()

	r := rafts["0"]
	err := r.Apply([]byte{}, time.Second).Error()
	require.NoError(t, err)
	err = r.Apply([]byte{}, time.Second).Error()
	assert.EqualError(t, err, raft.ErrLeadershipLost.Error())

	control.Elect("0")
	r = rafts["0"]
	err = r.Apply([]byte{}, time.Second).Error()
	require.NoError(t, err)
	assert.Equal(t, uint64(3), control.Commands("0"))
}

// Depose a previously elected leader after a certain command log gets
// appended by all followers.
func TestControl_DeposeAfterCommandAppended(t *testing.T) {
	rafts, control := rafttest.Cluster(t, rafttest.FSMs(3), rafttest.DiscardLogger())
	defer control.Close()

	control.Elect("0").When().Command(1).Appended().Depose()

	r := rafts["0"]
	err := r.Apply([]byte{}, time.Second).Error()
	assert.EqualError(t, err, raft.ErrLeadershipLost.Error())
	assert.Equal(t, uint64(0), control.Commands("0"))
}

// Depose a previously elected leader after a certain command log gets
// appended and then elect another one.
func TestControl_DeposeAfterCommandAppendedThenElect(t *testing.T) {
	n := 3
	for i := 0; i < n; i++ {
		id := raft.ServerID(strconv.Itoa(i)) // ID of the next leader
		t.Run(string(id), func(t *testing.T) {
			rafts, control := rafttest.Cluster(t, rafttest.FSMs(n))
			defer control.Close()

			control.Elect("0").When().Command(2).Appended().Depose()

			r := rafts["0"]
			err := r.Apply([]byte{}, time.Second).Error()
			require.NoError(t, err)
			err = r.Apply([]byte{}, time.Second).Error()
			assert.EqualError(t, err, raft.ErrLeadershipLost.Error())

			control.Elect(id)
			r = rafts[id]
			err = r.Apply([]byte{}, time.Second).Error()
			require.NoError(t, err)
			assert.Equal(t, uint64(3), control.Commands(id))
		})
	}
}

// Depose a previously elected leader after a certain command log gets
// committed.
func TestControl_DeposeAfterCommandCommitted(t *testing.T) {
	rafts, control := rafttest.Cluster(t, rafttest.FSMs(3)) // rafttest.DiscardLogger())
	defer control.Close()

	control.Elect("0").When().Command(1).Committed().Depose()

	r := rafts["0"]
	err := r.Apply([]byte{}, time.Second).Error()
	require.NoError(t, err)
	err = r.Apply([]byte{}, time.Second).Error()
	assert.EqualError(t, err, raft.ErrNotLeader.Error())
	assert.Equal(t, uint64(1), control.Commands("0"))
}

// Depose a previously elected leader after a certain command log gets
// committed and then elect another one.
func TestControl_DeposeAfterCommandCommittedThenElect(t *testing.T) {
	n := 3
	for i := 0; i < n; i++ {
		id := raft.ServerID(strconv.Itoa(i)) // ID of the next leader
		t.Run(string(id), func(t *testing.T) {
			rafts, control := rafttest.Cluster(t, rafttest.FSMs(n)) // rafttest.DiscardLogger())
			defer control.Close()

			control.Elect("0").When().Command(1).Committed().Depose()

			r := rafts["0"]
			err := r.Apply([]byte{}, time.Second).Error()
			require.NoError(t, err)
			err = r.Apply([]byte{}, time.Second).Error()
			assert.EqualError(t, err, raft.ErrNotLeader.Error())

			control.Elect(id)
			r = rafts[id]
			err = r.Apply([]byte{}, time.Second).Error()
			require.NoError(t, err)
			assert.Equal(t, uint64(2), control.Commands(id))
		})
	}
}

// Take a snapshot on the leader when a certain command log gets committed.
func TestControl_SnapshotAfterCommandCommitted(t *testing.T) {
	rafts, control := rafttest.Cluster(t, rafttest.FSMs(3)) // rafttest.DiscardLogger())
	defer control.Close()

	control.Elect("0").When().Command(2).Committed().Snapshot()

	r := rafts["0"]
	err := r.Apply([]byte{}, time.Second).Error()
	require.NoError(t, err)
	err = r.Apply([]byte{}, time.Second).Error()
	require.NoError(t, err)

	control.Barrier()

	assert.Equal(t, uint64(1), control.Snapshots("0"))
}

// Make a follower restore from a snapshot after a disconnection.
func TestControl_RestoreAfterDisconnection(t *testing.T) {
	rafts, control := rafttest.Cluster(t, rafttest.FSMs(3)) // rafttest.DiscardLogger())
	defer control.Close()

	term := control.Elect("0")
	term.When().Command(4).Committed().Snapshot()

	r := rafts["0"]
	for i := 0; i < 6; i++ {
		err := r.Apply([]byte{}, time.Second).Error()
		require.NoError(t, err)
		if i == 0 {
			term.Disconnect("1")
		}
		if i == 4 {
			term.Reconnect("1")
		}
	}

	control.Barrier()

	assert.Equal(t, uint64(1), control.Snapshots("0"))
	assert.Equal(t, uint64(1), control.Restores("1"))

	assert.Equal(t, uint64(6), control.Commands("0"))
	assert.Equal(t, uint64(6), control.Commands("1"))
	assert.Equal(t, uint64(6), control.Commands("2"))
}
