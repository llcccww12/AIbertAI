package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
)

const (
	legacySessionSchema = 8
	legacyReplayVersion = 11
	maxSessionMetadata  = 4 << 20
	maxReplayRecord     = 16 << 20
	maxReplayExport     = 64 << 20
)

var errInvalidSessionArchive = errors.New("invalid_session_archive")

type legacySessionMetadata struct {
	ID             string `json:"id"`
	SchemaVersion  int    `json:"schema_version"`
	ReplayVersion  int    `json:"replay_version"`
	ReplayBytes    int64  `json:"replay_bytes"`
	MessageCount   int    `json:"message_count"`
	ReplayApplied  int64  `json:"replay_applied_seq"`
	ReplayIdentity string `json:"replay_generation"`
}

type legacySessionArchive struct {
	Session json.RawMessage   `json:"session"`
	Replay  []json.RawMessage `json:"replay"`
}

func readLegacySessionMetadata(root *os.Root) ([]byte, error) {
	info, err := root.Stat("meta.json")
	if err != nil || !info.Mode().IsRegular() || info.Size() > maxSessionMetadata {
		return nil, errInvalidSessionArchive
	}
	file, err := root.Open("meta.json")
	if err != nil {
		return nil, errInvalidSessionArchive
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, maxSessionMetadata+1))
	if err != nil || len(data) > maxSessionMetadata {
		return nil, errInvalidSessionArchive
	}
	return data, nil
}

func readLegacySessionArchive(profile, id string) (legacySessionArchive, error) {
	var archive legacySessionArchive
	if !filepath.IsAbs(profile) || !validUUID(id) {
		return archive, errInvalidSessionArchive
	}
	root, err := os.OpenRoot(profile)
	if err != nil {
		return archive, errInvalidSessionArchive
	}
	defer root.Close()
	session, err := root.OpenRoot(filepath.Join("runtime", "sessions", id))
	if err != nil {
		return archive, errInvalidSessionArchive
	}
	defer session.Close()
	data, err := readLegacySessionMetadata(session)
	if err != nil {
		return archive, err
	}
	var meta legacySessionMetadata
	if json.Unmarshal(data, &meta) != nil || meta.ID != id || meta.ReplayBytes < 0 || meta.MessageCount < 0 || meta.ReplayApplied < 0 {
		return archive, errInvalidSessionArchive
	}
	if meta.SchemaVersion != legacySessionSchema || meta.ReplayVersion != legacyReplayVersion {
		return archive, errors.New("unsupported_session_archive_version")
	}
	if meta.ReplayIdentity == "" || meta.ReplayBytes > maxReplayExport {
		return archive, errors.New("session_archive_unavailable")
	}
	replay, err := readLegacySessionReplay(session, meta)
	if err != nil {
		return archive, err
	}
	// The legacy runtime may append or rebuild the replay while it is being read.
	// The metadata is its commit boundary. Return only a coherent view of it.
	after, err := readLegacySessionMetadata(session)
	if err != nil || !bytes.Equal(data, after) {
		return archive, errors.New("session_archive_changed")
	}
	return legacySessionArchive{Session: data, Replay: replay}, nil
}

func readLegacySessionReplay(session *os.Root, meta legacySessionMetadata) ([]json.RawMessage, error) {
	records := []json.RawMessage{}
	if meta.ReplayBytes == 0 && meta.MessageCount == 0 {
		return records, nil
	}
	info, err := session.Stat("replay.jsonl")
	if err != nil || !info.Mode().IsRegular() || info.Size() < meta.ReplayBytes {
		return nil, errInvalidSessionArchive
	}
	file, err := session.Open("replay.jsonl")
	if err != nil {
		return nil, errInvalidSessionArchive
	}
	defer file.Close()
	opened, err := file.Stat()
	if err != nil || !os.SameFile(info, opened) || !opened.Mode().IsRegular() {
		return nil, errInvalidSessionArchive
	}
	reader := &io.LimitedReader{R: file, N: meta.ReplayBytes}
	lines := bufio.NewScanner(reader)
	lines.Buffer(make([]byte, 64<<10), maxReplayRecord)
	var consumed int64
	for lines.Scan() {
		line := lines.Bytes()
		var row struct {
			Version int             `json:"version"`
			From    int64           `json:"from_seq"`
			To      int64           `json:"to_seq"`
			Message json.RawMessage `json:"message"`
		}
		if json.Unmarshal(line, &row) != nil || row.Version != meta.ReplayVersion || row.From <= 0 || row.To < row.From || row.To > meta.ReplayApplied || len(row.Message) == 0 || row.Message[0] != '{' {
			return nil, errInvalidSessionArchive
		}
		records = append(records, bytes.Clone(line))
		consumed += int64(len(line)) + 1
	}
	if lines.Err() != nil || consumed != meta.ReplayBytes || reader.N != 0 || len(records) != meta.MessageCount {
		return nil, errInvalidSessionArchive
	}
	current, err := session.Stat("replay.jsonl")
	if err != nil || !os.SameFile(opened, current) || !opened.ModTime().Equal(current.ModTime()) {
		return nil, errors.New("session_archive_changed")
	}
	return records, nil
}
