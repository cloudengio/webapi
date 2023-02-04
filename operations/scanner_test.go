// Copyright 2023 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package operations_test

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"testing"

	"cloudeng.io/webapi/operations"
	"cloudeng.io/webapi/webapitestutil"
)

type paginator struct {
	url string
}

func (p *paginator) Next(payload webapitestutil.Paginated, resp *http.Response) (string, io.Reader, bool, error) {
	if resp == nil {
		// first time through, return the url and mark as paginated.
		return p.url, nil, false, nil
	}
	nextUrl := fmt.Sprintf(p.url+"?current=%v", payload.Current+1)
	return nextUrl, nil, payload.Current == payload.Last, nil
}

func TestScanner(t *testing.T) {
	ctx := context.Background()
	handler := &webapitestutil.PaginatedHandler{
		Last: 10,
	}
	srv := webapitestutil.NewServer(handler)
	defer srv.Close()
	paginator := &paginator{url: srv.URL}
	scanner := operations.NewScanner[webapitestutil.Paginated](paginator)
	expected := 0
	for scanner.Scan(ctx) {
		r := scanner.Response()
		if got, want := r.Payload, expected+1; got != want {
			t.Errorf("got %v, want %v", got, want)
		}
		if got, want := r.Current, expected; got != want {
			t.Errorf("got %v, want %v", got, want)
		}
		expected++
	}
	if err := scanner.Err(); err != nil {
		t.Fatal(err)
	}
}

type errPaginator struct {
	url      string
	failWhen int
	count    int
}

func (p *errPaginator) Next(payload webapitestutil.Paginated, resp *http.Response) (string, io.Reader, bool, error) {
	if resp == nil {
		if p.failWhen == 0 {
			return "", nil, false, fmt.Errorf("fail immediately")
		}
		return p.url, nil, false, nil
	}
	if p.count == p.failWhen {
		return "", nil, false, fmt.Errorf("fail immediately")
	}
	p.count++
	nextUrl := fmt.Sprintf(p.url+"?current=%v", payload.Current+1)
	return nextUrl, nil, payload.Current == payload.Last, nil
}

func TestScannerErrorImmediately(t *testing.T) {
	ctx := context.Background()
	handler := &webapitestutil.PaginatedHandler{
		Last: 10,
	}
	srv := webapitestutil.NewServer(handler)
	defer srv.Close()
	paginator := &errPaginator{url: srv.URL}
	scanner := operations.NewScanner[webapitestutil.Paginated](paginator)
	for scanner.Scan(ctx) {
		t.Error("expected Scan to return false")
	}
	if err := scanner.Err(); err == nil || err.Error() != "fail immediately" {
		t.Errorf("missing or unexpected error: %v", err)
	}
}

func TestScannerErrorAfterN(t *testing.T) {
	ctx := context.Background()
	handler := &webapitestutil.PaginatedHandler{
		Last: 10,
	}
	srv := webapitestutil.NewServer(handler)
	defer srv.Close()
	paginator := &errPaginator{url: srv.URL, failWhen: 5}
	scanner := operations.NewScanner[webapitestutil.Paginated](paginator)
	count := 0
	for scanner.Scan(ctx) {
		r := scanner.Response()
		if got, want := r.Current, count; got != want {
			t.Errorf("got %v, want %v", got, want)
		}
		count++
	}
	if err := scanner.Err(); err == nil || err.Error() != "fail immediately" {
		t.Errorf("missing or unexpected error: %v", err)
	}
	if got, want := count, paginator.failWhen; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
}