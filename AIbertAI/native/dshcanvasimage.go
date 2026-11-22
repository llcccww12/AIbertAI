package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"os"
	"regexp"
	"strings"
	"unicode/utf8"
)

const dshGeneratedImagePattern = `^\.aibertai/media/generated/[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}\.(png|jpg|jpeg|webp)$`

var dshGeneratedImagePath = regexp.MustCompile(dshGeneratedImagePattern)
var errDSHCanvasImageSource = errors.New("dsh_canvas_image_source_invalid")

func dshCanvasImageSource(raw json.RawMessage) (string, error) {
	var input map[string]json.RawMessage
	var path string
	if json.Unmarshal(raw, &input) != nil || json.Unmarshal(input["source_path"], &path) != nil || !dshGeneratedImagePath.MatchString(path) {
		return "", errDSHCanvasImageSource
	}
	for key, value := range input {
		if key == "source_path" {
			continue
		}
		var title string
		if key != "title" || json.Unmarshal(value, &title) != nil || strings.TrimSpace(title) == "" || utf8.RuneCountInString(title) > 160 {
			return "", errDSHCanvasImageSource
		}
	}
	return path, nil
}

// Shared project folders can contain other sessions' files. Bind the exact source
// and its bytes to the completed generation in this session's host journal.
func dshGeneratedImageDigest(journal, workspace *os.Root, row acpSession, source string) (string, error) {
	if journal == nil || workspace == nil || !validUUID(row.ID) || row.ID != row.DSHSessionID || !dshGeneratedImagePath.MatchString(source) {
		return "", errDSHCanvasImageSource
	}
	for _, turn := range row.Turns {
		for _, run := range turn.ToolRuns {
			if run.Pending || run.Report.Failed != 0 || run.DSHCallID == "" || run.Call == nil || run.Call.Function.Name != "media_generate_image" {
				continue
			}
			var result struct {
				Source string `json:"source_path"`
				Digest string `json:"source_sha256"`
			}
			if json.Unmarshal([]byte(run.Output), &result) != nil || result.Source != source {
				continue
			}
			raw, err := hex.DecodeString(result.Digest)
			if err != nil || len(raw) != sha256.Size || result.Digest != hex.EncodeToString(raw) {
				continue
			}
			claim, err := readConnectorClaim(journal, connectorClaimPath(row.ID, turn.Operation, run.DSHCallID), row.ID)
			if err != nil || claim.OperationID != turn.Operation || claim.CallID != run.DSHCallID || claim.State != "reconciled" || claim.Kind != "canvas" || claim.Output == nil || *claim.Output != dshCanvasResultDigest(run.Output) {
				continue
			}
			const maxImage = 30 << 20 // same limit as canvas uploads in the worker
			target, err := dshWorkspaceTarget(workspace, source)
			if err != nil || target.File == nil {
				return "", errDSHCanvasImageSource
			}
			f, err := openAttachmentFile(workspace, source, maxImage)
			if err != nil {
				return "", errDSHCanvasImageSource
			}
			hasher := sha256.New()
			n, readErr := io.Copy(hasher, io.LimitReader(f, maxImage+1))
			closeErr := f.Close()
			if readErr != nil || closeErr != nil || n == 0 || n > maxImage || hex.EncodeToString(hasher.Sum(nil)) != result.Digest {
				return "", errDSHCanvasImageSource
			}
			return result.Digest, nil
		}
	}
	return "", errDSHCanvasImageSource
}
