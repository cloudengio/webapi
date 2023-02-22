// Copyright 2023 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package operations_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"testing"

	"cloudeng.io/errors"
	"cloudeng.io/webapi/operations"
	"cloudeng.io/webapi/webapitestutil"
)

type Object struct {
	ID string `json:"id"`
}

type fetcher struct {
	url string
	ep  *operations.Endpoint[Object]
}

func (f *fetcher) Fetch(ctx context.Context, page webapitestutil.Paginated, ch chan<- operations.Crawled[Object]) error {
	fmt.Printf("fetching for %v\n", page)

	obj, _, err := f.ep.Get(ctx, fmt.Sprintf("%s/get?id=%v", f.url, page.Payload))
	if err != nil {
		return err
	}
	ch <- operations.Crawled[Object]{Object: obj}
	return nil
}

func TestCrawler(t *testing.T) {
	ctx := context.Background()

	mux := http.NewServeMux()
	mux.Handle("/list", &webapitestutil.PaginatedHandler{
		Last: 10,
	})
	mux.HandleFunc("/get", func(w http.ResponseWriter, r *http.Request) {
		enc := json.NewEncoder(w)
		id := r.URL.Query().Get("id")
		if err := enc.Encode(Object{ID: id}); err != nil {
			t.Fatal(err)
		}
	})

	srv := webapitestutil.NewServer(mux)
	defer srv.Close()
	paginator := &paginator{url: srv.URL + "/list"}
	scanner := operations.NewScanner[webapitestutil.Paginated](paginator)
	fetcher := &fetcher{url: srv.URL, ep: operations.NewEndpoint[Object]()}

	cr := operations.NewCrawler[webapitestutil.Paginated, Object](scanner, fetcher)
	ch := make(chan operations.Crawled[Object])

	var wg sync.WaitGroup
	wg.Add(1)
	errs := errors.M{}
	go func() {
		errs.Append(cr.Run(ctx, ch))
		wg.Done()
	}()

	id := 1
	for crawled := range ch {
		if got, want := crawled.Object.ID, fmt.Sprintf("%v", id); got != want {
			t.Errorf("got %v, want %v", got, want)
		}
		id++
	}

	wg.Wait()
	if err := errs.Err(); err != nil {
		t.Fatal(err)
	}
}
