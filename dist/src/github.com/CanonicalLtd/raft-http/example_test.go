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

package rafthttp_test

import (
	"bytes"
	"fmt"
	"log"
	"net"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/CanonicalLtd/raft-http"
	"github.com/CanonicalLtd/raft-membership"
	"github.com/CanonicalLtd/raft-test"
	"github.com/hashicorp/raft"
)

// Connect threed raft nodes using HTTP network layers.
func Example() {
	t := &testing.T{}

	// Create a set of transports using HTTP layers.
	handlers := make([]*rafthttp.Handler, 3)
	layers := make([]*rafthttp.Layer, 3)
	transports := make([]*raft.NetworkTransport, 3)
	out := bytes.NewBuffer(nil)
	layersCleanup := make([]func(), 3)
	for i := range layers {
		handler := rafthttp.NewHandler()
		layer, cleanup := newExampleLayer(handler)

		transport := raft.NewNetworkTransport(layer, 2, time.Second, out)

		layers[i] = layer
		layersCleanup[i] = cleanup
		handlers[i] = handler
		transports[i] = transport
	}

	// Create a raft.Transport factory that uses the above layers.
	transport := rafttest.Transport(func(i int) raft.Transport { return transports[i] })
	servers := rafttest.Servers(0)

	// Create a 3-node cluster with default test configuration.
	rafts, control := rafttest.Cluster(t, rafttest.FSMs(3), transport, servers)
	defer func() {
		control.Close()
		for _, transport := range transports {
			transport.CloseStreams()
		}
		for _, cleanup := range layersCleanup {
			defer cleanup()
		}
	}()

	// Start handling membership change requests on all nodes.
	for i, handler := range handlers {
		id := raft.ServerID(strconv.Itoa(i))
		go raftmembership.HandleChangeRequests(rafts[id], handler.Requests())
	}

	// Node 0 is the one supposed to get leadership, since it's currently
	// the only one in the cluster.
	control.Elect("0")

	// Request that the second node joins the cluster.
	if err := layers[1].Join("1", transports[0].LocalAddr(), time.Second); err != nil {
		log.Fatalf("joining server 1 failed: %v", err)
	}

	// Request that the third node joins the cluster, contacting
	// the non-leader node 1. The request will be automatically
	// redirected to node 0.
	if err := layers[2].Join("2", transports[1].LocalAddr(), time.Second); err != nil {
		log.Fatal(err)
	}

	// Rquest that the third node leaves the cluster.
	if err := layers[2].Leave("2", transports[2].LocalAddr(), time.Second); err != nil {
		log.Fatal(err)
	}

	// Output:
	// has logged connections: true
	// peers: 1
	fmt.Println("has logged connections:", strings.Contains(out.String(), "accepted connection from"))
	fmt.Println("peers:", rafts["0"].Stats()["num_peers"])
}

// Create a new Layer using a new Handler attached to a running HTTP
// server.
func newExampleLayer(handler *rafthttp.Handler) (*rafthttp.Layer, func()) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		log.Fatalf("listening to local port failed: %v", err)
	}
	layer := rafthttp.NewLayer("/", listener.Addr(), handler, rafthttp.NewDialTCP())
	server := &http.Server{Handler: handler}
	go server.Serve(listener)

	cleanup := func() {
		if err := server.Close(); err != nil {
			log.Fatalf("closing server failed: %v", err)
		}
	}

	return layer, cleanup
}
