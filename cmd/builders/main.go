// Read-only companion to cmd/example for builder registration and intent/bundle reconciliation.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/storm-trade/sdk-go/sequencer"
	"os"
	"strconv"
	"time"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
func run() error {
	baseURL := os.Getenv("STORM_V3_URL")
	if baseURL == "" {
		baseURL = sequencer.TestnetURL
	}
	c := sequencer.NewClient(baseURL)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	var value any
	var err error
	if len(os.Args) < 2 {
		return fmt.Errorf("usage: builders status|registry|intent HASH|bundles SA|bundle SA EXECUTOR_QUERY_ID")
	}
	switch os.Args[1] {
	case "status":
		value, err = c.GetStatus(ctx)
	case "registry":
		value, err = c.GetBuilders(ctx)
	case "intent":
		if len(os.Args) != 3 {
			return fmt.Errorf("intent requires a hash")
		}
		value, err = c.GetIntentStatus(ctx, os.Args[2])
	case "bundles":
		if len(os.Args) != 3 {
			return fmt.Errorf("bundles requires a Smart Account address")
		}
		value, err = c.GetAccountBundles(ctx, os.Args[2])
	case "bundle":
		if len(os.Args) != 4 {
			return fmt.Errorf("bundle requires SA and executor query ID")
		}
		id, e := strconv.ParseUint(os.Args[3], 10, 64)
		if e != nil {
			return e
		}
		value, err = c.GetAccountBundle(ctx, os.Args[2], id)
	default:
		return fmt.Errorf("unknown command")
	}
	if err != nil {
		return err
	}
	return json.NewEncoder(os.Stdout).Encode(value)
}
