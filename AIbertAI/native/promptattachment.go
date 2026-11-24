package main

import (
	"errors"
	"io"
	"os"
)

// readPromptAttachment validates a staged attachment and reads at most the
// remaining per-prompt byte budget. Callers must subtract the returned length
// before reading another file, and retain the original acpAttachment for
// history; the bytes or its name are not authorization for later access.
func (h *relaySessionHost) openPromptAttachment(session string, file acpAttachment, remaining int64) (*os.File, error) {
	var original *os.File
	var err error
	if h.local != nil {
		original, err = openDesktopAttachment(h.local.attachments, file, remaining)
	} else if h.uploads != nil {
		original, err = h.uploads.promptFile(file.Ref, session, file.Name, remaining)
	}
	if err != nil || original == nil {
		return nil, errors.New("attachment_unavailable_or_too_large")
	}
	return original, nil
}

func (h *relaySessionHost) readPromptAttachment(session string, file acpAttachment, remaining int64) ([]byte, error) {
	original, err := h.openPromptAttachment(session, file, remaining)
	if err != nil {
		return nil, err
	}
	data, err := io.ReadAll(io.LimitReader(original, remaining+1))
	_ = original.Close()
	if err != nil || int64(len(data)) > remaining {
		return nil, errors.New("attachment_unavailable_or_too_large")
	}
	return data, nil
}
