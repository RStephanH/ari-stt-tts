// transcribe_test.go

package stt

import (
	"testing"

	apiinterfaces "github.com/deepgram/deepgram-go-sdk/v3/pkg/api/listen/v1/rest/interfaces"
)

func TestExtractTranscript(t *testing.T) {
	tests := []struct {
		name string
		resp *apiinterfaces.PreRecordedResponse
		want string
	}{
		{
			name: "nil results",
			resp: &apiinterfaces.PreRecordedResponse{RequestID: "req-1"},
			want: "",
		},
		{
			name: "empty channels",
			resp: &apiinterfaces.PreRecordedResponse{
				RequestID: "req-2",
				Results:   &apiinterfaces.Result{Channels: []apiinterfaces.Channel{}},
			},
			want: "",
		},
		{
			name: "valid transcript",
			resp: &apiinterfaces.PreRecordedResponse{
				RequestID: "req-3",
				Results: &apiinterfaces.Result{
					Channels: []apiinterfaces.Channel{
						{Alternatives: []apiinterfaces.Alternative{{Transcript: "hello world"}}},
					},
				},
			},
			want: "hello world",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractTranscript(tt.resp)
			if got.Text != tt.want {
				t.Errorf("Text = %q, want %q", got.Text, tt.want)
			}
			if got.RequestID != tt.resp.RequestID {
				t.Errorf("RequestID = %q, want %q", got.RequestID, tt.resp.RequestID)
			}
		})
	}
}
