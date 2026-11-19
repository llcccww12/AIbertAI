package main

import (
	"net/http"
	"strings"
	"testing"
)

func TestFileTransferRejectsWrongTargetAndTraversal(t *testing.T) {
	const id = "6bfc669b-55d4-40eb-afb3-6a9df8247430"
	const upload = "d38ff8ce-b949-422d-a080-488b4c89c20a"
	const server = "https://private.example.test"
	files := server + "/api/v1/endpoints/" + id + "/proxy/files/fg1.payload.signature/"
	in := fileTransferInput{Server: server, MachineID: id, Ticket: strings.Repeat("a", 32), Method: http.MethodGet}
	in.URL = files + "folder/report%20final.txt"
	if _, err := fileTransferURL(in); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{"../report", "%2E%2E/report", "test%2fescape", "%5creport", "report?redirect=1"} {
		in.URL = files + path
		if _, err := fileTransferURL(in); err == nil {
			t.Errorf("accepted invalid signed file path %q", path)
		}
	}
	in.Method, in.SessionID, in.Chunk = http.MethodPut, "session-fixture", []byte("fixture")
	in.URL = server + "/api/v1/endpoints/" + id + "/proxy/uploads/" + upload
	if _, err := fileTransferURL(in); err != nil {
		t.Fatal(err)
	}
	in.URL = strings.Replace(in.URL, "private.example.test", "outside.example.test", 1)
	if _, err := fileTransferURL(in); err == nil {
		t.Fatal("accepted upload to an unrelated origin")
	}
}

func TestFileTransferUsesConfiguredHTTPServer(t *testing.T) {
	const machine = "6bfc669b-55d4-40eb-afb3-6a9df8247430"
	const server = "http://ai-server.internal:5173"
	in := fileTransferInput{Server: server, MachineID: machine, Ticket: strings.Repeat("a", 32), Method: http.MethodGet}
	in.URL = server + "/api/v1/endpoints/" + machine + "/proxy/files/fg1.payload.signature/report.txt"
	if _, err := fileTransferURL(in); err != nil {
		t.Fatal(err)
	}
	in.URL = strings.Replace(in.URL, "ai-server.internal", "other.internal", 1)
	if _, err := fileTransferURL(in); err == nil {
		t.Fatal("accepted a file request for an unrelated HTTP origin")
	}
}
