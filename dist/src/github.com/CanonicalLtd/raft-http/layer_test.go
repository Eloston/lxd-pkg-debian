package rafthttp_test

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/CanonicalLtd/raft-http"
	"github.com/hashicorp/raft"
)

// The Accept method receives connections from the conns channel.
func TestLayer_Accept(t *testing.T) {
	handler, layer := newHandlerAndLayer(t)
	server, client := net.Pipe()
	go func() {
		w := newResponseHijacker()
		w.HijackConn = server
		r := &http.Request{Method: "GET", Header: make(http.Header)}
		r.Header.Set("Upgrade", "raft")
		handler.ServeHTTP(w, r)
	}()
	reader := bufio.NewReader(client)
	reader.ReadString('\n')

	got, err := layer.Accept()
	if err != nil {
		t.Fatal(err)
	}
	if got != server {
		t.Fatal("Accept() didn't receive the connection through the conns channel")
	}
}

// The Accept method returns an error if the layer was closed.
func TestLayer_AcceptWhenClosed(t *testing.T) {
	layer := newLayer(t)
	layer.Close()
	conn, err := layer.Accept()
	if conn != nil {
		t.Fatal("Expected nil connection")
	}
	if err != io.EOF {
		t.Fatal("Expected EOF error")
	}
}

// The Close method is a no-op.
func TestLayer_Close(t *testing.T) {
	layer := newLayer(t)
	if err := layer.Close(); err != nil {
		t.Fatal(err)
	}
}

// The Addr method return the local address.
func TestLayer_Addr(t *testing.T) {
	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	addr := listener.Addr()
	layer := newLayerWithAddr(t, addr)
	if layer.Addr() != addr {
		t.Fatal("Addr() did not return the configured address")
	}
}

// If the given HTTP path is invalid, an error is returned.
func TestLayer_DialPathParseError(t *testing.T) {
	layer := newLayerWithPath(t, "%:not\a/path")
	want := "invalid URL path %:not\a/path"
	defer func() {
		if got := recover(); got != want {
			t.Fatalf("Invalid path did not panic as expected: %s (should be %s)", got, want)
		}
	}()
	layer.Dial(":0", time.Second)
}

// If the Dial function returns an error when trying to connect, it
// gets propagated.
func TestLayer_DialDialError(t *testing.T) {
	dial := func(string, time.Duration) (net.Conn, error) {
		return nil, fmt.Errorf("connection error")
	}
	layer := newLayerWithDial(t, dial)
	conn, err := layer.Dial(":0", time.Second)
	if conn != nil {
		t.Fatal("expected nil conn")
	}
	if err == nil {
		t.Fatal("expected error")
	}
	const want = "dialing failed: connection error"
	got := err.Error()
	if want != got {
		t.Fatalf("expected error\n%q\ngot\n%q", want, got)
	}

}

// If an error occurs while writing the HTTP request, it is returned.
func TestLayer_DialRequestWriteError(t *testing.T) {
	dial := func(string, time.Duration) (net.Conn, error) {
		conn, _ := net.Pipe()
		conn.Close() // This will trigger a write error
		return conn, nil
	}
	layer := newLayerWithDial(t, dial)
	conn, err := layer.Dial(":0", time.Second)
	if conn != nil {
		t.Fatal("expected nil conn")
	}
	if err == nil {
		t.Fatal("expected error")
	}
	const want = "sending HTTP request failed: io: read/write on closed pipe"
	got := err.Error()
	if want != got {
		t.Fatalf("expected error error\n%q\ngot\n%q", want, got)
	}

}

// If the response sent by the server has not code 101, an error is
// returned.
func TestLayer_DialResponseWrongStatusCode(t *testing.T) {
	server, client := net.Pipe()
	dial := func(string, time.Duration) (net.Conn, error) {
		return client, nil
	}
	layer := newLayerWithDial(t, dial)
	go func() {
		// Read a line from the request data written by the
		// layer via the client conn. This unblocks reading
		// the response
		buffer := bufio.NewReader(server)
		_, _, err := buffer.ReadLine()
		if err != nil {
			t.Fatalf("Failed to read client request: %v", err)
		}
		// Send a malformed response to the client.
		server.Write([]byte("HTTP/1.1 200 OK\r\n\r\n"))
	}()
	conn, err := layer.Dial(":0", time.Second)
	if conn != nil {
		t.Fatal("Dial() returned a non-nil connection")
	}
	if err == nil {
		t.Fatal("Dial() didn't return any error")
	}
	if err.Error() != "dialing fail: expected status code 101 got 200" {
		t.Fatalf("Dial() returned unexpected error: %v", err)
	}
}

// If an error occurs while reading the HTTP request, it is returned.
func TestLayer_DialResponseReadError(t *testing.T) {
	server, client := net.Pipe()
	dial := func(string, time.Duration) (net.Conn, error) {
		return client, nil
	}
	layer := newLayerWithDial(t, dial)
	go func() {
		// Read a line from the request data written by the
		// layer via the client conn. This unblocks reading
		// the response
		buffer := bufio.NewReader(server)
		_, _, err := buffer.ReadLine()
		if err != nil {
			t.Fatalf("Failed to read client request: %v", err)
		}
		// Send a malformed response to the client.
		server.Write([]byte("garbage\n"))
	}()
	conn, err := layer.Dial("127.0.0.1:0", time.Second)
	if conn != nil {
		t.Fatal("expected nil conn")
	}
	if err == nil {
		t.Fatal("expected error")
	}
	const want = `failed to read response: malformed HTTP response "garbage"`
	got := err.Error()
	if want != got {
		t.Fatalf("expected error\n%q\ngot\n%q", want, got)
	}
}

// If the response sent by the server has not an Upgrade header set to
// "raft", an error is returned.
func TestLayer_DialResponseNoUpgradeHeader(t *testing.T) {
	server, client := net.Pipe()
	dial := func(string, time.Duration) (net.Conn, error) {
		return client, nil
	}
	layer := newLayerWithDial(t, dial)
	go func() {
		// Read a line from the request data written by the
		// layer via the client conn. This unblocks reading
		// the response
		buffer := bufio.NewReader(server)
		_, _, err := buffer.ReadLine()
		if err != nil {
			t.Fatalf("Failed to read client request: %v", err)
		}
		// Send a malformed response to the client.
		server.Write([]byte("HTTP/1.1 101 Switching Protocols\r\n\r\n"))
	}()
	conn, err := layer.Dial(":0", time.Second)
	if conn != nil {
		t.Fatal("Dial() returned a non-nil connection")
	}
	if err == nil {
		t.Fatal("Dial() didn't return any error")
	}
	if err.Error() != "missing or unexpected Upgrade header in response" {
		t.Fatalf("Dial() returned unexpected error: %v", err)
	}
}

// If the HTTP request to join a cluster returns an unexpected status
// code, an error is returned.
func TestLayer_JoinErrorStatusCode(t *testing.T) {
	// Setup a handler that always returns 404.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	}))
	defer server.Close()

	addr := server.Listener.Addr()
	layer := newLayerWithAddrAndDial(t, addr, rafthttp.NewDialTCP())
	err := layer.Join("1", raftAddress(addr), time.Second)
	if err == nil {
		t.Fatal("Join call did not fail")
	}
	if err.Error() != fmt.Sprintf("server join failed: http code 404 '404 page not found'") {
		t.Fatalf("Got unexpected error: %v", err)
	}
}

// If there's a network failure while leaving, an error is returned.
func TestLayer_LeaveNetworkError(t *testing.T) {
	addr := &net.TCPAddr{IP: []byte{0, 0, 0, 0}, Port: 0}
	layer := newLayerWithAddrAndDial(t, addr, rafthttp.NewDialTCP())
	err := layer.Leave("1", raftAddress(addr), time.Second)
	if err == nil {
		t.Fatal("Leave call did not fail")
	}
}

// If there's a network failure while joining, an error is returned.
func TestLayer_JoinNetworkError(t *testing.T) {
	addr := &net.TCPAddr{IP: []byte{0, 0, 0, 0}, Port: 0}
	layer := newLayerWithAddrAndDial(t, addr, rafthttp.NewDialTCP())
	err := layer.Join("1", raftAddress(addr), time.Second)
	if err == nil {
		t.Fatal("Join call did not fail")
	}
}

// If the URL in the Location header of a redirect is not valid, an
// error is returned.
func TestLayer_JoinLocationParseError(t *testing.T) {
	// Setup a handler that returns an invalid Location.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, ":/%:not\a/url", http.StatusPermanentRedirect)
	}))
	defer server.Close()

	addr := server.Listener.Addr()
	layer := newLayerWithAddrAndDial(t, addr, rafthttp.NewDialTCP())
	if err := layer.Join("1", raftAddress(addr), time.Second); err == nil {
		t.Fatal("Join call did not fail")
	}
}

func TestLayer_JoinRetryIfServiceUnavailable(t *testing.T) {
	// Setup a handler that fails only the first time.
	retried := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !retried {
			http.Error(w, "leader unknown", http.StatusServiceUnavailable)
			retried = true
		}
	}))
	defer server.Close()

	addr := server.Listener.Addr()
	layer := newLayerWithAddrAndDial(t, addr, rafthttp.NewDialTCP())
	if err := layer.Join("1", raftAddress(addr), time.Second); err != nil {
		t.Fatalf("Join request failed although it was supposed to retry: %v", err)
	}
}

// Create both a new Handler and a new Layer, with sane defaults.
func newHandlerAndLayer(t *testing.T) (*rafthttp.Handler, *rafthttp.Layer) {
	handler := newHandler(t)
	layer := rafthttp.NewLayerWithLogger("/", newAddr(), handler, nil, newLogger(t))
	return handler, layer
}

// Create a new Layer, with sane defaults.
func newLayer(t *testing.T) *rafthttp.Layer {
	return rafthttp.NewLayerWithLogger("/", newAddr(), newHandler(t), nil, newLogger(t))
}

// Create a new Layer, with the given address.
func newLayerWithAddr(t *testing.T, addr net.Addr) *rafthttp.Layer {
	return rafthttp.NewLayerWithLogger("/", addr, newHandler(t), nil, newLogger(t))
}

// Create a new Layer, with the given path.
func newLayerWithPath(t *testing.T, path string) *rafthttp.Layer {
	return rafthttp.NewLayerWithLogger(path, newAddr(), newHandler(t), nil, newLogger(t))
}

// Create a new Layer, with the given dial function.
func newLayerWithDial(t *testing.T, dial rafthttp.Dial) *rafthttp.Layer {
	return rafthttp.NewLayerWithLogger("/", newAddr(), newHandler(t), dial, newLogger(t))
}

// Create a new Layer, with the given address and dial function.
func newLayerWithAddrAndDial(t *testing.T, addr net.Addr, dial rafthttp.Dial) *rafthttp.Layer {
	return rafthttp.NewLayerWithLogger("/", addr, newHandler(t), dial, newLogger(t))
}

// Create a new test network address.
func newAddr() net.Addr {
	return &net.TCPAddr{
		IP:   net.IPv4(127, 0, 0, 1),
		Port: 1234,
	}
}

// Covert a net.Addr to raft.ServerAddress
func raftAddress(addr net.Addr) raft.ServerAddress {
	return raft.ServerAddress(addr.String())
}
