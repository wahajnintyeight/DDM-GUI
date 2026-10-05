package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type SteamGame struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

func (a *App) SearchSteamGames(term string) ([]SteamGame, error) {
	return searchSteamGames(a.ctx, term, http.DefaultClient)
}

func searchSteamGames(ctx context.Context, term string, client *http.Client) ([]SteamGame, error) {
	term = strings.TrimSpace(term)
	if term == "" {
		return []SteamGame{}, nil
	}
	if len(term) > 200 {
		return nil, errors.New("search must be 200 bytes or fewer")
	}
	ctx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	query := url.Values{"term": {term}, "l": {"english"}, "cc": {"US"}}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://store.steampowered.com/api/storesearch/?"+query.Encode(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	response, err := client.Do(req)
	if err != nil {
		return nil, errors.New("Steam search is unavailable. Try again or enter the app ID manually.")
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Steam search returned HTTP %d. Try again or enter the app ID manually.", response.StatusCode)
	}
	var data struct {
		Items []struct {
			ID   int64  `json:"id"`
			Name string `json:"name"`
			Type string `json:"type"`
		} `json:"items"`
	}
	if err := json.NewDecoder(io.LimitReader(response.Body, 1024*1024)).Decode(&data); err != nil {
		return nil, errors.New("Steam returned an invalid search response. Enter the app ID manually.")
	}
	games := make([]SteamGame, 0, 10)
	seen := make(map[int64]bool)
	for _, item := range data.Items {
		if item.Type != "app" || item.ID <= 0 || strings.TrimSpace(item.Name) == "" || seen[item.ID] {
			continue
		}
		seen[item.ID] = true
		games = append(games, SteamGame{ID: item.ID, Name: item.Name})
		if len(games) == 10 {
			break
		}
	}
	return games, nil
}
