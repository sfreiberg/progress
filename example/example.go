package main

import (
	"log"

	"github.com/sfreiberg/progress"
)

func main() {
	token := "super-secret-slack-token"
	channel := "demo"

	pbar := progress.New(token, channel, nil)

	for i := 0; i <= pbar.Opts.TotalUnits; i++ {
		if err := pbar.Update(i); err != nil {
			log.Printf("Error updating progress bar: %s\n", err)
		}
	}
}
