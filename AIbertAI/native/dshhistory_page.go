package main

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"unicode/utf8"
)

type dshHistoryLargeEvent struct {
	Seq    *int64 `json:"seq"`
	Total  int    `json:"total"`
	SHA256 string `json:"sha256"`
}
type dshVerifiedHistoryPage struct {
	SessionID string                `json:"sessionId"`
	Through   *int64                `json:"throughSeq"`
	Events    []json.RawMessage     `json:"events"`
	Large     *dshHistoryLargeEvent `json:"largeEvent"`
}

// Fetches a fixed-prefix page, authenticating and reassembling an oversized event
// before exposing it to any projection. The existing total budget includes fragments.
func dshFetchHistoryPage(ctx context.Context, session string, after int64, ceiling *int64, fetch dshConnectorHistoryFetch) (dshVerifiedHistoryPage, int, error) {
	var page dshVerifiedHistoryPage
	params := map[string]any{"sessionId": session, "afterSeq": after}
	if ceiling != nil {
		params["throughSeq"] = *ceiling
	}
	raw, err := fetch(ctx, params)
	if err != nil {
		return page, 0, err
	}
	used := len(raw)
	if used > 1<<20 {
		return page, used, errDSHHistoryTooLarge
	}
	if _, err = strictDSHJSON(raw); err != nil || json.Unmarshal(raw, &page) != nil || page.SessionID != session || page.Through == nil || len(page.Events) > 128 || *page.Through < after || ceiling != nil && *page.Through != *ceiling {
		return page, used, errDSHInvalidHistory
	}
	if page.Large == nil {
		return page, used, nil
	}
	d := page.Large
	hash, hashErr := hex.DecodeString(d.SHA256)
	if len(page.Events) != 0 || d.Seq == nil || *d.Seq != after+1 || *d.Seq > *page.Through || d.Total <= 512<<10 || d.Total > dshConnectorHistoryMaxBytes || hashErr != nil || len(hash) != sha256.Size || hex.EncodeToString(hash) != d.SHA256 {
		return page, used, errDSHInvalidHistory
	}
	event := make([]byte, 0, d.Total)
	for len(event) < d.Total {
		if err := ctx.Err(); err != nil {
			return page, used, err
		}
		offset := len(event)
		fragment, err := fetch(ctx, map[string]any{"sessionId": session, "afterSeq": after, "throughSeq": *page.Through, "fragmentSeq": *d.Seq, "offset": offset})
		if err != nil {
			return page, used, err
		}
		if len(fragment) > 1<<20 {
			return page, used, errDSHHistoryTooLarge
		}
		var f struct {
			SessionID string `json:"sessionId"`
			Through   *int64 `json:"throughSeq"`
			Seq       *int64 `json:"seq"`
			Offset    *int   `json:"offset"`
			Total     int    `json:"total"`
			SHA256    string `json:"sha256"`
			Data      string `json:"data"`
		}
		if _, err := strictDSHJSON(fragment); err != nil || json.Unmarshal(fragment, &f) != nil || f.SessionID != session || f.Through == nil || *f.Through != *page.Through || f.Seq == nil || *f.Seq != *d.Seq || f.Offset == nil || *f.Offset != offset || f.Total != d.Total || f.SHA256 != d.SHA256 {
			return page, used, errDSHInvalidHistory
		}
		b, err := base64.StdEncoding.Strict().DecodeString(f.Data)
		expected := d.Total - offset
		if expected > 512<<10 {
			expected = 512 << 10
		}
		if err != nil || len(b) != expected {
			return page, used, errDSHInvalidHistory
		}
		event = append(event, b...)
	}
	sum := sha256.Sum256(event)
	if hex.EncodeToString(sum[:]) != d.SHA256 || !utf8.Valid(event) {
		return page, used, errDSHInvalidHistory
	}
	if _, err := strictDSHJSON(event); err != nil {
		return page, used, errDSHInvalidHistory
	}
	page.Events = []json.RawMessage{event}
	return page, used + len(event), nil
}
