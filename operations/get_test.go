// Copyright 2023 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package operations_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"reflect"
	"testing"
	"time"

	"cloudeng.io/webapi/operations"
	"cloudeng.io/webapi/webapitestutil"
)

type example struct {
	Name  string
	Value int
}

func TestEcho(t *testing.T) {
	ctx := context.Background()

	eg := example{"foo", 42}
	handler := webapitestutil.NewEchoHandler(&eg)
	srv := webapitestutil.NewServer(handler)
	defer srv.Close()

	client := operations.NewEndpoint[example](srv.URL)

	egr, body, err := client.Get(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := egr, eg; !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
	data, err := json.Marshal(eg)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := body, data; !bytes.Equal(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestRequestsPerMinute(t *testing.T) {
	ctx := context.Background()
	timestamps := []time.Time{}
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		timestamps = append(timestamps, time.Now())
		enc := json.NewEncoder(w)
		enc.Encode(len(timestamps))
	})

	srv := webapitestutil.NewServer(handler)
	defer srv.Close()

	requestsPerMinute := 360
	client := operations.NewEndpoint[int](srv.URL, operations.WithRequestsPerMinute(requestsPerMinute))
	nTimestamps := 5
	for i := 0; i < nTimestamps; i++ {
		n, _, err := client.Get(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if got, want := n, i+1; got != want {
			t.Errorf("got %v, want %v", got, want)
		}
	}

	if got, want := len(timestamps), nTimestamps; got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	expectedDelay := time.Minute / time.Duration(requestsPerMinute)
	for i := 0; i < nTimestamps-1; i++ {
		elapsed := timestamps[i+1].Sub(timestamps[i])
		if got, want := elapsed, expectedDelay; got < (want-want/2) || got > (want+want/2) {
			t.Errorf("got %v, want %v..%v", got, (want - want/2), want+want/2)
		}
	}
}

func TestBackoff(t *testing.T) {
	ctx := context.Background()
	numRetries := 2
	handler := webapitestutil.NewRetryHandler(numRetries)
	srv := webapitestutil.NewServer(handler)
	defer srv.Close()

	client := operations.NewEndpoint[int](srv.URL, operations.WithBackoffParameters(nil, http.StatusTooManyRequests, time.Millisecond, 2))
	n, _, err := client.Get(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := n, numRetries; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
}

type authorizer struct{}

func (a *authorizer) WithAuthorization(ctx context.Context, req *http.Request) error {
	req.Header.Add("something", "secret")
	return nil
}

func TestAuth(t *testing.T) {
	ctx := context.Background()
	handler := webapitestutil.NewHeaderEchoHandler()

	srv := webapitestutil.NewServer(handler)
	defer srv.Close()
	client := operations.NewEndpoint[map[string][]string](srv.URL,
		operations.WithAuth(&authorizer{}))
	headers, _, err := client.Get(ctx)
	if err != nil {
		t.Fatal(err)
	}

	want := map[string][]string{
		"Accept-Encoding": {"gzip"},
		"User-Agent":      {"Go-http-client/1.1"},
		"Something":       {"secret"},
	}

	if got := headers; !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}