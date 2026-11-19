package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/coder/websocket"
)

func (c *client) verifyRelayFiles(ctx context.Context, ws *websocket.Conn, connect func() (*websocket.Conn, error), machine, session, ticket, content string) error {
	bytesToTransfer := bytes.Repeat([]byte(content), max(1, min(3000, (128<<10)/len(content))))
	uploadID, err := newUUID()
	if err != nil {
		return err
	}
	operation, err := newUUID()
	if err != nil {
		return err
	}
	digest := sha256.Sum256(bytesToTransfer)
	params := relayUploadParams{SessionID: session, UploadID: uploadID, Name: "验收 文件%.txt", Size: int64(len(bytesToTransfer)), SHA: hex.EncodeToString(digest[:])}
	var slot relayUpload
	if _, err := acpVerifyRPC(ctx, ws, 7, "_monkeyai/uploads/create", params, &slot); err != nil || slot.State != "uploading" || slot.Offset != 0 {
		return errors.Join(errors.New("could not create relay upload"), err)
	}
	complete := map[string]any{"sessionId": session, "uploadId": uploadID, "_meta": map[string]string{"operationId": operation}}
	if _, err := acpVerifyRPC(ctx, ws, 8, "_monkeyai/uploads/complete", complete, nil); err == nil {
		return errors.New("incomplete upload was published")
	}
	prefix := c.server + "/api/v1/endpoints/" + machine + "/proxy/"
	input := fileTransferInput{Server: c.server, MachineID: machine, URL: prefix + "uploads/" + uploadID, Method: http.MethodPut, Ticket: ticket, SessionID: session}
	put := func(offset int, chunk []byte, status, next int) error {
		input.Offset, input.Chunk = int64(offset), chunk
		meta, _, err := verifyTransfer(ctx, input)
		if err != nil || meta.Status != status || meta.Headers.Get("Upload-Offset") != strconv.Itoa(next) || meta.Headers.Get("Upload-Length") != strconv.Itoa(len(bytesToTransfer)) {
			return errors.Join(errors.New("upload offset mismatch"), err)
		}
		return nil
	}
	split := min(relayUploadChunk, len(bytesToTransfer)/2)
	if err := put(0, bytesToTransfer[:split], http.StatusNoContent, split); err != nil {
		return err
	}
	if err := ws.CloseNow(); err != nil {
		return err
	}
	resumed, err := connect()
	if err != nil {
		return err
	}
	defer resumed.CloseNow()
	if _, err := acpVerifyRPC(ctx, resumed, 1, "initialize", map[string]int{"protocolVersion": 1}, nil); err != nil {
		return err
	}
	if _, err := acpVerifyRPC(ctx, resumed, 2, "_monkeyai/uploads/status", map[string]string{"sessionId": session, "uploadId": uploadID}, &slot); err != nil || slot.Offset != int64(split) {
		return errors.Join(errors.New("upload did not resume at committed offset"), err)
	}
	if err := put(0, bytesToTransfer[:split], http.StatusConflict, split); err != nil {
		return err
	}
	for offset := split; offset < len(bytesToTransfer); {
		end := min(offset+relayUploadChunk, len(bytesToTransfer))
		if err := put(offset, bytesToTransfer[offset:end], http.StatusNoContent, end); err != nil {
			return err
		}
		offset = end
	}
	if _, err := acpVerifyRPC(ctx, resumed, 3, "_monkeyai/uploads/complete", complete, &slot); err != nil || slot.State != "completed" || slot.Ref == nil {
		return errors.Join(errors.New("uploaded bytes were not finalized"), err)
	}
	var repeated relayUpload
	if _, err := acpVerifyRPC(ctx, resumed, 4, "_monkeyai/uploads/complete", complete, &repeated); err != nil || repeated.Ref == nil || *repeated.Ref != *slot.Ref {
		return errors.Join(errors.New("upload completion was not idempotent"), err)
	}
	var queried struct {
		State  string
		Result relayUpload
	}
	if _, err := acpVerifyRPC(ctx, resumed, 6, "_monkeyai/operations/get", map[string]string{"operationId": operation}, &queried); err != nil ||
		queried.State != "completed" || queried.Result.Ref == nil || *queried.Result.Ref != *slot.Ref {
		return errors.Join(errors.New("upload result did not survive a status query"), err)
	}
	var link struct{ URL string }
	if _, err := acpVerifyRPC(ctx, resumed, 5, "_monkeyai/files/link", map[string]any{"fileRef": slot.Ref, "purpose": "download"}, &link); err != nil {
		return err
	}
	input = fileTransferInput{Server: c.server, MachineID: machine, URL: link.URL, Method: http.MethodHead, Ticket: ticket}
	meta, _, err := verifyTransfer(ctx, input)
	if err != nil || meta.Status != http.StatusOK || meta.Headers.Get("Content-Length") != strconv.Itoa(len(bytesToTransfer)) {
		return errors.Join(errors.New("download HEAD mismatch"), err)
	}
	input.Method, input.Range = http.MethodGet, "bytes=10-30"
	meta, received, err := verifyTransfer(ctx, input)
	if err != nil || meta.Status != http.StatusPartialContent || !bytes.Equal(received, bytesToTransfer[10:31]) {
		return errors.Join(errors.New("download Range mismatch"), err)
	}
	input.Range = ""
	meta, received, err = verifyTransfer(ctx, input)
	if err != nil || meta.Status != http.StatusOK || !bytes.Equal(received, bytesToTransfer) {
		return errors.Join(errors.New("download byte mismatch"), err)
	}
	input.URL = input.URL[:strings.LastIndexByte(input.URL, '/')+1] + "other.txt"
	meta, _, err = verifyTransfer(ctx, input)
	if err != nil || meta.Status != http.StatusForbidden {
		return errors.Join(errors.New("file grant accepted another filename"), err)
	}
	return nil
}

type transferMetadata struct {
	Status  int         `json:"status"`
	Headers http.Header `json:"headers"`
}

func verifyTransfer(ctx context.Context, in fileTransferInput) (transferMetadata, []byte, error) {
	var meta transferMetadata
	request, err := json.Marshal(in)
	if err != nil {
		return meta, nil, err
	}
	var output bytes.Buffer
	if err := streamFileTransfer(ctx, bytes.NewReader(request), &output); err != nil {
		return meta, nil, err
	}
	line, err := output.ReadBytes('\n')
	if err != nil {
		return meta, nil, err
	}
	if err := json.Unmarshal(line, &meta); err != nil {
		return meta, nil, err
	}
	data, err := io.ReadAll(&output)
	return meta, data, err
}
