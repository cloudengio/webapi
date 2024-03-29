// Copyright 2023 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package benchlingcmd

import (
	"cloudeng.io/net/ratecontrol"
	"cloudeng.io/webapi/clients/benchling"
	"cloudeng.io/webapi/clients/benchling/benchlingsdk"
	"cloudeng.io/webapi/operations"
	"cloudeng.io/webapi/operations/apicrawlcmd"
	"cloudeng.io/webapi/operations/apitokens"
)

type Service struct {
	ServiceURL       string `yaml:"service_url" cmd:"benchling service URL, typically https://altoslabs.benchling.com/api/v2/"`
	UsersPageSize    int    `yaml:"users_page_size" cmd:"number of users in each page of results, typically 50"`
	EntriesPageSize  int    `yaml:"entries_page_size" cmd:"number of entries in each page of results, typically 50"`
	FoldersPageSize  int    `yaml:"folders_page_size" cmd:"number of folders in each page of results, typically 50"`
	ProjectsPageSize int    `yaml:"projects_page_size" cmd:"number of projects in each page of results, typically 50"`
}

type Config apicrawlcmd.Crawl[Service]

var (
	// The sort order is used to enable checkpointing.
	userSortOrder    benchlingsdk.ListUsersParamsSort    = "modifiedAt:asc"
	entrySortOrder   benchlingsdk.ListEntriesParamsSort  = "modifiedAt:asc"
	folderSortOrder  benchlingsdk.ListFoldersParamsSort  = "modifiedAt:asc"
	projectSortOrder benchlingsdk.ListProjectsParamsSort = "modifiedAt:asc"
)

func (s Service) ListUsersConfig() *benchlingsdk.ListUsersParams {
	return &benchlingsdk.ListUsersParams{
		Sort:     &userSortOrder,
		PageSize: &s.UsersPageSize,
	}
}

func (s Service) ListEntriesConfig() *benchlingsdk.ListEntriesParams {
	return &benchlingsdk.ListEntriesParams{
		Sort:     &entrySortOrder,
		PageSize: &s.EntriesPageSize,
	}
}

func (s Service) ListFoldersConfig() *benchlingsdk.ListFoldersParams {
	return &benchlingsdk.ListFoldersParams{
		Sort:     &folderSortOrder,
		PageSize: &s.FoldersPageSize,
	}
}

func (s Service) ListProjectsConfig() *benchlingsdk.ListProjectsParams {
	return &benchlingsdk.ListProjectsParams{
		Sort:     &projectSortOrder,
		PageSize: &s.ProjectsPageSize,
	}
}

func OptionsForEndpoint(cfg apicrawlcmd.Crawl[Service], token *apitokens.T) ([]operations.Option, error) {
	opts := []operations.Option{}
	if tv := token.Token(); len(tv) > 0 {
		opts = append(opts, operations.WithAuth(benchling.APIToken{Token: string(tv)}))
	}
	rateCfg := cfg.RateControl
	rcopts := []ratecontrol.Option{}
	if rateCfg.Rate.BytesPerTick > 0 {
		rcopts = append(rcopts, ratecontrol.WithBytesPerTick(rateCfg.Rate.Tick, rateCfg.Rate.BytesPerTick))
	}
	if rateCfg.Rate.RequestsPerTick > 0 {
		rcopts = append(rcopts, ratecontrol.WithRequestsPerTick(rateCfg.Rate.Tick, rateCfg.Rate.RequestsPerTick))
	}
	if rateCfg.ExponentialBackoff.InitialDelay > 0 {
		rcopts = append(rcopts,
			ratecontrol.WithCustomBackoff(
				func() ratecontrol.Backoff {
					return benchling.NewBackoff(rateCfg.ExponentialBackoff.InitialDelay, rateCfg.ExponentialBackoff.Steps)
				}))
	}
	rc := ratecontrol.New(rcopts...)
	opts = append(opts, operations.WithRateController(rc, cfg.RateControl.ExponentialBackoff.StatusCodes...))
	return opts, nil
}
