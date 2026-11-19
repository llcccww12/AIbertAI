package main

import (
	"bufio"
	"errors"
	"io"
	"strings"
	"testing"
	"testing/iotest"
)

func TestResponsesReadsToEOFBeforeReportingCompletion(t *testing.T) {
	stream := "data: {\"type\":\"response.output_text.delta\",\"delta\":\"test\"}\n\n" +
		"data: {\"type\":\"response.completed\",\"response\":{\"usage\":{\"input_tokens\":3,\"output_tokens\":4}}}\n\n"

	finalRead := errors.New("connection interrupted")
	for _, tc := range []struct {
		name   string
		reader io.Reader
		want   error
	}{
		{"graceful end", strings.NewReader(stream), nil},
		{"terminal event then read error", io.MultiReader(strings.NewReader(stream), iotest.ErrReader(finalRead)), finalRead},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var output string
			usage, stop, err := readResponses(bufio.NewScanner(tc.reader), func(text string) error {
				output += text
				return nil
			})
			if !errors.Is(err, tc.want) || output != "test" {
				t.Fatalf("output=%q err=%v", output, err)
			}
			if tc.want == nil && (stop != "stop" || usage.Output == nil || *usage.Output != 4) {
				t.Fatalf("stop=%q usage=%v", stop, usage)
			}
		})
	}
}
