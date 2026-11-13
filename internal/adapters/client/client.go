package client

import (
	"bytes"
	"context"
	"cron/internal/dto"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
)

type Client struct {
	client  *http.Client
	baseUrl string
	token   string
}

type AuthRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type AuthResponse struct {
	AccessToken string `json:"access_token"`
}

func NewClient(baseUrl string) *Client {
	return &Client{
		client: &http.Client{
			Timeout: 5 * time.Minute,
		},
		baseUrl: baseUrl,
	}
}

func (c *Client) Login(ctx context.Context, username, password string) error {
	payload := AuthRequest{
		Username: username,
		Password: password,
	}

	jsonPayload, err := json.Marshal(payload)

	if err != nil {
		return fmt.Errorf("Error on serealice credentials %w", err)
	}

	url := fmt.Sprintf("%s/auth", c.baseUrl)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonPayload))
	if err != nil {
		return fmt.Errorf("Error creating request Login: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("Error sending request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Login failed with status code: %d", resp.StatusCode)
	}
	var respData AuthResponse
	if err := json.NewDecoder(resp.Body).Decode(&respData); err != nil {
		return fmt.Errorf("Error decoding response: %w", err)
	}
	c.token = respData.AccessToken
	return nil
}

func (c *Client) GetSession(ctx context.Context, storeId string) (string, error) {
	url := fmt.Sprintf("%s/status/session?store=%s", c.baseUrl, storeId)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("Error creating request get Session: %w", err)
	}
	req.Header.Set("Authorization", fmt.Sprintf("jwt %s", c.token))
	resp, err := c.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("Error sending request get Session: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("GetSession failed with status code: %d", resp.StatusCode)
	}
	type sessionItem struct {
		Session string `json:"session"`
		Date    string `json:"date"`
	}

	var sessions []sessionItem

	if err := json.NewDecoder(resp.Body).Decode(&sessions); err != nil {
		return "", fmt.Errorf("Error decoding history sessions: %w", err)
	}
	if len(sessions) == 0 {
		return "", fmt.Errorf("Sessions not found for store %s", storeId)
	}
	log.Printf("Last Session: %s (Date: %s)", sessions[0].Session, sessions[0].Date)

	return sessions[0].Session, nil
}

func (c *Client) GetLabels(ctx context.Context, storeId, session string) ([]dto.BatchDataDTO, error) {
	url := fmt.Sprintf("%s/data/labels?store=%s&session=%s", c.baseUrl, storeId, session)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("Error creating request get Labels: %w", err)
	}
	req.Header.Set("Authorization", fmt.Sprintf("jwt %s", c.token))
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("Error sending request get Labels: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GetLabels failed with status code: %d", resp.StatusCode)
	}
	var labels []dto.BatchDataDTO
	if err := json.NewDecoder(resp.Body).Decode(&labels); err != nil {
		return nil, fmt.Errorf("Error on decode labels: %w", err)
	}
	return labels, nil
}
