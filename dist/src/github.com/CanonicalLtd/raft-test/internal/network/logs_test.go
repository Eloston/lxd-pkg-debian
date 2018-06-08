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

func TestStringifyLogs(t *testing.T) {
	cases := []struct {
		logs []*raft.Log
		text string
	}{
		{
			[]*raft.Log{},
			"0 entries",
		},
		{
			[]*raft.Log{
				{Type: raft.LogNoop, Term: 1, Index: 1},
			},
			"1 entry [Noop:term=1,index=1]",
		},
		{
			[]*raft.Log{
				{Type: raft.LogNoop, Term: 1, Index: 1},
				{Type: raft.LogCommand, Term: 1, Index: 2},
			},
			"2 entries [Noop:term=1,index=1 Command:term=1,index=2]",
		},
	}
	for _, c := range cases {
		t.Run(c.text, func(t *testing.T) {
			assert.Equal(t, c.text, stringifyLogs(c.logs))
		})
	}
}

func TestFilterLogsWithOlderTerms(t *testing.T) {
	cases := []struct {
		title string
		in    []*raft.Log
		out   []*raft.Log
	}{
		{
			"one log",
			[]*raft.Log{
				{Type: raft.LogNoop, Term: 1, Index: 1},
			},
			[]*raft.Log{
				{Type: raft.LogNoop, Term: 1, Index: 1},
			},
		},
		{
			"two logs, one has an older term",
			[]*raft.Log{
				{Type: raft.LogNoop, Term: 1, Index: 1},
				{Type: raft.LogNoop, Term: 2, Index: 2},
			},
			[]*raft.Log{
				{Type: raft.LogNoop, Term: 2, Index: 2},
			},
		},
	}
	for i, c := range cases {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			assert.Equal(t, c.out, filterLogsWithOlderTerms(c.in))
		})
	}
}
