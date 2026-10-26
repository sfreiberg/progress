package progress

import (
	"bytes"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/slack-go/slack"
)

func TestDrawBar(t *testing.T) {
	tests := []struct {
		width, pct int
		want       string
	}{
		{10, 0, "----------"},
		{10, 55, "#####-----"},
		{10, 100, "##########"},
		{20, 50, "##########----------"},
		{20, 100, "####################"},
		{5, 100, "#####"}, // used to panic: 100/5 = 20 filled for a width of 5
		{4, 50, "##--"},
	}

	for _, tt := range tests {
		p := &Progress{Opts: &Options{Fill: "#", Empty: "-", Width: tt.width}}
		if got := p.drawBar(tt.pct); got != tt.want {
			t.Errorf("drawBar(%d) with width %d = %q, want %q", tt.pct, tt.width, got, tt.want)
		}
	}
}

func TestDrawBarMultiRuneFill(t *testing.T) {
	// Fill and Empty can be multi-character strings such as Slack emoji
	// shortcodes; the bar should still be Width segments long.
	p := &Progress{Opts: &Options{Fill: ":x:", Empty: ":o:", Width: 4}}
	if got, want := p.drawBar(50), ":x::x::o::o:"; got != want {
		t.Errorf("drawBar(50) = %q, want %q", got, want)
	}
}

// fakeSlack answers Slack Web API calls. Requests to methods listed in fail
// get an error response.
type fakeSlack struct {
	calls []string
	fail  map[string]bool
}

func (f *fakeSlack) Do(req *http.Request) (*http.Response, error) {
	method := req.URL.Path[strings.LastIndex(req.URL.Path, "/")+1:]
	f.calls = append(f.calls, method)

	body := `{"ok":true,"channel":"C123","ts":"1.000"}`
	if f.fail[method] {
		body = `{"ok":false,"error":"internal_error"}`
	}
	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": {"application/json"}},
		Body:       io.NopCloser(bytes.NewBufferString(body)),
		Request:    req,
	}, nil
}

func newTestProgress(f *fakeSlack) *Progress {
	p := New("token", "general", nil)
	p.client = slack.New("token", slack.OptionHTTPClient(f))
	return p
}

func TestUpdateRetriesAfterFailedPost(t *testing.T) {
	f := &fakeSlack{fail: map[string]bool{"chat.postMessage": true}}
	p := newTestProgress(f)

	if err := p.Update(10); err == nil {
		t.Fatal("Update() with a failing chat.postMessage: expected an error")
	}
	if p.channel != "general" {
		t.Errorf("channel = %q after a failed post, want it unchanged", p.channel)
	}

	f.fail = nil
	if err := p.Update(10); err != nil {
		t.Fatalf("Update() retry: %v", err)
	}
	if p.channel != "C123" || p.ts != "1.000" {
		t.Errorf("channel, ts = %q, %q, want C123, 1.000", p.channel, p.ts)
	}
}

func TestUpdateKeepsMessageAfterFailedUpdate(t *testing.T) {
	f := &fakeSlack{}
	p := newTestProgress(f)

	if err := p.Update(10); err != nil {
		t.Fatal(err)
	}

	f.fail = map[string]bool{"chat.update": true}
	if err := p.Update(20); err == nil {
		t.Fatal("Update() with a failing chat.update: expected an error")
	}

	f.fail = nil
	if err := p.Update(30); err != nil {
		t.Fatal(err)
	}

	want := []string{"chat.postMessage", "chat.update", "chat.update"}
	if strings.Join(f.calls, ",") != strings.Join(want, ",") {
		t.Errorf("slack calls = %v, want %v (a failed update must not start a new message)", f.calls, want)
	}
}

func TestUpdateSkipsUnchangedPercent(t *testing.T) {
	f := &fakeSlack{}
	p := newTestProgress(f)

	for _, pos := range []int{10, 10, 20} {
		if err := p.Update(pos); err != nil {
			t.Fatal(err)
		}
	}

	want := []string{"chat.postMessage", "chat.update"}
	if strings.Join(f.calls, ",") != strings.Join(want, ",") {
		t.Errorf("slack calls = %v, want %v", f.calls, want)
	}
}
