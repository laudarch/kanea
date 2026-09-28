package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"time"
)

// runImages is `kanea images [--clean]` (PRD v1.111, §16.2): the node's
// containerd images with the collector's in-use verdict, and the manual GC
// sweep. One verb for both because the sweep's natural next question ("what
// is left?") is the list, and the list's ("can I free this?") is the sweep.
func runImages(args []string) error {
	fs := flag.NewFlagSet("images", flag.ContinueOnError)
	ep := endpointFlags(fs)
	clean := fs.Bool("clean", false, "remove unused images now (one GC sweep)")
	asJSON := fs.Bool("json", false, "machine-readable output")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() > 0 {
		return fmt.Errorf("kanea images takes no arguments (got %q); did you mean --clean?", fs.Arg(0))
	}

	client, err := ep.client()
	if err != nil {
		return err
	}
	ctx := context.Background()
	o := newOut()

	if *clean {
		summary, err := client.ImagesClean(ctx)
		if err != nil {
			return err
		}
		if *asJSON {
			body, err := json.MarshalIndent(summary, "", "  ")
			if err != nil {
				return err
			}
			o.println(string(body))
			return o.Err()
		}
		if summary.Removed == 0 {
			o.println("Nothing to remove: every image is in use, kept, or younger than min_age.")
			return o.Err()
		}
		o.printf("Removed %d unused image(s), reclaiming %s.\n",
			summary.Removed, humanBytes(summary.ReclaimedBytes))
		return o.Err()
	}

	resp, err := client.Images(ctx)
	if err != nil {
		return err
	}
	if *asJSON {
		body, err := json.MarshalIndent(resp, "", "  ")
		if err != nil {
			return err
		}
		o.println(string(body))
		return o.Err()
	}
	if len(resp.Images) == 0 {
		o.println("No images.")
		return o.Err()
	}

	o.table()
	o.println("PROJECT\tIMAGE\tSIZE\tAGE\tIN-USE")
	for _, img := range resp.Images {
		inUse := ""
		if img.InUse {
			inUse = "yes"
		}
		o.printf("%s\t%s\t%s\t%s\t%s\n",
			img.Project, img.Ref, humanBytes(img.SizeBytes),
			shortDuration(time.Since(img.CreatedAt)), inUse)
	}
	o.endTable()
	return o.Err()
}
