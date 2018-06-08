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

package network

import (
	"strconv"
	"testing"

	"github.com/CanonicalLtd/raft-test/internal/logging"
	"github.com/hashicorp/raft"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// By default an append entries RPC to target server fails.
func TestFaultyTransport_AppendEntries_Default(t *testing.T) {
	transports, cleanup := newTransports(t, 2)
	defer cleanup()

	transport0 := transports["0"]

	args, resp := newAppendEntries(0)
	err := transport0.AppendEntries("1", "1", args, resp)
	require.EqualError(t, err, "cannot reach server 1")
}

// The append entries RPC succeeds if the transport is connected to the target
// node.
func TestFaultyTransport_AppendEntries_Connected(t *testing.T) {
	transports, cleanup := newTransports(t, 2)
	defer cleanup()

	transport0 := transports["0"]
	transport0.Electing()

	args, resp := newAppendEntries(1, raft.LogNoop)
	err := transport0.AppendEntries("1", "1", args, resp)
	require.NoError(t, err)
	assert.True(t, transport0.HasAppendedLogsTo("1"))
}

// By a pipeline append entries RPC to target server fails if the peer is not
// connected.
func TestFaultyTransport_PipelineAppendEntries_Disconnected(t *testing.T) {
	transports, cleanup := newTransports(t, 2)
	defer cleanup()

	transport0 := transports["0"]
	transport0.Electing()

	pipeline0, err := transport0.AppendEntriesPipeline("1", "1")
	require.NoError(t, err)

	transport0.Deposing()

	args, resp := newAppendEntries(0)

	future, err := pipeline0.AppendEntries(args, resp)
	assert.Nil(t, future)
	assert.EqualError(t, err, "cannot reach server 1")
}

// The pipeline append entries RPC succeeds if the transport is connected to
// the target node.
func TestFaultyTransport_PipelineAppendEntries_Connected(t *testing.T) {
	transports, cleanup := newTransports(t, 2)
	defer cleanup()

	transport0 := transports["0"]
	transport0.Electing()

	pipeline0, err := transport0.AppendEntriesPipeline("1", "1")
	require.NoError(t, err)

	args, resp := newAppendEntries(0)

	future, err := pipeline0.AppendEntries(args, resp)
	require.NoError(t, err)
	require.NoError(t, future.Error())
	assert.Equal(t, uint64(0), future.Response().Term)
	assert.Equal(t, uint64(0), future.Response().LastLog)
}

// A server in leader state that is being deposed still flushes pending log
// entries to followers that are lagging behind.
func TestFaultyTransport_Deposing(t *testing.T) {
	n := 3
	transports, cleanup := newTransports(t, n)
	defer cleanup()

	transport := transports["0"]
	transport.Electing()

	followers := []raft.ServerID{"1", "2"}
	// Append a first noop log entry, as all newly elected leaders do.
	for _, id := range followers {
		args, resp := newAppendEntries(1, raft.LogNoop)
		err := transport.AppendEntries(id, raft.ServerAddress(id), args, resp)
		require.NoError(t, err)
	}

	// Append a command log, only to follower 1, this time using a
	// pipeline.
	pipeline1, err := transport.AppendEntriesPipeline("1", "1")
	require.NoError(t, err)

	args, resp := newAppendEntries(3, raft.LogCommand)
	future, err := pipeline1.AppendEntries(args, resp)
	require.NoError(t, err)
	require.NoError(t, future.Error())

	// Now disconnect the followers.
	transport.Deposing()

	// Append the same command log as before, but to follower 2, which has
	// been disconnected.
	pipeline2, err := transport.AppendEntriesPipeline("2", "2")
	require.NoError(t, err)
	future, err = pipeline2.AppendEntries(args, resp)
	require.NoError(t, err)
	require.NoError(t, future.Error())

	// Trying to append more entries fails.
	_, err = pipeline1.AppendEntries(args, resp)
	require.EqualError(t, err, "cannot reach server 1")

	_, err = pipeline2.AppendEntries(args, resp)

	err = transport.AppendEntries("1", "1", args, resp)
	require.EqualError(t, err, "cannot reach server 1")

	err = transport.AppendEntries("2", "2", args, resp)
	require.EqualError(t, err, "cannot reach server 2")
}

// Create n faulty transports that wrap n connected InmemTransport's.
//
// A fake consumer will be created for each transport, that just blindly
// replying to RPCs.
//
// The returned cleanup function stops all fake consumer goroutines.
func newTransports(t testing.TB, n int) (map[raft.ServerID]*eventTransport, func()) {
	// Create the in-memory transports, with addresses "0", "1", etc.
	inmemTransports := make([]*raft.InmemTransport, n)
	for i := 0; i < n; i++ {
		addr := raft.ServerAddress(strconv.Itoa(i))
		_, inmemTransports[i] = raft.NewInmemTransport(addr)
	}

	// Connect the in-memory transports to each other
	for i, t1 := range inmemTransports {
		for j, t2 := range inmemTransports {
			if i != j {
				t1.Connect(t2.LocalAddr(), t2)
				t2.Connect(t1.LocalAddr(), t1)
			}
		}
	}

	// Create the transport wrappers and their consumers.
	transports := make(map[raft.ServerID]*eventTransport)
	logger := logging.New(t, "DEBUG")
	shutdownCh := make(chan struct{})
	for i, inmemTransport := range inmemTransports {
		id := raft.ServerID(strconv.Itoa(i))
		transports[id] = newEventTransport(logger, id, inmemTransport)
		go fakeConsumer(transports[id], shutdownCh)
	}

	// Link the stores to the wrappers.
	for i, transport1 := range transports {
		for j, transport2 := range transports {
			if i == j {
				continue
			}
			transport1.AddPeer(transport2)
		}
	}

	cleanup := func() {
		close(shutdownCh)
	}

	return transports, cleanup
}

// Create a new pair of AppendEntriesRequest and AppendEntriesResponse,
// carrying logs of the given types with indexes starting at the given one.
func newAppendEntries(first uint64, types ...raft.LogType) (*raft.AppendEntriesRequest, *raft.AppendEntriesResponse) {
	entries := make([]*raft.Log, len(types))
	for i, t := range types {
		entries[i] = &raft.Log{
			Term:  1,
			Index: first + uint64(i),
			Type:  t,
		}
	}

	args := &raft.AppendEntriesRequest{Entries: entries}
	resp := &raft.AppendEntriesResponse{}

	return args, resp
}

func fakeConsumer(transport raft.Transport, shutdownCh chan struct{}) {
	for {
		select {
		case rpc := <-transport.Consumer():
			req := rpc.Command.(*raft.AppendEntriesRequest)
			var index uint64
			if n := len(req.Entries); n > 0 {
				index = req.Entries[n-1].Index
			}
			resp := &raft.AppendEntriesResponse{
				Term:    req.Term,
				LastLog: index,
				Success: true,
			}
			rpc.Respond(resp, nil)
		case <-shutdownCh:
			return
		}
	}
}
