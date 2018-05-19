# Progress

[![GoDoc](https://godoc.org/github.com/sfreiberg/progress?status.png)](https://godoc.org/github.com/sfreiberg/progress)

## About

Progress is a go library for creating a live progress bar into slack. Inspiration came from [slack-progress](https://github.com/bcicen/slack-progress/).

## Example

```go
token := "super-secret-slack-token"
channel := "demo"

pbar := progress.New(token, channel, nil)

for i := 0; i <= pbar.Opts.TotalUnits; i++ {
    if err := pbar.Update(i); err != nil {
        log.Printf("Error updating progress bar: %s\n", err)
    }
}
```