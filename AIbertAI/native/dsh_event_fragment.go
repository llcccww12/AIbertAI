package main

import (
	"encoding/base64"
	"encoding/json"
	"errors"
)

type dshEventAssembler struct {
	session, hash string
	seq           int64
	total         int
	data          []byte
}

func (a *dshEventAssembler) accept(f dshFrame) (dshFrame, bool, error) {
	if f.Method != "aibertai/event-fragment" {
		if a.data != nil {
			return dshFrame{}, false, errors.New("dsh_invalid_event_fragment")
		}
		return f, true, nil
	}
	var x struct {
		Session      string `json:"sessionId"`
		Seq          *int64 `json:"seq"`
		Offset       *int   `json:"offset"`
		Total        int    `json:"total"`
		SHA256, Data string
	}
	if _, err := strictDSHJSON(f.Params); err != nil || json.Unmarshal(f.Params, &x) != nil || !validUUID(x.Session) || x.Seq == nil || *x.Seq < 0 || x.Offset == nil || x.Total <= 512<<10 || x.Total > dshConnectorHistoryMaxBytes || !validDesktopHash(x.SHA256) {
		return dshFrame{}, false, errors.New("dsh_invalid_event_fragment")
	}
	if a.data == nil {
		if *x.Offset != 0 {
			return dshFrame{}, false, errors.New("dsh_invalid_event_fragment")
		}
		a.session, a.hash, a.seq, a.total = x.Session, x.SHA256, *x.Seq, x.Total
		a.data = make([]byte, 0, x.Total)
	}
	if x.Session != a.session || x.SHA256 != a.hash || *x.Seq != a.seq || x.Total != a.total || *x.Offset != len(a.data) {
		return dshFrame{}, false, errors.New("dsh_invalid_event_fragment")
	}
	b, err := base64.StdEncoding.Strict().DecodeString(x.Data)
	expected := min(128<<10, a.total-len(a.data))
	if err != nil || len(b) != expected {
		return dshFrame{}, false, errors.New("dsh_invalid_event_fragment")
	}
	a.data = append(a.data, b...)
	if len(a.data) < a.total {
		return dshFrame{}, false, nil
	}
	if desktopHash(a.data) != a.hash {
		return dshFrame{}, false, errors.New("dsh_invalid_event_fragment")
	}
	if _, err := strictDSHJSON(a.data); err != nil {
		return dshFrame{}, false, errors.New("dsh_invalid_event_fragment")
	}
	var payload struct {
		Session string `json:"sessionId"`
		Event   struct {
			Seq *int64 `json:"seq"`
		} `json:"event"`
	}
	if json.Unmarshal(a.data, &payload) != nil || payload.Session != a.session || payload.Event.Seq == nil || *payload.Event.Seq != a.seq {
		return dshFrame{}, false, errors.New("dsh_invalid_event_fragment")
	}
	params := append(json.RawMessage(nil), a.data...)
	*a = dshEventAssembler{}
	return dshFrame{JSONRPC: "2.0", Method: "session.event", Params: params}, true, nil
}
