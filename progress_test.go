package progress_test

import (
	"log"
	"os"
	"testing"

	"github.com/sfreiberg/progress"
)

func TestProgressBar(t *testing.T) {
	var (
		token   = os.Getenv("SLACK_TOKEN")
		channel = os.Getenv("SLACK_CHANNEL")
	)

	if token == "" || channel == "" {
		t.Fatalf("You must set the SLACK_TOKEN and SLACK_CHANNEL environment variables.")
	}

	pbar := progress.New(token, channel, nil)

	for i := 0; i <= pbar.Opts.TotalUnits; i++ {
		if err := pbar.Update(i); err != nil {
			log.Printf("Error updating progress bar: %s\n", err)
		}
	}
}
