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

package network_test

import (
	"strconv"
	"testing"

	"github.com/CanonicalLtd/raft-test/internal/logging"
	"github.com/CanonicalLtd/raft-test/internal/network"
	"github.com/hashicorp/raft"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNetwork_FaultyEnqueue(t *testing.T) {
	transports := newTransports(2)
	network := network.New(logging.New(t, "DEBUG"))
	for i, transport := range transports {
		network.Add(itoID(i), transport)
	}

	network.Electing("0")

	transport := network.Transport("0")

	// The follower consume RPCs.
	go func() {
		resp := &raft.AppendEntriesResponse{
			Term:    1,
			LastLog: 0,
			Success: true,
		}
		rpc := <-network.Transport("1").Consumer()
		rpc.Respond(resp, nil)
	}()

	// Append a first noop log entry, as all newly elected leaders do.
	args := &raft.AppendEntriesRequest{
		Entries: []*raft.Log{{Index: 1, Term: 1, Type: raft.LogNoop}},
	}
	resp := &raft.AppendEntriesResponse{}
	err := transport.AppendEntries("1", "1", args, resp)
	require.NoError(t, err)
	require.True(t, network.HasAppendedLogsFromTo("0", "1"))

	event := network.ScheduleEnqueueFailure("0", 1)

	// Asynchronously handle the event by disconnecting the transport.
	go func() {
		<-event.Watch()
		network.Deposing("0")
		event.Ack()
	}()

	// Append further entries using a pipeline, as a leader would do after
	// the first no-op command.
	pipeline, err := transport.AppendEntriesPipeline("1", "1")
	require.NoError(t, err)

	args = &raft.AppendEntriesRequest{
		Entries: []*raft.Log{{Type: raft.LogCommand}},
	}
	resp = &raft.AppendEntriesResponse{}
	_, err = pipeline.AppendEntries(args, resp)
	assert.EqualError(t, err, "cannot reach server 1")
}

// Create n connected InmemTransport's.
func newTransports(n int) []*raft.InmemTransport {
	// Create the in-memory transports, with addresses "0", "1", etc.
	transports := make([]*raft.InmemTransport, n)
	for i := 0; i < n; i++ {
		_, transports[i] = raft.NewInmemTransport(itoAddr(i))
	}

	// Connect the in-memory transports to each other
	for i, t1 := range transports {
		for j, t2 := range transports {
			if i != j {
				t1.Connect(t2.LocalAddr(), t2)
				t2.Connect(t1.LocalAddr(), t1)
			}
		}
	}

	return transports
}

// Convert an integer to a server ID.
func itoID(i int) raft.ServerID {
	return raft.ServerID(strconv.Itoa(i))
}

// Convert an integer to a server address.
func itoAddr(i int) raft.ServerAddress {
	return raft.ServerAddress(strconv.Itoa(i))
}
