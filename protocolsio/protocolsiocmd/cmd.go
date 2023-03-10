// Copyright 2023 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

// Package protocolsiocmd provides support for building command line tools
// that access protocols.io.
package protocolsiocmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"text/template"

	"cloudeng.io/cmdutil/flags"
	"cloudeng.io/file/checkpoint"
	"cloudeng.io/file/content"
	"cloudeng.io/path"
	"cloudeng.io/webapi/operations"
	"cloudeng.io/webapi/operations/apicrawlcmd"
	"cloudeng.io/webapi/protocolsio/protocolsiosdk"
)

type CommonFlags struct {
	Config string `subcmd:"config,$HOME/.protocolsio.yaml,'protocols.io config file'"`
}

type GetFlags struct {
	CommonFlags
}

type CrawlFlags struct {
	CommonFlags
	Save             bool               `subcmd:"save,true,'save downloaded protocols to disk'"`
	IgnoreCheckpoint bool               `subcmd:"ignore-checkpoint,false,'ignore the checkpoint files'"`
	Pages            flags.IntRangeSpec `subcmd:"pages,,page range to return"`
	PageSize         int                `subcmd:"size,50,number of items in each page"`
	Key              string             `subcmd:"key,,'string may contain any characters, numbers and special symbols. System will search around protocol name, description, authors. If the search keywords are enclosed in double quotes, then result contains only the exact match of the combined term'"`
}

type ScanFlags struct {
	Template string `subcmd:"template,'{{.ID}}',template to use for printing fields in the downloaded Protocol objects"`
}

// Çommand implements the command line operations available for protocols.io.
type Command struct {
	Config
}

// NewCommand returns a new Command instance for the specified API crawl.
func NewCommand(crawls apicrawlcmd.Crawls, name string) (*Command, error) {
	c := &Command{}
	ok, err := apicrawlcmd.ParseCrawlConfig(crawls, name, (*apicrawlcmd.Crawl[Service])(&c.Config))
	if !ok {
		return nil, fmt.Errorf("no configuration found for %v", name)
	}
	if err != nil {
		return nil, err
	}
	return c, nil
}

func handleCrawledObject(ctx context.Context,
	save bool,
	sharder path.Sharder,
	cachePath string,
	chk checkpoint.Operation,
	obj content.Object[protocolsiosdk.ProtocolPayload, operations.Response]) error {

	if obj.Response.Current != 0 && obj.Response.Total != 0 {
		fmt.Printf("progress: %v/%v\n", obj.Response.Current, obj.Response.Total)
	}
	if obj.Value.Protocol.ID == 0 {
		// Protocol is up-to-date on disk.
		return nil
	}
	fmt.Printf("protocol: %v\n", obj.Value.Protocol.ID)
	if !save {
		return nil
	}
	// Save the protocol object to disk.
	prefix, suffix := sharder.Assign(fmt.Sprintf("%v", obj.Value.Protocol.ID))
	path := filepath.Join(cachePath, prefix, suffix)
	if err := obj.WriteObjectBinary(path); err != nil {
		fmt.Printf("failed to write: %v as %v: %v\n", obj.Value.Protocol.ID, path, err)
	}
	if state := obj.Response.Checkpoint; len(state) > 0 {
		name, err := chk.Checkpoint(ctx, "", state)
		if err != nil {
			fmt.Printf("failed to save checkpoint: %v: %v\n", name, err)
		} else {
			fmt.Printf("checkpoint: %v\n", name)
		}
	}
	return nil
}

func (c *Command) Crawl(ctx context.Context, cacheRoot string, fv *CrawlFlags) error {
	cachePath, op, err := c.Cache.Initialize(cacheRoot)
	if err != nil {
		return err
	}

	if fv.IgnoreCheckpoint {
		op = nil
	}

	sharder := path.NewSharder(path.WithSHA1PrefixLength(c.Cache.ShardingPrefixLen))
	crawler, err := c.NewProtocolCrawler(ctx, op, fv)
	if err != nil {
		return err
	}
	return operations.RunCrawl(ctx, crawler,
		func(ctx context.Context, object content.Object[protocolsiosdk.ProtocolPayload, operations.Response]) error {
			return handleCrawledObject(ctx, fv.Save, sharder, cachePath, op, object)
		})

}

func (c *Command) Get(ctx context.Context, fv *GetFlags, args []string) error {
	opts, err := c.Config.OptionsForEndpoint()
	if err != nil {
		return err
	}
	ep := operations.NewEndpoint[protocolsiosdk.ProtocolPayload](opts...)
	for _, id := range args {
		u := fmt.Sprintf("%v/%v", protocolsiosdk.GetProtocolV4Endpoint, id)
		obj, _, _, err := ep.Get(ctx, u)
		if err != nil {
			return err
		}
		fmt.Printf("%v\n", obj)
	}
	return nil
}

func (c *Command) ScanDownloaded(ctx context.Context, root string, fv *ScanFlags) error {
	tpl, err := template.New("protocolsio").Parse(fv.Template)
	if err != nil {
		return fmt.Errorf("failed to parse template: %q: %v", fv.Template, err)
	}
	cp := filepath.Join(root, c.Cache.Prefix)
	err = filepath.Walk(cp, func(path string, info os.FileInfo, err error) error {
		if errors.Is(err, os.ErrNotExist) {
			return err
		}
		if path == c.Cache.Checkpoint {
			return filepath.SkipDir
		}
		if !info.Mode().IsRegular() {
			return nil
		}
		var obj content.Object[protocolsiosdk.ProtocolPayload, operations.Response]
		if err := obj.ReadObjectBinary(path); err != nil {
			return err
		}
		if err := tpl.Execute(os.Stdout, obj.Value.Protocol); err != nil {
			return err
		}
		fmt.Printf("\n")
		return nil
	})
	return err
}
