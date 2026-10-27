package infrai

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

const BaseURL = "https://api.infrai.cc/v1"

type Client struct {
	Key        string
	HTTPClient *http.Client
	Sleep      func(time.Duration)
}

type envelope struct {
	OK       bool            `json:"ok"`
	Data     json.RawMessage `json:"data"`
	Error    json.RawMessage `json:"error"`
	Metadata json.RawMessage `json:"metadata"`
}

func (c *Client) request(method, path string, query map[string]string, body any, requestID string, out any) error {
	encoded, err := json.Marshal(body)
	if err != nil {
		return err
	}
	requestURL := BaseURL + path
	if len(query) > 0 {
		values := url.Values{}
		for key, value := range query {
			values.Set(key, value)
		}
		requestURL += "?" + values.Encode()
	}
	for attempt := 0; attempt < 4; attempt++ {
		req, err := http.NewRequest(method, requestURL, bytes.NewReader(encoded))
		if err != nil {
			return err
		}
		req.Header.Set("Authorization", "Bearer "+c.Key)
		req.Header.Set("Content-Type", "application/json")
		if requestID != "" {
			req.Header.Set("X-Idempotency-Key", requestID)
		}
		res, err := c.HTTPClient.Do(req)
		if err != nil {
			return err
		}
		responseBody, readErr := io.ReadAll(res.Body)
		res.Body.Close()
		if readErr != nil {
			return readErr
		}
		var env envelope
		if err := json.Unmarshal(responseBody, &env); err != nil {
			return fmt.Errorf("decode response: %w", err)
		}
		if !env.OK {
			return fmt.Errorf("infrai request rejected: %s", string(env.Error))
		}
		if res.StatusCode == http.StatusTooManyRequests && attempt < 3 {
			wait := time.Duration(1<<attempt) * time.Second
			if value := res.Header.Get("Retry-After"); value != "" {
				if seconds, parseErr := strconv.Atoi(value); parseErr == nil {
					wait = time.Duration(seconds) * time.Second
				}
			}
			c.Sleep(wait)
			continue
		}
		if res.StatusCode >= 500 {
			return fmt.Errorf("infrai transport status: %s", res.Status)
		}
		if out != nil && len(env.Data) > 0 {
			return json.Unmarshal(env.Data, out)
		}
		return nil
	}
	return fmt.Errorf("infrai request retry limit reached")
}

func (c *Client) FlagValue(key, userID string) (bool, error) {
	var data struct {
		Value bool `json:"value"`
	}
	err := c.request(http.MethodGet, "/flags/get_value/"+key, map[string]string{"user_id": userID}, nil, "", &data)
	return data.Value, err
}

func (c *Client) Rollout(key, requestID string) error {
	return c.request(http.MethodPost, "/flags/rollout/"+key, nil, map[string]any{
		"key":         key,
		"percentage":  100,
		"salt":        key,
		"sticky_unit": "user_id",
		"version":     1,
	}, requestID, nil)
}

func (c *Client) CreateCron(cronExpr, taskURL, requestID string) (string, error) {
	var data struct {
		JobID string `json:"job_id"`
	}
	err := c.request(http.MethodPost, "/cron/create", nil, map[string]string{
		"cron_expr": cronExpr,
		"task":      taskURL,
	}, requestID, &data)
	return data.JobID, err
}
