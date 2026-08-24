package source

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

const (
	defaultBaseURL = "https://api.football-data.org/v4"
	maxAttempts    = 3
	defaultBackoff = 500 * time.Millisecond
	maxBackoffWait = 15 * time.Second
)

type FootballData struct {
	APIKey     string
	BaseURL    string
	HTTPClient *http.Client
	Backoff    time.Duration
}

func NewFootballData(apiKey string) *FootballData {
	return &FootballData{
		APIKey:     apiKey,
		BaseURL:    defaultBaseURL,
		HTTPClient: http.DefaultClient,
		Backoff:    defaultBackoff,
	}
}

type fdMatchesResponse struct {
	Matches []fdMatch `json:"matches"`
}

type fdMatch struct {
	ID       int64   `json:"id"`
	UTCDate  string  `json:"utcDate"`
	Status   string  `json:"status"`
	Minute   *int    `json:"minute"`
	HomeTeam fdTeam  `json:"homeTeam"`
	AwayTeam fdTeam  `json:"awayTeam"`
	Score    fdScore `json:"score"`
}

type fdTeam struct {
	Name string `json:"name"`
}

type fdScore struct {
	FullTime fdFullTime `json:"fullTime"`
}

type fdFullTime struct {
	Home *int `json:"home"`
	Away *int `json:"away"`
}

func (f *FootballData) LiveMatches(ctx context.Context, leagueCode string) ([]Match, error) {
	endpoint := fmt.Sprintf("%s/matches?status=LIVE&competitions=%s", f.base(), url.QueryEscape(leagueCode))
	var payload fdMatchesResponse
	if err := f.getJSON(ctx, endpoint, &payload); err != nil {
		return nil, fmt.Errorf("live matches %s: %w", leagueCode, err)
	}
	matches := make([]Match, 0, len(payload.Matches))
	for _, m := range payload.Matches {
		matches = append(matches, Match{
			ExternalID: strconv.FormatInt(m.ID, 10),
			HomeTeam:   m.HomeTeam.Name,
			AwayTeam:   m.AwayTeam.Name,
			HomeGoals:  m.Score.FullTime.Home,
			AwayGoals:  m.Score.FullTime.Away,
			Minute:     m.Minute,
			Status:     NormalizeStatus(m.Status),
			UTCDate:    m.UTCDate,
		})
	}
	return matches, nil
}

func (f *FootballData) base() string {
	if f.BaseURL == "" {
		return defaultBaseURL
	}
	return f.BaseURL
}

func (f *FootballData) backoff() time.Duration {
	if f.Backoff <= 0 {
		return defaultBackoff
	}
	return f.Backoff
}

func (f *FootballData) getJSON(ctx context.Context, endpoint string, dst any) error {
	backoff := f.backoff()
	var lastErr error
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		err := f.attempt(ctx, endpoint, dst)
		if err == nil {
			return nil
		}
		lastErr = err
		switch e := err.(type) {
		case retryableError:
			if attempt < maxAttempts {
				if serr := sleepCtx(ctx, backoff); serr != nil {
					return serr
				}
				backoff *= 2
			}
		case retryAfterError:
			delay := max(e.wait, backoff)
			if attempt < maxAttempts {
				if serr := sleepCtx(ctx, min(delay, maxBackoffWait)); serr != nil {
					return serr
				}
				backoff *= 2
			}
		default:
			return err
		}
	}
	return fmt.Errorf("after %d attempts: %w", maxAttempts, lastErr)
}

func (f *FootballData) attempt(ctx context.Context, endpoint string, dst any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("X-Auth-Token", f.APIKey)
	client := f.HTTPClient
	if client == nil {
		client = http.DefaultClient
	}
	resp, err := client.Do(req)
	if err != nil {
		return retryableError{fmt.Errorf("network: %w", err)}
	}
	defer resp.Body.Close()
	switch {
	case resp.StatusCode == http.StatusOK:
		if err := json.NewDecoder(resp.Body).Decode(dst); err != nil {
			return fmt.Errorf("decode response: %w", err)
		}
		return nil
	case resp.StatusCode == http.StatusTooManyRequests:
		return retryAfterError{wait: parseRetryAfter(resp.Header.Get("Retry-After"))}
	case resp.StatusCode >= 500:
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return retryableError{fmt.Errorf("server error %d: %s", resp.StatusCode, body)}
	default:
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("unexpected status %d: %s", resp.StatusCode, body)
	}
}

func parseRetryAfter(v string) time.Duration {
	secs, err := strconv.Atoi(v)
	if err != nil || secs < 0 {
		return 0
	}
	return time.Duration(secs) * time.Second
}

func sleepCtx(ctx context.Context, d time.Duration) error {
	if d <= 0 {
		return ctx.Err()
	}
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}

type retryableError struct{ err error }

func (e retryableError) Error() string { return e.err.Error() }
func (e retryableError) Unwrap() error { return e.err }

type retryAfterError struct {
	wait time.Duration
}

func (e retryAfterError) Error() string {
	return fmt.Sprintf("rate limited (429), retry after %s", e.wait)
}
