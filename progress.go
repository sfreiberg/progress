// Package progress is a small library for creating a progress bar in slack
package progress

import (
	"errors"
	"strings"
	"text/template"
	"time"

	"github.com/nlopes/slack"
)

var (
	// ErrMaxPosExceeded is returned when the position passed to Progress.Update
	// is greater than Progress.Total or DefaultTotal if Progress.Total is 0.
	ErrMaxPosExceeded = errors.New("Maximum position exceeded")

	// ErrNegativePos is returned when a negative value is passed to Progress.Update.
	// All positions should be >= 0.
	ErrNegativePos = errors.New("Invalid position")
)

// Options can be used to customize look of the progress bar. DefaultOptions() has pretty good defaults.
type Options struct {
	Fill        string // The character(s) used to fill in the progress bar
	Empty       string // The character(s) used to indicate empty space at the end of progress bar
	Width       int    // How many characters wide the progress bar should be. A value of 10 looks good on slack phone clients.
	TotalUnits  int    // Total possible units. Graph will always display 0-100%.
	Msg         string // The message template that will be sent to slack. Uses text/template for creating templates.
	Task        string // Name of the task we are showing progress for.
	AsUser      bool   // Whether or not to post as the user. If false posts as a generic bot and doesn't show edited next to messages. If true the opposite of both is true. Defaults to false.
	ShowEstTime bool   // Whether or not to show estimated time remaining
}

// DefaultOptions creates an Options struct with decent defaults.
func DefaultOptions(task string) *Options {
	return &Options{
		Fill:       "⬛",
		Empty:      "⬜",
		Width:      10, // Looks good on slack phone clients
		TotalUnits: 100,
		Msg: "{{.Task}}\n`{{.ProgBar}}` {{.Pos}}%\n" +
			"{{ if .ShowEstTime }}" +
			"{{ if .Complete }}Completed in *{{ .Elapsed }}*" +
			"{{ else }}{{ .Remaining }} remaining...{{ end }}" +
			"{{ end }}",
		Task:        task,
		ShowEstTime: true,
	}
}

// Progress is a struct that creates the progress bar in slack
type Progress struct {
	Opts    *Options
	Start   time.Time     // When the task began running. Initialized to current time when New() is called.
	client  *slack.Client // Slack client
	channel string        // Channel to post progress bar to
	ts      string        // The last timestamp we saw. Used for editing the progress bar
	lastPct int           // The last percent that was posted to slack. No reason to update if nothing has changed.
}

// Update either posts a new progress bar if this is the first call or updates an existing progress bar.
func (p *Progress) Update(pos int) error {
	if pos < 0 {
		return ErrNegativePos
	}

	if pos > p.Opts.TotalUnits {
		return ErrMaxPosExceeded
	}

	pct := int(float32(pos) / float32(p.Opts.TotalUnits) * 100)
	if pct <= p.lastPct { // We haven't progressed so no need to update slack
		return nil
	}

	msg, err := p.msg(pct)
	if err != nil {
		return err
	}

	// If there's no timestamp this is the first time we've run so post a normal message
	if p.ts == "" {
		msgOpts := []slack.MsgOption{
			slack.MsgOptionText(msg, false),
			slack.MsgOptionAsUser(p.Opts.AsUser),
		}
		p.channel, p.ts, _, err = p.client.SendMessage(p.channel, msgOpts...)
		return err
	}

	_, ts, _, err := p.client.UpdateMessage(p.channel, p.ts, msg)
	p.ts = ts
	p.lastPct = pct
	return err
}

func (p *Progress) drawBar(pos int) string {
	if pos == 0 {
		return strings.Repeat(p.Opts.Empty, p.Opts.Width)
	}

	bar := strings.Repeat(p.Opts.Fill, pos/p.Opts.Width)
	bar += strings.Repeat(p.Opts.Empty, p.Opts.Width-len([]rune(bar)))

	return bar
}

func (p *Progress) msg(pos int) (string, error) {
	msg := &strings.Builder{}

	data := map[string]interface{}{
		"Task":        p.Opts.Task,
		"ProgBar":     p.drawBar(pos),
		"Pos":         pos,
		"Remaining":   p.remaining(pos),
		"Complete":    pos == 100,
		"Elapsed":     time.Now().Sub(p.Start).Round(time.Millisecond),
		"ShowEstTime": p.Opts.ShowEstTime,
	}

	tmpl, err := template.New("msg").Parse(p.Opts.Msg)
	if err != nil {
		return "", err
	}
	err = tmpl.Execute(msg, data)

	return msg.String(), err
}

// Calculate the remaining time
func (p *Progress) remaining(pct int) time.Duration {
	elapsed := time.Now().Sub(p.Start)
	estTime := time.Duration(elapsed.Nanoseconds() / int64(pct) * int64(100))
	remaining := estTime - elapsed
	return remaining.Round(time.Second)
}

// New creates a new progress bar. If opts is nil then Progress will be created
// with DefaultOptions. The timer that is used for calculating time remaining
// is based on when this is instantiated so if it's not called around the time
// the task begins running it might report inaccurate results. You can fix this
// by setting Progress.Start manually.
func New(token, channel string, opts *Options) *Progress {
	progress := &Progress{
		client:  slack.New(token),
		channel: channel,
		Start:   time.Now(),
		Opts:    opts,
	}

	if opts == nil {
		progress.Opts = DefaultOptions("Unknown Task")
	}

	return progress
}
