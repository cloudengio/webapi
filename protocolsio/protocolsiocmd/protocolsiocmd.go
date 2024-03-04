// Copyright 2023 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

// Package protocolsiocmd provides support for building command line tools
// that access protocols.io.
package protocolsiocmd

import (
	"context"
	"fmt"
	"log"
	"os"
	"sync"
	"text/template"

	"cloudeng.io/cmdutil/flags"
	"cloudeng.io/errors"
	"cloudeng.io/file/checkpoint"
	"cloudeng.io/file/content"
	"cloudeng.io/file/content/stores"
	"cloudeng.io/file/filewalk"
	"cloudeng.io/path"
	"cloudeng.io/webapi/operations"
	"cloudeng.io/webapi/operations/apicrawlcmd"
	"cloudeng.io/webapi/operations/apitokens"
	"cloudeng.io/webapi/protocolsio/protocolsiosdk"
	"gopkg.in/yaml.v3"
)

type CommonFlags struct {
	ProtocolsConfig string `subcmd:"protocolsio-config,$HOME/.protocolsio.yaml,'protocols.io auth config file'"`
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
	CommonFlags
	Template string `subcmd:"template,'{{.ID}}',template to use for printing fields in the downloaded Protocol objects"`
}

// Çommand implements the command line operations available for protocols.io.
type Command struct {
	token  *apitokens.T
	config apicrawlcmd.Crawl[Service]
	cfs    operations.FS
	chkpt  checkpoint.Operation
}

// NewCommand returns a new Command instance for the specified API crawl
// with API authentication information read from the specified file or
// from the context.
func NewCommand(ctx context.Context, crawl apicrawlcmd.Crawl[yaml.Node], fs operations.FS, chkpt checkpoint.Operation, token *apitokens.T) (*Command, error) {
	c := &Command{cfs: fs, chkpt: chkpt, token: token}
	err := apicrawlcmd.ParseCrawlConfig(crawl, &c.config)
	if err != nil {
		return nil, err
	}
	return c, nil
}

func (c *Command) Crawl(ctx context.Context, fs content.FS, cacheRoot string, fv *CrawlFlags) error {
	_, downloadsPath, chkptPath := c.config.Cache.AbsolutePaths(c.cfs, cacheRoot)
	if err := c.config.Cache.PrepareDownloads(ctx, c.cfs, downloadsPath); err != nil {
		return err
	}
	if err := c.config.Cache.PrepareCheckpoint(ctx, c.chkpt, chkptPath); err != nil {
		return err
	}
	if fv.IgnoreCheckpoint {
		if err := c.chkpt.Clear(ctx); err != nil {
			return err
		}
	}

	sharder := path.NewSharder(path.WithSHA1PrefixLength(c.config.Cache.ShardingPrefixLen))

	crawler, err := NewProtocolCrawler(ctx, c.config, c.cfs, downloadsPath, c.chkpt, fv, c.token)
	if err != nil {
		return err
	}

	var errs errors.M
	err = operations.RunCrawl(ctx, crawler,
		func(ctx context.Context, objects []content.Object[protocolsiosdk.ProtocolPayload, operations.Response]) error {
			return handleCrawledObject(ctx, fv.Save, sharder, fs, downloadsPath, c.chkpt, objects)
		})
	errs.Append(err)
	errs.Append(c.chkpt.Compact(ctx, ""))
	return errs.Err()
}

func handleCrawledObject(ctx context.Context,
	save bool,
	sharder path.Sharder,
	fs content.FS,
	root string,
	chk checkpoint.Operation,
	objs []content.Object[protocolsiosdk.ProtocolPayload, operations.Response]) error {

	store := stores.New(fs, 0)
	for _, obj := range objs {
		if obj.Response.Current != 0 && obj.Response.Total != 0 {
			log.Printf("progress: %v/%v\n", obj.Response.Current, obj.Response.Total)
		}
		if obj.Value.Protocol.ID == 0 {
			// Protocol is up-to-date on disk.
			return nil
		}
		log.Printf("protocol ID: %v\n", obj.Value.Protocol.ID)
		if !save {
			return nil
		}
		// Save the protocol object to disk.
		prefix, suffix := sharder.Assign(fmt.Sprintf("%v", obj.Value.Protocol.ID))
		prefix = store.FS().Join(root, prefix)
		if err := obj.Store(ctx, store, prefix, suffix, content.GOBObjectEncoding, content.GOBObjectEncoding); err != nil {
			return err
		}

		if state := obj.Response.Checkpoint; len(state) > 0 {
			name, err := chk.Checkpoint(ctx, "", state)
			if err != nil {
				log.Printf("failed to save checkpoint: %v: %v\n", name, err)
			} else {
				log.Printf("checkpoint: %v\n", name)
			}
		}
	}
	return store.Finish(ctx)
}

func (c *Command) Get(ctx context.Context, _ *GetFlags, args []string) error {
	opts, err := OptionsForEndpoint(c.config, c.token)
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
	_, downloadsPath, _ := c.config.Cache.AbsolutePaths(c.cfs, root)
	store := stores.New(c.cfs, c.config.Cache.Concurrency)
	var mu sync.Mutex
	err = filewalk.ContentsOnly(ctx, c.cfs, downloadsPath, func(ctx context.Context, prefix string, contents []filewalk.Entry, err error) error {
		if err != nil {
			log.Printf("error: %v: %v", prefix, err)
		}
		names := make([]string, len(contents))
		for i, c := range contents {
			names[i] = c.Name
		}
		return store.ReadV(ctx, prefix, names, func(_ context.Context, _, _ string, _ content.Type, buf []byte, err error) error {
			if err != nil {
				return err
			}
			mu.Lock()
			defer mu.Unlock()
			var obj content.Object[protocolsiosdk.ProtocolPayload, operations.Response]
			if err := obj.Decode(buf); err != nil {
				return err
			}
			if err := tpl.Execute(os.Stdout, obj.Value.Protocol); err != nil {
				return err
			}
			fmt.Printf("\n")
			return nil
		})
	})
	return err
}
