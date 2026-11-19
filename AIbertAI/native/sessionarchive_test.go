package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestReadLegacySessionArchivePreservesCommittedReplay(t *testing.T) {
	const id = "06bfc669-55d4-40eb-afb3-6a9df8247430"
	const message = `{"version":11,"from_seq":2,"to_seq":4,"message":{"id":"user-1","role":"user","parts":[{"type":"file","name":"sample.txt","uri":"fixture"}]},"toolCalls":{"tool-1":{"output":"result"}},"unknown":{"kept":true}}`
	profile := t.TempDir()
	session := filepath.Join(profile, "runtime", "sessions", id)
	if err := os.MkdirAll(session, 0700); err != nil {
		t.Fatal(err)
	}
	meta := legacySessionMetadata{id, legacySessionSchema, legacyReplayVersion, int64(len(message) + 1), 1, 4, "fixture"}
	metadata, err := json.Marshal(meta)
	if err != nil {
		t.Fatal(err)
	}
	write := func(name string, data []byte) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(session, name), data, 0600); err != nil {
			t.Fatal(err)
		}
	}
	write("meta.json", metadata)
	write("replay.jsonl", []byte(message+"\n{uncommitted}"))
	archive, err := readLegacySessionArchive(profile, id)
	if err != nil {
		t.Fatal(err)
	}
	if string(archive.Session) != string(metadata) || len(archive.Replay) != 1 || string(archive.Replay[0]) != message {
		t.Fatal("archive changed legacy metadata or replay content")
	}
	write("replay.jsonl", []byte(message))
	if _, err := readLegacySessionArchive(profile, id); err == nil {
		t.Fatal("accepted a truncated replay")
	}
	if _, err := readLegacySessionArchive(profile, "../"+id); err == nil {
		t.Fatal("accepted a session ID that escapes the storage directory")
	}
}
