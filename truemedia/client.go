package truemedia

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

type Client struct {
	baseURL    string
	apiKey     string
	HTTPClient *http.Client
}

func NewClient(apiKey string, baseURL url.URL) *Client {
	return &Client{
		apiKey:     apiKey,
		baseURL:    baseURL.String(),
		HTTPClient: http.DefaultClient,
	}
}

func (c Client) ResolveMedia(url string) (*ResolveMediaResponse, error) {
	reqBody, err := json.Marshal(ResolveMediaRequest{PostURL: url})
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("%s/resolve-media", c.baseURL), bytes.NewReader(reqBody))
	if err != nil {
		return nil, err
	}
	req.Header.Add("X-API-KEY", c.apiKey)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var rmr ResolveMediaResponse
	if err = json.Unmarshal(respBody, &rmr); err != nil {
		return nil, err
	}

	return &rmr, nil
}

func (c Client) GetResults(mediaID string) (*GetResultResponse, error) {
	url, err := url.Parse(c.baseURL + "/get-results")
	if err != nil {
		return nil, err
	}
	q := url.Query()
	q.Add("id", mediaID)
	url.RawQuery = q.Encode()

	req, err := http.NewRequest(http.MethodGet, url.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Add("X-API-KEY", c.apiKey)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var grr GetResultResponse
	if err = json.Unmarshal(body, &grr); err != nil {
		return nil, err
	}

	return &grr, nil
}
