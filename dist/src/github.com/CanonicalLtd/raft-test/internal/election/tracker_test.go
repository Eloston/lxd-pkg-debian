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

package election_test

import (
	"strconv"
	"testing"
	"time"

	"github.com/CanonicalLtd/raft-test/internal/election"
	"github.com/CanonicalLtd/raft-test/internal/logging"
	"github.com/hashicorp/raft"
	"github.com/stretchr/testify/assert"
)

// Observe that a given server has acquired leadership.
func TestTracker_Acquired(t *testing.T) {
	tracker := newTestTracker(t)
	defer tracker.Close()

	n := 3
	notifyChs := make([]chan bool, n)
	for i := 0; i < n; i++ {
		id := raft.ServerID(strconv.Itoa(i))
		notifyChs[i] = make(chan bool)
		tracker.Track(id, notifyChs[i])
	}

	future := tracker.Expect("0", 100*time.Millisecond)
	notifyChs[0] <- true
	_, err := future.Done()
	assert.NoError(t, err)
}

// If leadership is not acquired within the given timeout, an error is returned.
func TestTracker_AcquiredTimeout(t *testing.T) {
	tracker := newTestTracker(t)
	defer tracker.Close()

	tracker.Track("0", make(chan bool))

	future := tracker.Expect("0", time.Nanosecond)
	_, err := future.Done()
	assert.EqualError(t, err, "server 0: leadership not acquired within 1ns")
}

// While a server has acquired leadership, it's not possible to make another
// acquire request.
func TestTracker_AcquiredTwice(t *testing.T) {
	tracker := newTestTracker(t)
	defer tracker.Close()

	n := 3
	notifyChs := make([]chan bool, n)
	for i := 0; i < n; i++ {
		id := raft.ServerID(strconv.Itoa(i))
		notifyChs[i] = make(chan bool)
		tracker.Track(id, notifyChs[i])
	}

	future := tracker.Expect("0", 100*time.Millisecond)
	notifyChs[0] <- true
	_, err := future.Done()
	assert.NoError(t, err)

	f := func() {
		tracker.Expect("1", time.Millisecond)
	}
	assert.PanicsWithValue(t, "server 0 has already requested leadership", f)
}

// After a first leadership has completed, it's possible to observe a new one.
func TestTracker_AcquiredAfterLost(t *testing.T) {
	tracker := newTestTracker(t)
	defer tracker.Close()

	n := 3
	notifyChs := make([]chan bool, n)
	for i := 0; i < n; i++ {
		id := raft.ServerID(strconv.Itoa(i))
		notifyChs[i] = make(chan bool)
		tracker.Track(id, notifyChs[i])
	}

	future := tracker.Expect("0", 100*time.Millisecond)
	notifyChs[0] <- true
	leadership, err := future.Done()
	assert.NoError(t, err)
	notifyChs[0] <- false
	<-leadership.Lost()

	tracker.Expect("1", time.Nanosecond)
}

// It's not possible to add two trackers for the same server.
func TestTracker_AddSameServerID(t *testing.T) {
	tracker := newTestTracker(t)
	defer tracker.Close()

	tracker.Track("0", make(chan bool, 1))

	f := func() {
		tracker.Track("0", make(chan bool, 1))
	}
	assert.PanicsWithValue(t, "an observer for server 0 is already registered", f)
}

// After the tracker has been used to acquire leadership, it's not possible to
// add more server trackers.
func TestTracker_AddAfterObserving(t *testing.T) {
	tracker := newTestTracker(t)
	defer tracker.Close()

	tracker.Track("0", make(chan bool, 1))
	tracker.Expect("0", 100*time.Millisecond)

	f := func() {
		tracker.Track("1", make(chan bool, 1))
	}
	assert.PanicsWithValue(t, "can't track new server while observing", f)
}

func newTestTracker(t testing.TB) *election.Tracker {
	logger := logging.New(t, "DEBUG")
	return election.NewTracker(logger)
}
