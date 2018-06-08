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

package raftmembership

import (
	"testing"

	"github.com/CanonicalLtd/raft-test"
)

func TestHandleChangeRequests_InvalidChangeRequestKind(t *testing.T) {
	raft, cleanup := rafttest.Server(t, rafttest.FSM())
	defer cleanup()

	request := &ChangeRequest{
		kind: -1,
	}
	requests := make(chan *ChangeRequest, 1)
	requests <- request
	close(requests)

	const want = "invalid change request kind: -1"
	defer func() {
		got := recover()
		if got != want {
			t.Errorf("expected panic\n%q\ngot\n%q", want, got)
		}
	}()
	HandleChangeRequests(raft, requests)
}
