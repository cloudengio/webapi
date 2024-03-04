// Copyright 2023 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

// Package benchlingcmd provides support for building command line tools
// that access benchling.com
package benchlingcmd

import (
	"context"
	"fmt"
	"log"
	"time"

	"cloudeng.io/errors"
	"cloudeng.io/file/checkpoint"
	"cloudeng.io/file/content"
	"cloudeng.io/file/content/stores"
	"cloudeng.io/path"
	"cloudeng.io/sync/errgroup"
	"cloudeng.io/webapi/benchling"
	"cloudeng.io/webapi/benchling/benchlingsdk"
	"cloudeng.io/webapi/operations"
	"cloudeng.io/webapi/operations/apicrawlcmd"
	"cloudeng.io/webapi/operations/apitokens"
	"gopkg.in/yaml.v3"
)

type GetFlags struct{}

type CrawlFlags struct{}

type IndexFlags struct{}

// Çommand implements the command line operations available for protocols.io.
type Command struct {
	token     *apitokens.T
	config    apicrawlcmd.Crawl[Service]
	chkpt     checkpoint.Operation
	cfs       operations.FS
	cacheRoot string
}

// NewCommand returns a new Command instance for the specified API crawl
// with API authentication information read from the specified file or
// from the context.
func NewCommand(crawl apicrawlcmd.Crawl[yaml.Node], cfs operations.FS, chkpt checkpoint.Operation, cacheRoot string, token *apitokens.T) (*Command, error) {
	c := &Command{cfs: cfs, cacheRoot: cacheRoot, chkpt: chkpt, token: token}
	err := apicrawlcmd.ParseCrawlConfig(crawl, &c.config)
	if err != nil {
		return nil, err
	}
	return c, nil
}

func (c *Command) Crawl(ctx context.Context, _ CrawlFlags, entities ...string) error {
	opts, err := OptionsForEndpoint(c.config, c.token)
	if err != nil {
		return err
	}
	_, downloadsPath, checkpointPath := c.config.Cache.AbsolutePaths(c.cfs, c.cacheRoot)
	if err := c.config.Cache.PrepareDownloads(ctx, c.cfs, downloadsPath); err != nil {
		return err
	}
	if err := c.config.Cache.PrepareCheckpoint(ctx, c.chkpt, checkpointPath); err != nil {
		return err
	}

	state, err := loadCheckpoint(ctx, c.chkpt)
	if err != nil {
		return err
	}
	log.Printf("checkpoint: user date: %v, entry date: %v", state.UsersDate, state.EntriesDate)

	ch := make(chan any, 100)

	var crawlGroup errgroup.T
	var errs errors.M
	crawlGroup.Go(func() error {
		return c.crawlSaver(ctx, state, downloadsPath, ch)
	})

	var entityGroup errgroup.T
	for _, entity := range entities {
		entity := entity
		entityGroup.Go(func() error {
			err := c.crawlEntity(ctx, state, entity, ch, opts)
			log.Printf("completed crawl of %v: %v", entity, err)
			return err
		})
	}
	errs.Append(entityGroup.Wait())
	close(ch)
	errs.Append(crawlGroup.Wait())
	errs.Append(c.chkpt.Compact(ctx, ""))
	return errs.Err()
}

func (c *Command) crawlSaver(ctx context.Context, state Checkpoint, downloadsPath string, ch <-chan any) error {
	sharder := path.NewSharder(path.WithSHA1PrefixLength(c.config.Cache.ShardingPrefixLen))
	var nUsers, nEntries, nFolders, nProjects int
	var written int64
	defer func() {
		log.Printf("total written: %v (users: %v, entries %v, folders %v, projects %v)", written, nUsers, nEntries, nFolders, nProjects)
	}()
	concurrency := c.config.Cache.Concurrency
	for {
		start := time.Now()
		var entity any
		var ok bool
		select {
		case <-ctx.Done():
			return ctx.Err()
		case entity, ok = <-ch:
			if !ok {
				return nil
			}
		}
		saveStart := time.Now()
		var err error
		switch v := entity.(type) {
		case benchling.Users:
			nUsers += len(v.Users)
			err = save(ctx, c.cfs, downloadsPath, concurrency, sharder, v.Users)
		case benchling.Entries:
			nEntries += len(v.Entries)
			err = save(ctx, c.cfs, downloadsPath, concurrency, sharder, v.Entries)
			state.EntriesDate = *(v.Entries[len(v.Entries)-1].ModifiedAt)
			state.UsersDate = time.Now().Format(time.RFC3339)
			if err := saveCheckpoint(ctx, c.chkpt, state); err != nil {
				log.Printf("failed to save checkpoint: %v", err)
				return err
			}
		case benchling.Folders:
			nFolders += len(v.Folders)
			err = save(ctx, c.cfs, downloadsPath, concurrency, sharder, v.Folders)
		case benchling.Projects:
			nProjects += len(v.Projects)
			err = save(ctx, c.cfs, downloadsPath, concurrency, sharder, v.Projects)
		}
		total := nUsers + nEntries + nFolders + nProjects
		log.Printf("written: %v (users: %v, entries %v, folders %v, projects %v) crawl: %v, save: %v", total, nUsers, nEntries, nFolders, nProjects, saveStart.Sub(start), time.Since(saveStart))
		if err != nil {
			return err
		}
	}
}

func (c *Command) crawlEntity(ctx context.Context, state Checkpoint, entity string, ch chan<- any, opts []operations.Option) error {
	switch entity {
	case "users":
		params := c.config.Service.ListUsersConfig()
		from := "> " + state.UsersDate
		params.ModifiedAt = &from
		cr := &crawler[benchling.Users, *benchlingsdk.ListUsersParams]{
			serviceURL: c.config.Service.ServiceURL,
			params:     params,
		}
		return cr.run(ctx, ch, opts)
	case "entries":
		params := c.config.Service.ListEntriesConfig()
		from := "> " + state.EntriesDate
		params.ModifiedAt = &from
		cr := &crawler[benchling.Entries, *benchlingsdk.ListEntriesParams]{
			serviceURL: c.config.Service.ServiceURL,
			params:     params,
		}
		return cr.run(ctx, ch, opts)
	case "folders":
		params := c.config.Service.ListFoldersConfig()
		cr := &crawler[benchling.Folders, *benchlingsdk.ListFoldersParams]{
			serviceURL: c.config.Service.ServiceURL,
			params:     params,
		}
		return cr.run(ctx, ch, opts)
	case "projects":
		params := c.config.Service.ListProjectsConfig()
		cr := &crawler[benchling.Projects, *benchlingsdk.ListProjectsParams]{
			serviceURL: c.config.Service.ServiceURL,
			params:     params,
		}
		return cr.run(ctx, ch, opts)
	default:
		return fmt.Errorf("unknown entity %v", entity)
	}
}

type crawler[ScannerT benchling.Scanners, ParamsT benchling.Params] struct {
	serviceURL string
	params     ParamsT
}

func (c *crawler[ScannerT, ParamsT]) run(ctx context.Context, ch chan<- any, opts []operations.Option) error {
	sc := benchling.NewScanner[ScannerT](ctx, c.serviceURL, c.params, opts...)
	for sc.Scan(ctx) {
		u := sc.Response()
		select {
		case <-ctx.Done():
			return ctx.Err()
		case ch <- u:
		}
	}
	return sc.Err()
}

func save[ObjectT benchling.Objects](ctx context.Context, fs content.FS, root string, concurrency int, sharder path.Sharder, obj []ObjectT) error {
	store := stores.New(fs, concurrency)
	for _, o := range obj {
		id := benchling.ObjectID(o)
		obj := content.Object[ObjectT, *operations.Response]{
			Type:     benchling.ContentType(o),
			Value:    o,
			Response: &operations.Response{},
		}
		prefix, suffix := sharder.Assign(fmt.Sprintf("%v", id))
		prefix = store.FS().Join(root, prefix)
		if err := obj.Store(ctx, store, prefix, suffix, content.JSONObjectEncoding, content.GOBObjectEncoding); err != nil {
			fmt.Printf("failed to write object id %v: %v %v: %v\n", id, prefix, suffix, err)
		}
	}
	return store.Finish(ctx)
}

// CreateIndexableDocuments constructs the documents to be indexed from the
// various objects crawled from the benchling.com API.
func (c *Command) CreateIndexableDocuments(ctx context.Context, _ IndexFlags) error {
	_, downloads, _ := c.config.Cache.AbsolutePaths(c.cfs, c.cacheRoot)
	sharder := path.NewSharder(path.WithSHA1PrefixLength(c.config.Cache.ShardingPrefixLen))
	nd := benchling.NewDocumentIndexer(c.cfs, downloads, sharder)
	return nd.Index(ctx)
}
