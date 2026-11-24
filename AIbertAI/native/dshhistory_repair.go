package main

import (
	"context"
	"encoding/json"
	"errors"
)

var errDSHInvalidHistory = errors.New("dsh_invalid_history")
var errDSHHistoryTooLarge = errors.New("dsh_history_too_large")

// dshRepairHistory reads one immutable prefix from a durable SDK history fetch.
// A caller applying projections must persist each projection and sequence atomically.
func dshRepairHistory(ctx context.Context, session string, after int64, fetch dshConnectorHistoryFetch, apply func(int64, json.RawMessage) error) (int64, error) {
	if !validUUID(session) || after < -1 || fetch == nil || apply == nil {
		return after, errDSHInvalidHistory
	}
	var ceiling *int64
	count, bytes := 0, 0
	for {
		if err := ctx.Err(); err != nil {
			return after, err
		}
		page, used, err := dshFetchHistoryPage(ctx, session, after, ceiling, fetch)
		if err != nil {
			return after, err
		}
		bytes += used
		if bytes > dshConnectorHistoryMaxBytes {
			return after, errDSHHistoryTooLarge
		}
		if ceiling == nil {
			value := *page.Through
			ceiling = &value
		}
		previous := after
		for _, event := range page.Events {
			var e struct {
				Seq  *int64          `json:"seq"`
				Type string          `json:"type"`
				Data json.RawMessage `json:"data"`
			}
			if json.Unmarshal(event, &e) != nil || e.Seq == nil || (len(event) > 512<<10 && page.Large == nil) || len(e.Data) == 0 || *e.Seq != previous+1 || *e.Seq > *ceiling || e.Type == "" {
				return after, errDSHInvalidHistory
			}
			previous = *e.Seq
		}
		if previous < *ceiling && len(page.Events) == 0 {
			return after, errDSHInvalidHistory
		}
		count += len(page.Events)
		if count > dshConnectorHistoryMaxEvents {
			return after, errDSHHistoryTooLarge
		}
		for _, event := range page.Events {
			var e struct {
				Seq int64 `json:"seq"`
			}
			_ = json.Unmarshal(event, &e)
			if err := apply(e.Seq, event); err != nil {
				return after, err
			}
			after = e.Seq
		}
		if after == *ceiling {
			return after, nil
		}
	}
}
