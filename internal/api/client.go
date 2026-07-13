package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/TrustEdgeOrg/TrustEdge-Agent/internal/codec"
	"github.com/TrustEdgeOrg/TrustEdge-Agent/internal/models"
)

var ErrUnauthorized = errors.New("unauthorized")

type Client struct {
	BaseURL     string
	EnrollToken string
	DeviceToken string
	HTTP        *http.Client
}

func New(baseURL, enrollToken, deviceToken string) *Client {
	return &Client{
		BaseURL:     baseURL,
		EnrollToken: enrollToken,
		DeviceToken: deviceToken,
		HTTP:        &http.Client{Timeout: 30 * time.Second},
	}
}

func (c *Client) SetDeviceToken(token string) {
	c.DeviceToken = token
}

func (c *Client) Register(req models.RegisterRequest) (*models.RegisterResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	httpReq, err := http.NewRequest(http.MethodPost, c.BaseURL+"/v1/register", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if c.EnrollToken != "" {
		httpReq.Header.Set("Authorization", "Bearer "+c.EnrollToken)
	}
	resp, err := c.HTTP.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode == http.StatusUnauthorized {
		return nil, ErrUnauthorized
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("register: %s: %s", resp.Status, string(data))
	}
	var out models.RegisterResponse
	if err := json.Unmarshal(data, &out); err != nil {
		return nil, err
	}
	c.DeviceToken = out.DeviceToken
	return &out, nil
}

func (c *Client) PostEvent(ev models.Event) error {
	return c.PostEvents([]models.Event{ev})
}

func (c *Client) PostEvents(events []models.Event) error {
	if len(events) == 0 {
		return nil
	}
	if len(events) == 1 {
		body, err := json.Marshal(events[0])
		if err != nil {
			return err
		}
		return c.postEventsBody(body)
	}
	body, err := json.Marshal(models.EventBatch{Events: events})
	if err != nil {
		return err
	}
	return c.postEventsBody(body)
}

func (c *Client) postEventsBody(body []byte) error {
	payload, compressed, err := codec.MaybeCompress(body)
	if err != nil {
		return err
	}
	httpReq, err := http.NewRequest(http.MethodPost, c.BaseURL+"/v1/events", bytes.NewReader(payload))
	if err != nil {
		return err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if compressed {
		httpReq.Header.Set("Content-Encoding", codec.ContentEncoding)
	}
	if c.DeviceToken != "" {
		httpReq.Header.Set("Authorization", "Bearer "+c.DeviceToken)
	}
	resp, err := c.HTTP.Do(httpReq)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode == http.StatusUnauthorized {
		return ErrUnauthorized
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("events: %s: %s", resp.Status, string(data))
	}
	return nil
}
