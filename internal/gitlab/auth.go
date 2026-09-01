// Package gitlab handles GitLab OAuth device flow and account lookup for
// Settings → Accounts.
package gitlab

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

// Scopes cover git push/pull and release publishing on gitlab.com.
const Scopes = "read_user api read_repository write_repository"

// ErrAuthPending means the user has not finished authorizing yet.
var ErrAuthPending = errors.New("authorization pending")

// ErrSlowDown means GitLab wants a longer gap between poll attempts.
var ErrSlowDown = errors.New("slow down")

var (
	deviceAuthURL = "https://gitlab.com/oauth/authorize_device"
	tokenURL      = "https://gitlab.com/oauth/token"
	userURL       = "https://gitlab.com/api/v4/user"
)

// DeviceCode is what the UI shows the user during device authorization.
type DeviceCode struct {
	DeviceCode               string `json:"deviceCode"`
	UserCode                 string `json:"userCode"`
	VerificationURI          string `json:"verificationUri"`
	VerificationURIComplete  string `json:"verificationUriComplete,omitempty"`
	ExpiresIn                int    `json:"expiresIn"`
	Interval                 int    `json:"interval"`
}

// User is the authenticated account behind a token.
type User struct {
	Login      string `json:"login"`
	Name       string `json:"name"`
	AvatarURL  string `json:"avatarUrl"`
	Bio        string `json:"bio,omitempty"`
	Location   string `json:"location,omitempty"`
	WebsiteURL string `json:"websiteUrl,omitempty"`
	ProfileURL string `json:"profileUrl,omitempty"`
}

// StartDeviceFlow asks GitLab for a device and user code pair.
func StartDeviceFlow(ctx context.Context, clientID string) (*DeviceCode, error) {
	if strings.TrimSpace(clientID) == "" {
		return nil, errors.New("no GitLab OAuth client ID is configured in this build; paste a personal access token instead")
	}
	form := url.Values{}
	form.Set("client_id", clientID)
	form.Set("scope", Scopes)

	var out struct {
		DeviceCode              string `json:"device_code"`
		UserCode                string `json:"user_code"`
		VerificationURI         string `json:"verification_uri"`
		VerificationURIComplete string `json:"verification_uri_complete"`
		ExpiresIn               int    `json:"expires_in"`
		Interval                int    `json:"interval"`
		Error                   string `json:"error"`
		ErrorDesc               string `json:"error_description"`
	}
	if err := postForm(ctx, deviceAuthURL, form, &out); err != nil {
		return nil, err
	}
	if out.Error != "" {
		return nil, fmt.Errorf("GitLab: %s", firstNonEmpty(out.ErrorDesc, out.Error))
	}
	if out.Interval <= 0 {
		out.Interval = 5
	}
	verificationURI := out.VerificationURI
	if out.VerificationURIComplete != "" {
		verificationURI = out.VerificationURIComplete
	}
	return &DeviceCode{
		DeviceCode:              out.DeviceCode,
		UserCode:                out.UserCode,
		VerificationURI:         verificationURI,
		VerificationURIComplete: out.VerificationURIComplete,
		ExpiresIn:               out.ExpiresIn,
		Interval:                out.Interval,
	}, nil
}

// PollDeviceFlow exchanges a device code for an access token once the user
// has authorized. It returns ErrAuthPending or ErrSlowDown while waiting.
func PollDeviceFlow(ctx context.Context, clientID, deviceCode string) (string, error) {
	form := url.Values{}
	form.Set("client_id", clientID)
	form.Set("device_code", deviceCode)
	form.Set("grant_type", "urn:ietf:params:oauth:grant-type:device_code")

	var out struct {
		AccessToken string `json:"access_token"`
		Error       string `json:"error"`
		ErrorDesc   string `json:"error_description"`
	}
	if err := postForm(ctx, tokenURL, form, &out); err != nil {
		return "", err
	}
	switch out.Error {
	case "":
	case "authorization_pending":
		return "", ErrAuthPending
	case "slow_down":
		return "", ErrSlowDown
	case "expired_token":
		return "", errors.New("the code expired, start again")
	case "access_denied":
		return "", errors.New("authorization was declined")
	default:
		return "", fmt.Errorf("GitLab: %s", firstNonEmpty(out.ErrorDesc, out.Error))
	}
	if out.AccessToken == "" {
		return "", errors.New("GitLab returned no access token")
	}
	return out.AccessToken, nil
}

// Whoami verifies the token and returns the account behind it.
func Whoami(ctx context.Context, token string) (*User, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, userURL, nil)
	if err != nil {
		return nil, fmt.Errorf("building request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+strings.TrimSpace(token))
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", userAgent)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("calling GitLab: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusUnauthorized {
		return nil, errors.New("GitLab rejected the token, reconnect in Settings")
	}
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("GitLab returned %s", resp.Status)
	}

	var out struct {
		Username    string `json:"username"`
		Name        string `json:"name"`
		AvatarURL   string `json:"avatar_url"`
		Bio         string `json:"bio"`
		Location    string `json:"location"`
		WebsiteURL  string `json:"website_url"`
		WebURL      string `json:"web_url"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}
	return &User{
		Login:      out.Username,
		Name:       out.Name,
		AvatarURL:  out.AvatarURL,
		Bio:        out.Bio,
		Location:   out.Location,
		WebsiteURL: out.WebsiteURL,
		ProfileURL: out.WebURL,
	}, nil
}

const userAgent = "JumpStart"

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
		return fmt.Errorf("calling GitLab: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("GitLab returned %s", resp.Status)
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
