package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestBrowserHostRelaysOnlyAuthenticatedFrames(t *testing.T) {
	path := filepath.Join(t.TempDir(), "endpoint.json")
	desktopInput, desktopWrite := io.Pipe()
	desktopRead, desktopOutput := io.Pipe()
	t.Cleanup(func() {
		_ = desktopWrite.Close()
		_ = desktopRead.Close()
	})
	done := make(chan error, 1)
	go func() { done <- serve(path, desktopInput, desktopOutput) }()
	event := func() envelope {
		t.Helper()
		body, err := readFrame(desktopRead)
		if err != nil {
			t.Fatal(err)
		}
		var message envelope
		if err := json.Unmarshal(body, &message); err != nil {
			t.Fatal(err)
		}
		return message
	}
	if ready := event(); ready.Type != "ready" || ready.Protocol != 4 {
		t.Fatal(ready)
	}
	address, err := readEndpoint(path)
	if err != nil {
		t.Fatal(err)
	}
	rejected, err := net.DialTimeout("tcp4", fmt.Sprintf("127.0.0.1:%d", address.Port), time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer rejected.Close()
	_ = rejected.SetDeadline(time.Now().Add(time.Second))
	if err := writeFrame(rejected, []byte("wrong token")); err != nil {
		t.Fatal(err)
	}
	if _, err := readFrame(rejected); err == nil {
		t.Fatal("unauthenticated connection unexpectedly received a frame")
	}

	browserInput, browserWrite := io.Pipe()
	browserRead, browserOutput := io.Pipe()
	t.Cleanup(func() {
		_ = browserWrite.Close()
		_ = browserRead.Close()
	})
	relayDone := make(chan error, 1)
	go func() { relayDone <- relay(path, browserInput, browserOutput) }()
	connection := event()
	if connection.Type != "connect" || len(connection.ConnectionID) != 32 {
		t.Fatal(connection)
	}
	if err := writeFrame(browserWrite, []byte(`{"type":"hello"}`)); err != nil {
		t.Fatal(err)
	}
	received := event()
	if received.Type != "message" || received.ConnectionID != connection.ConnectionID ||
		string(received.Message) != `{"type":"hello"}` {
		t.Fatal(received)
	}
	body, err := json.Marshal(envelope{
		Type: "send", ConnectionID: connection.ConnectionID, Message: json.RawMessage(`{"type":"welcome"}`),
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := writeFrame(desktopWrite, body); err != nil {
		t.Fatal(err)
	}
	frame, err := readFrame(browserRead)
	if err != nil || string(frame) != `{"type":"welcome"}` {
		t.Fatalf("unexpected browser frame: %s, %v", frame, err)
	}
	if err := browserWrite.Close(); err != nil {
		t.Fatal(err)
	}
	if disconnected := event(); disconnected.Type != "disconnect" || disconnected.ConnectionID != connection.ConnectionID {
		t.Fatal(disconnected)
	}
	if err := <-relayDone; err != nil {
		t.Fatal(err)
	}
	if err := desktopWrite.Close(); err != nil {
		t.Fatal(err)
	}
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("temporary browser host endpoint was not removed: %v", err)
	}
}
