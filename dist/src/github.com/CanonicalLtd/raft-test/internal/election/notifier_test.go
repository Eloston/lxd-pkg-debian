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

package election

import (
	"testing"
	"time"

	"github.com/CanonicalLtd/raft-test/internal/logging"
	"github.com/hashicorp/raft"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Leadership is acquired when the notify channel receives a true value.
func TestNotifier_Acquired(t *testing.T) {
	notifier, notifyCh := newTestNotifier(t)
	defer notifier.Close()

	future := notifier.Acquired(100 * time.Millisecond)
	notifyCh <- true
	leadership, err := future.Done()
	assert.NotNil(t, leadership)
	assert.NoError(t, err)
}

// If leadership is not acquired within the given timeout, an error is
// returned.
func TestNotifier_AcquiredTimeout(t *testing.T) {
	notifier, _ := newTestNotifier(t)
	defer notifier.Close()

	future := notifier.Acquired(time.Nanosecond)
	leadership, err := future.Done()
	assert.Nil(t, leadership)
	assert.EqualError(t, err, "server 0: leadership not acquired within 1ns")
}

// Leadership is lost when a false value is received from the notify channel.
func TestNotifier_LeaderhsipLost(t *testing.T) {
	notifier, notifyCh := newTestNotifier(t)
	defer notifier.Close()

	future := notifier.Acquired(100 * time.Millisecond)
	notifyCh <- true
	leadership, err := future.Done()
	assert.NotNil(t, leadership)
	assert.NoError(t, err)

	notifyCh <- false
	select {
	case <-leadership.Lost():
	case <-time.After(100 * time.Millisecond):
		t.Fatal("no leadership lost notification received")
	}
}

// The Lost() method can be called multiple times.
func TestNotifier_LostTwice(t *testing.T) {
	notifier, notifyCh := newTestNotifier(t)
	defer notifier.Close()

	future := notifier.Acquired(100 * time.Millisecond)
	notifyCh <- true
	leadership, err := future.Done()
	assert.NotNil(t, leadership)
	assert.NoError(t, err)

	leadership.Lost()
	leadership.Lost()
}

// If a leadership change is received when no leadership request has been
// submitted yet, the notifier panics.
func TestNotifier_UnexpectedLeadershipChange(t *testing.T) {
	logger := logging.New(t, "DEBUG")
	id := raft.ServerID("0")
	notifyCh := make(chan bool, 1)

	notifier := &notifier{
		logger:   logger,
		id:       id,
		notifyCh: notifyCh,
	}

	go func() {
		notifyCh <- true
		notifyCh <- true
	}()
	assert.PanicsWithValue(t, "server 0: unexpected leadership change", notifier.start)
}

// If the notifier receives the same bool value twice from the notify channel,
// it panics.
func TestNotifier_InconsistentNotifications(t *testing.T) {
	logger := logging.New(t, "DEBUG")
	id := raft.ServerID("0")
	notifyCh := make(chan bool)

	notifier := &notifier{
		logger:   logger,
		id:       id,
		notifyCh: notifyCh,
		futureCh: make(chan *Future, 1),
	}

	notifier.futureCh <- newFuture(id, 100*time.Millisecond)
	go func() {
		time.Sleep(10 * time.Millisecond)
		notifyCh <- true
		notifyCh <- true
	}()
	assert.PanicsWithValue(t, "server 0 acquired leadership twice in a row", notifier.start)
}

// If a leadership request is submitted when another one is not done, the
// notifier panics.
func TestNotifier_DoubleAcquiredRequest(t *testing.T) {
	logger := logging.New(t, "DEBUG")
	id := raft.ServerID("0")
	notifyCh := make(chan bool, 1)

	notifier := &notifier{
		logger:   logger,
		id:       id,
		notifyCh: notifyCh,
		futureCh: make(chan *Future, 2),
	}

	notifier.futureCh <- newFuture(id, time.Nanosecond)
	notifier.futureCh <- newFuture(id, time.Nanosecond)
	assert.PanicsWithValue(t, "server 0: duplicate leadership request", notifier.start)
}

// A leadership object can't be notified more than once of leadership acquired
// or lost.
func TestNotifier_DoubleNotification(t *testing.T) {
	logger := logging.New(t, "DEBUG")
	id := raft.ServerID("0")
	notifyCh := make(chan bool, 1)

	notifier := &notifier{
		logger:   logger,
		id:       id,
		notifyCh: notifyCh,
		futureCh: make(chan *Future, 1),
	}

	future := newFuture(id, 100*time.Millisecond)

	// Pretend that this future instance already received a acquired
	// notification.
	close(future.acquiredCh)

	go func() {
		time.Sleep(10 * time.Millisecond)
		notifyCh <- true
	}()

	notifier.futureCh <- future
	assert.PanicsWithValue(t, "server 0: duplicate leadership acquired notification", notifier.start)
}

// Once a leadership is acquired and then lost, it's possible to submit another one.
func TestNotifier_OneLeadershipAfterTheOther(t *testing.T) {
	notifier, notifyCh := newTestNotifier(t)
	defer notifier.Close()

	go func() {
		time.Sleep(10 * time.Millisecond)
		notifyCh <- true
		notifyCh <- false
	}()

	future := notifier.Acquired(100 * time.Millisecond)
	notifyCh <- true
	leadership, err := future.Done()
	require.NoError(t, err)

	notifyCh <- false
	select {
	case <-leadership.Lost():
	case <-time.After(100 * time.Millisecond):
		t.Fatal("leadership was not lost")
	}

	future = notifier.Acquired(100 * time.Millisecond)
	notifyCh <- true
	_, err = future.Done()
	require.NoError(t, err)
}

func newTestNotifier(t testing.TB) (*notifier, chan bool) {
	logger := logging.New(t, "DEBUG")
	id := raft.ServerID("0")
	notifyCh := make(chan bool)

	notifier := newNotifier(logger, id, notifyCh)
	return notifier, notifyCh
}
