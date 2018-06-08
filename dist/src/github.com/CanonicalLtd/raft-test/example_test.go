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
	"fmt"
	"log"
	"testing"
	"time"

	"github.com/CanonicalLtd/raft-test"
	"github.com/hashicorp/raft"
)

// A three-server raft cluster is created, the first server gets elected as
// leader, but when it tries to append the second FSM command log the two
// followers disconnect just before acknowledging the leader that they have
// appended the command.
func ExampleControl_Commands() {
	t := &testing.T{}

	// Create 3 dummy raft FSMs. This are just for the example, you should
	// use you own FSMs implementation.
	fsms := rafttest.FSMs(3)

	// Create a cluster of 3 raft servers, using the dummy FSMs.
	rafts, control := rafttest.Cluster(t, fsms)
	defer control.Close()

	// Elect the first server as leader, and set it up to lose leadership
	// when the second FSM command log is appended to the two follower
	// servers. Both servers will append the log, but they will disconnect
	// from the leader before they can report the successful append.
	control.Elect("0").When().Command(2).Appended().Depose()

	// The raft server with server ID "0" is now the leader.
	r := rafts["0"]

	// Apply the first command log, which succeeds.
	if err := r.Apply([]byte{}, time.Second).Error(); err != nil {
		log.Fatal("failed to apply first FSM command log", err)
	}

	// Apply the second command log, which fails.
	err := r.Apply([]byte{}, time.Second).Error()
	if err == nil {
		log.Fatal("applyig the second FSM command log did not fail")
	}
	if err != raft.ErrLeadershipLost {
		log.Fatal("wrong error when applying the second FSM command log", err)
	}

	// Elect the second server as leader and let it catch up with
	// logs. Since the second FSM command log has reached a quorum it will
	// be committed everywhere.
	control.Elect("1")
	control.Barrier()

	// Output:
	// number of commands applied by the first FSM: 2
	// number of commands applied by the second FSM: 2
	// number of commands applied by the third FSM: 2
	fmt.Println("number of commands applied by the first FSM:", control.Commands("0"))
	fmt.Println("number of commands applied by the second FSM:", control.Commands("1"))
	fmt.Println("number of commands applied by the third FSM:", control.Commands("2"))
}

// A three-server raft cluster is created, the first server gets elected as
// leader, after it commits the first FSM log command, one of the followers
// disconnects. The leader applies another four FSM log commands, taking a
// snapshot after the third is committed, and reconnecting the follower after
// the fourth is committed. After reconnection the follower will restore its
// FSM state from the leader's snapshot, since the TrailingLogs config is set
// to 1 by default.
func ExampleControl_Snapshots() {
	t := &testing.T{}

	// Create 3 dummy raft FSMs. This are just for the example, you should
	// use you own FSMs implementation.
	fsms := rafttest.FSMs(3)

	// Create a cluster of 3 raft servers, using the dummy FSMs.
	rafts, control := rafttest.Cluster(t, fsms)
	defer control.Close()

	// Elect the first server as leader
	term := control.Elect("0")

	// Set up the leader to take a snapshot after committing the fifth FSM
	// command log.
	term.When().Command(4).Committed().Snapshot()

	// The raft server with server ID "0" is now the leader.
	r := rafts["0"]

	// Apply four command logs, which all succeed. The fourth log is what
	// triggers the snapshot.
	for i := 0; i < 4; i++ {
		if err := r.Apply([]byte{}, time.Second).Error(); err != nil {
			log.Fatal("failed to apply first FSM command log", err)
		}
		if i == 0 {
			term.Disconnect("1")
		}
	}

	// Wait for the cluster to settle, in particular for the snapshot to
	// complete.
	control.Barrier()

	// Apply the fifth log, which will reconnect the disconnected follower.
	if err := r.Apply([]byte{}, time.Second).Error(); err != nil {
		log.Fatal("failed to apply first FSM command log", err)
	}
	term.Reconnect("1")

	// Wait for the cluster to settle, in particular for all FSMs to catch
	// up (the disconnected follower will restore from the snapshot).
	control.Barrier()

	// Output:
	// number of snapshots performed by the first server: 1
	// number of restores performed by the second server: 1
	// number of commands applied by the first FSM: 5
	// number of commands applied by the second FSM: 5
	// number of commands applied by the third FSM: 5
	fmt.Println("number of snapshots performed by the first server:", control.Snapshots("0"))
	fmt.Println("number of restores performed by the second server:", control.Restores("1"))
	fmt.Println("number of commands applied by the first FSM:", control.Commands("0"))
	fmt.Println("number of commands applied by the second FSM:", control.Commands("1"))
	fmt.Println("number of commands applied by the third FSM:", control.Commands("2"))
}
