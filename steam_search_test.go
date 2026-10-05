package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
)

type steamSearchTransport func(*http.Request) (*http.Response, error)

func (f steamSearchTransport) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }

func TestSteamSearch(t *testing.T) {
	items := `{"id":1,"name":"Bundle","type":"sub"},{"id":0,"name":"Invalid","type":"app"},`
	for i := 1; i <= 12; i++ {
		items += fmt.Sprintf(`{"id":%d,"name":"Game %d","type":"app"},`, i, i)
	}
	items += `{"id":1,"name":"Duplicate","type":"app"}`
	client := &http.Client{Transport: steamSearchTransport(func(req *http.Request) (*http.Response, error) {
		if req.URL.Query().Get("term") != "a & b" {
			t.Fatalf("incorrect search encoding: %s", req.URL)
		}
		if _, ok := req.Context().Deadline(); !ok {
			t.Fatal("missing timeout")
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"items":[` + items + `]}`)), Header: make(http.Header)}, nil
	})}
	games, err := searchSteamGames(context.Background(), " a & b ", client)
	if err != nil || len(games) != 10 || games[0].ID != 1 || games[9].ID != 10 {
		t.Fatalf("unexpected results: %+v, %v", games, err)
	}
	games, err = searchSteamGames(context.Background(), " ", client)
	if err != nil || len(games) != 0 {
		t.Fatal("empty query was not skipped")
	}
}

func TestSteamSearchFailure(t *testing.T) {
	for _, status := range []int{200, 429} {
		client := &http.Client{Transport: steamSearchTransport(func(*http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: status, Body: io.NopCloser(strings.NewReader("invalid JSON")), Header: make(http.Header)}, nil
		})}
		if _, err := searchSteamGames(context.Background(), "a", client); err == nil {
			t.Fatalf("accepted bad response with status %d", status)
		}
	}
}
