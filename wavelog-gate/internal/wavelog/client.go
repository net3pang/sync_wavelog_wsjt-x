package wavelog

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type Client struct {
	URL       string
	APIKey    string
	StationID string
	hc        *http.Client
}

func New(url, key, sid string) *Client {
	return &Client{
		URL:       strings.TrimRight(url, "/"),
		APIKey:    key,
		StationID: sid,
		hc:        &http.Client{Timeout: 30 * time.Second},
	}
}

// ---- Station profiles (GET /api/station_info/<key>) ----

type StationProfile struct {
	ID   string `json:"station_id"`
	Name string `json:"station_profile_name"`
}

func (c *Client) GetStationProfiles() ([]StationProfile, error) {
	url := c.URL + "/api/station_info/" + c.APIKey
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	resp, err := c.hc.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read: %w", err)
	}
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}
	var raw []json.RawMessage
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("invalid JSON: %w", err)
	}
	var out []StationProfile
	for _, item := range raw {
		var sp StationProfile
		if err := json.Unmarshal(item, &sp); err != nil {
			continue
		}
		if sp.ID != "" {
			out = append(out, sp)
		}
	}
	return out, nil
}

// ---- Forward QSO (POST /api/qso) ----

func (c *Client) ForwardQSO(adifStr string) error {
	payload := map[string]interface{}{
		"key":                c.APIKey,
		"station_profile_id": c.StationID,
		"type":               "adif",
		"string":             adifStr,
	}
	return c.post(c.URL+"/api/qso", payload)
}

// ---- Pull contacts (POST /api/get_contacts_adif) ----

type getContactsResponse struct {
	Status    string                  `json:"status"`
	Message   string                  `json:"message"`
	LastID    string                  `json:"lastfetchedid"`
	Count     int                     `json:"exported_records"`
	QSOs      []map[string]interface{} `json:"qsos"`
}

func (c *Client) GetContacts() ([]map[string]interface{}, error) {
	payload := map[string]interface{}{
		"key":           c.APIKey,
		"station_id":    c.StationID,
		"fetchfromid":   0,
		"output_format": "json",
		"fields":        []string{"CALL", "BAND", "MODE", "QSO_DATE", "TIME_ON", "RST_RCVD", "RST_SENT"},
	}
	body, err := c.doPost(c.URL+"/api/get_contacts_adif", payload)
	if err != nil {
		return nil, err
	}
	var resp getContactsResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("invalid JSON: %w", err)
	}
	if resp.Status != "successful" {
		return nil, fmt.Errorf("API error: %s", resp.Message)
	}
	return resp.QSOs, nil
}

// ---- HTTP helpers ----

func (c *Client) post(url string, payload interface{}) error {
	_, err := c.doPost(url, payload)
	return err
}

func (c *Client) doPost(url string, payload interface{}) ([]byte, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal: %w", err)
	}
	req, err := http.NewRequest("POST", url, bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.hc.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read: %w", err)
	}
	if resp.StatusCode != 200 && resp.StatusCode != 201 {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}
	return body, nil
}
