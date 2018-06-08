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

	"github.com/hashicorp/raft"
	"github.com/stretchr/testify/assert"
)

// Only logs for the current term are kept.
func TestPeer_UpdateLogs(t *testing.T) {
	cases := []struct {
		title    string
		initial  []*raft.Log // Initial logs in peer.logs
		appended []*raft.Log // Logs passed to peer.UpdateLogs()
		final    []*raft.Log // Logs in the peer.logs after the update
	}{
		{
			"no initial logs, no appended logs",
			[]*raft.Log{},
			[]*raft.Log{},
			[]*raft.Log{},
		},
		{
			"some initial logs, no appended logs",
			[]*raft.Log{
				{Type: raft.LogNoop, Term: 1, Index: 1},
			},
			[]*raft.Log{},
			[]*raft.Log{
				{Type: raft.LogNoop, Term: 1, Index: 1},
			},
		},
		{
			"no initial logs, one appended log",
			[]*raft.Log{},
			[]*raft.Log{
				{Type: raft.LogNoop, Term: 1, Index: 1},
			},
			[]*raft.Log{
				{Type: raft.LogNoop, Term: 1, Index: 1},
			},
		},
		{
			"no initial logs, two appended logs with different terms",
			[]*raft.Log{},
			[]*raft.Log{
				{Type: raft.LogNoop, Term: 1, Index: 1},
				{Type: raft.LogNoop, Term: 2, Index: 2},
			},
			[]*raft.Log{
				{Type: raft.LogNoop, Term: 2, Index: 2},
			},
		},
		{
			"one initial log with older term, one appended log with newer term",
			[]*raft.Log{
				{Type: raft.LogNoop, Term: 1, Index: 1},
			},
			[]*raft.Log{
				{Type: raft.LogNoop, Term: 2, Index: 2},
			},
			[]*raft.Log{
				{Type: raft.LogNoop, Term: 2, Index: 2},
			},
		},
		{
			"one initial log, two appended logs with one duplicate",
			[]*raft.Log{
				{Type: raft.LogNoop, Term: 1, Index: 1},
			},
			[]*raft.Log{
				{Type: raft.LogNoop, Term: 1, Index: 1},
				{Type: raft.LogCommand, Term: 1, Index: 2},
			},
			[]*raft.Log{
				{Type: raft.LogNoop, Term: 1, Index: 1},
				{Type: raft.LogCommand, Term: 1, Index: 2},
			},
		},
	}
	for i, c := range cases {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			peer := newPeer("0", "1")
			peer.logs = c.initial
			peer.UpdateLogs(c.appended)
			assert.Equal(t, c.final, peer.logs)
		})
	}

}

// Only command logs are counted.
func TestPeer_CommandLogsCount(t *testing.T) {
	peer := newPeer("0", "1")
	peer.UpdateLogs([]*raft.Log{
		{Type: raft.LogNoop, Term: 1, Index: 1},
		{Type: raft.LogCommand, Term: 1, Index: 2},
	})
	assert.Equal(t, uint64(1), peer.CommandLogsCount())
}
