package main

import (
	"bytes"
	"encoding/json"
	"testing"
)

func TestDSHChatToolImagesStayInTheirOwnTurn(t *testing.T) {
	const prior = `{"role":"user","content":[{"type":"text","text":"Attached image(s) from tool result:"},{"type":"image_url","image_url":{"url":"stale-preview"}}]}`
	const current = `{"role":"user","content":[{"type":"text","text":"Attached image(s) from tool result:"},{"type":"image_url","image_url":{"url":"current-preview"}}]}`
	const attachment = `{"role":"user","content":[{"type":"text","text":"Inspect my attachment"},{"type":"image_url","image_url":{"url":"user-attachment"}}]}`
	const tool = `{"role":"tool","tool_call_id":"call1","content":"Read preview.png"}`
	const text = `{"role":"user","content":"Continue in this session"}`
	first := []byte(`{"messages":[` + attachment + "," + tool + "," + prior + "]}")
	if got := projectDSHChatToolImages(first); !bytes.Equal(got, first) {
		t.Fatalf("removed an image within its own turn: %s", got)
	}
	followup := []byte(`{"model":"demo","messages":[` + attachment + "," + tool + "," + prior + "," + text + "," + current + "],\"stream\":true}")
	got := projectDSHChatToolImages(followup)
	if !json.Valid(got) || bytes.Contains(got, []byte("stale-preview")) {
		t.Fatalf("replayed the old preview or produced invalid JSON: %s", got)
	}
	for _, want := range []string{"user-attachment", "current-preview", "Continue in this session", "Read preview.png", `"stream":true`} {
		if !bytes.Contains(got, []byte(want)) {
			t.Errorf("lost %q when projecting the follow-up", want)
		}
	}
}
