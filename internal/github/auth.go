package github

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Scopes requested for the device flow. "project" covers Projects v2
// read and write; "repo" is needed to open issues and read their state.
const Scopes = "repo project read:org"

// ErrAuthPending means the user has not finished entering the code yet.
var ErrAuthPending = errors.New("authorization pending")

// ErrSlowDown means GitHub wants a longer gap between poll attempts.
var ErrSlowDown = errors.New("slow down")

// DeviceCode is what the UI shows the user: a short code to type at a
// verification URL, plus how long it is good for.
type DeviceCode struct {
	DeviceCode      string `json:"deviceCode"`
	UserCode        string `json:"userCode"`
	VerificationURI string `json:"verificationUri"`
	ExpiresIn       int    `json:"expiresIn"`
	Interval        int    `json:"interval"`
}

// StartDeviceFlow asks GitHub for a device and user code pair.
func StartDeviceFlow(ctx context.Context, clientID string) (*DeviceCode, error) {
	if strings.TrimSpace(clientID) == "" {
		return nil, errors.New("no GitHub OAuth client ID is configured in this build; paste a personal access token instead")
	}
	form := url.Values{}
	form.Set("client_id", clientID)
	form.Set("scope", Scopes)

	var out struct {
		DeviceCode      string `json:"device_code"`
		UserCode        string `json:"user_code"`
		VerificationURI string `json:"verification_uri"`
		ExpiresIn       int    `json:"expires_in"`
		Interval        int    `json:"interval"`
		Error           string `json:"error"`
		ErrorDesc       string `json:"error_description"`
	}
	if err := postForm(ctx, deviceCodeURL, form, &out); err != nil {
		return nil, err
	}
	if out.Error != "" {
		return nil, fmt.Errorf("GitHub: %s", firstNonEmpty(out.ErrorDesc, out.Error))
	}
	if out.Interval <= 0 {
		out.Interval = 5
	}
	return &DeviceCode{
		DeviceCode:      out.DeviceCode,
		UserCode:        out.UserCode,
		VerificationURI: out.VerificationURI,
		ExpiresIn:       out.ExpiresIn,
		Interval:        out.Interval,
	}, nil
}

// PollDeviceFlow makes one attempt to exchange a device code for a token
// set. It returns ErrAuthPending or ErrSlowDown while the user is still
// authorizing, so the caller controls the polling cadence.
//
// The whole set matters, not just the access token: a GitHub App with
// expiring user tokens (the default for new apps) hands back a token
// good for a few hours plus a refresh token good for months, and
// dropping the latter is what forces the user to reconnect by hand
// twice a day.
func PollDeviceFlow(ctx context.Context, clientID, deviceCode string) (*TokenSet, error) {
	form := url.Values{}
	form.Set("client_id", clientID)
	form.Set("device_code", deviceCode)
	form.Set("grant_type", "urn:ietf:params:oauth:grant-type:device_code")

	var out tokenResponse
	if err := postForm(ctx, accessTokenURL, form, &out); err != nil {
		return nil, err
	}
	switch out.Error {
	case "":
	case "authorization_pending":
		return nil, ErrAuthPending
	case "slow_down":
		return nil, ErrSlowDown
	case "expired_token":
		return nil, errors.New("the code expired, start again")
	case "access_denied":
		return nil, errors.New("authorization was declined")
	default:
		return nil, fmt.Errorf("GitHub: %s", firstNonEmpty(out.ErrorDesc, out.Error))
	}
	if out.AccessToken == "" {
		return nil, errors.New("GitHub returned no access token")
	}
	return out.toTokenSet(time.Now()), nil
}

// Viewer is the authenticated account, used to confirm a token works and
// to label the connection in Settings.
type Viewer struct {
	Login      string `json:"login"`
	Name       string `json:"name"`
	AvatarURL  string `json:"avatarUrl"`
	Bio        string `json:"bio,omitempty"`
	Company    string `json:"company,omitempty"`
	Location   string `json:"location,omitempty"`
	WebsiteURL string `json:"websiteUrl,omitempty"`
	ProfileURL string `json:"url,omitempty"`
}

// Whoami verifies the token and returns the account behind it.
func (c *Client) Whoami(ctx context.Context) (*Viewer, error) {
	var out struct {
		Viewer Viewer `json:"viewer"`
	}
	if err := c.Query(ctx, queryViewer, nil, &out); err != nil {
		return nil, err
	}
	return &out.Viewer, nil
}

func postForm(ctx context.Context, endpoint string, form url.Values, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return fmt.Errorf("building request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", userAgent)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("calling GitHub: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("GitHub returned %s", resp.Status)
	}
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return fmt.Errorf("decoding response: %w", err)
	}
	return nil
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}
