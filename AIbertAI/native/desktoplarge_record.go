package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
)

// An immutable content-addressed chunk set keeps the small record as the commit
// point. Old chunks remain until the replacement record is durable.
const maxDesktopLargeRecordBytes = 32 << 20
const desktopRecordChunkBytes = 1 << 20

type desktopChunkRecord struct {
	Format string   `json:"_aibertaiChunkRecord"`
	Size   int      `json:"size"`
	SHA256 string   `json:"sha256"`
	Chunks []string `json:"chunks"`
}

func desktopHash(data []byte) string { h := sha256.Sum256(data); return hex.EncodeToString(h[:]) }
func validDesktopHash(s string) bool {
	b, err := hex.DecodeString(s)
	return err == nil && len(b) == sha256.Size && hex.EncodeToString(b) == s
}
func readDesktopChunk(root *os.Root, name string, maxSize int) ([]byte, error) {
	info, err := root.Lstat(name)
	if err != nil || !info.Mode().IsRegular() || info.Size() > int64(maxSize) {
		return nil, errors.New("invalid_desktop_session_snapshot")
	}
	file, err := root.Open(name)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	opened, err := file.Stat()
	if err != nil || !os.SameFile(info, opened) {
		return nil, errors.New("desktop_session_snapshot_changed")
	}
	data, err := io.ReadAll(io.LimitReader(file, int64(maxSize)+1))
	if err != nil || len(data) > maxSize {
		return nil, errors.New("invalid_desktop_session_snapshot")
	}
	return data, nil
}
func readDesktopLargeRecord(root *os.Root, name string, record desktopChunkRecord, destination any) error {
	if record.Format != "v1" || record.Size <= maxDesktopSessionBytes || record.Size > maxDesktopLargeRecordBytes || !validDesktopHash(record.SHA256) || len(record.Chunks) != (record.Size+desktopRecordChunkBytes-1)/desktopRecordChunkBytes {
		return errors.New("invalid_desktop_session_snapshot")
	}
	var data []byte
	for i, hash := range record.Chunks {
		if !validDesktopHash(hash) {
			return errors.New("invalid_desktop_session_snapshot")
		}
		chunk, err := readDesktopChunk(root, filepath.Join(name+".chunks", hash), desktopRecordChunkBytes)
		expected := min(desktopRecordChunkBytes, record.Size-i*desktopRecordChunkBytes)
		if err != nil || len(chunk) != expected || desktopHash(chunk) != hash {
			return errors.New("invalid_desktop_session_snapshot")
		}
		data = append(data, chunk...)
	}
	if desktopHash(data) != record.SHA256 {
		return errors.New("invalid_desktop_session_snapshot")
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	if decoder.Decode(destination) != nil || decoder.Decode(new(any)) != io.EOF {
		return errors.New("invalid_desktop_session_snapshot")
	}
	return nil
}
func syncDesktopDir(root *os.Root, dir string) error {
	f, err := root.Open(dir)
	if err != nil {
		return err
	}
	defer f.Close()
	return syncDirFile(f)
}

// Called under the session store lock, only after the replacement commit is durable.
func pruneDesktopChunks(root *os.Root, name string, keep map[string]bool) {
	dir := name + ".chunks"
	f, err := root.Open(dir)
	if err != nil {
		return
	}
	defer f.Close()
	entries, err := f.ReadDir(-1)
	if err != nil {
		return
	}
	for _, entry := range entries {
		if validDesktopHash(entry.Name()) && !keep[entry.Name()] {
			_ = root.Remove(filepath.Join(dir, entry.Name()))
		}
	}
}
func saveDesktopLargeRecord(root *os.Root, name string, data []byte) error {
	if len(data) > maxDesktopLargeRecordBytes {
		return errors.New("desktop_session_snapshot_too_large")
	}
	dir := name + ".chunks"
	if err := root.MkdirAll(dir, 0700); err != nil {
		return err
	}
	info, err := root.Lstat(dir)
	if err != nil || !info.IsDir() {
		return errors.New("invalid_desktop_session_snapshot")
	}
	record := desktopChunkRecord{Format: "v1", Size: len(data), SHA256: desktopHash(data)}
	for start := 0; start < len(data); start += desktopRecordChunkBytes {
		chunk := data[start:min(start+desktopRecordChunkBytes, len(data))]
		hash := desktopHash(chunk)
		path := filepath.Join(dir, hash)
		if err := persistSourceState(root, path, chunk, true); err != nil {
			if !errors.Is(err, os.ErrExist) {
				return err
			}
			existing, e := readDesktopChunk(root, path, desktopRecordChunkBytes)
			if e != nil || !bytes.Equal(existing, chunk) {
				return errors.New("invalid_desktop_session_snapshot")
			}
		}
		record.Chunks = append(record.Chunks, hash)
	}
	if err := syncDesktopDir(root, dir); err != nil {
		return err
	}
	wrapper, err := json.Marshal(record)
	if err != nil {
		return err
	}
	if err := writeSourceState(root, name, wrapper); err != nil {
		return err
	}
	if err := syncDesktopDir(root, filepath.Dir(name)); err != nil {
		return err
	}
	keep := make(map[string]bool, len(record.Chunks))
	for _, hash := range record.Chunks {
		keep[hash] = true
	}
	pruneDesktopChunks(root, name, keep)
	return nil
}
