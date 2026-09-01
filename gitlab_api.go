package main

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"devdeck/internal/gitlab"
	"devdeck/internal/secrets"
)

// glState holds the in-flight GitLab device authorization.
type glState struct {
	mu       sync.Mutex
	deviceID string
}

// GitLabStatus describes the current GitLab connection for Settings.
type GitLabStatus struct {
	Connected  bool   `json:"connected"`
	Login      string `json:"login"`
	Name       string `json:"name"`
	AvatarURL  string `json:"avatarUrl"`
	Bio        string `json:"bio,omitempty"`
	Company    string `json:"company,omitempty"`
	Location   string `json:"location,omitempty"`
	WebsiteURL string `json:"websiteUrl,omitempty"`
	ProfileURL string `json:"profileUrl,omitempty"`
	DeviceFlow bool   `json:"deviceFlow"`
	Error      string `json:"error,omitempty"`
}

// GitLabGetStatus reports whether a usable GitLab token is stored and who
// it belongs to.
func (a *App) GitLabGetStatus() (*GitLabStatus, error) {
	st := &GitLabStatus{DeviceFlow: strings.TrimSpace(GitLabClientID) != ""}
	if !glHasToken() {
		return st, nil
	}
	ctx, cancel := context.WithTimeout(a.ctx, 20*time.Second)
	defer cancel()

	token, err := glLoadToken()
	if err != nil {
		st.Error = err.Error()
		return st, nil
	}
	user, err := gitlab.Whoami(ctx, token)
	if err != nil {
		st.Error = err.Error()
		return st, nil
	}
	st.Connected = true
	st.Login = user.Login
	st.Name = user.Name
	st.AvatarURL = user.AvatarURL
	st.Bio = user.Bio
	st.Location = user.Location
	st.WebsiteURL = user.WebsiteURL
	st.ProfileURL = user.ProfileURL
	return st, nil
}

// GitLabStartDeviceAuth begins the OAuth device flow and returns the code
// the user enters on gitlab.com.
func (a *App) GitLabStartDeviceAuth() (*gitlab.DeviceCode, error) {
	ctx, cancel := context.WithTimeout(a.ctx, 30*time.Second)
	defer cancel()

	code, err := gitlab.StartDeviceFlow(ctx, GitLabClientID)
	if err != nil {
		return nil, err
	}
	a.gl().mu.Lock()
	a.gl().deviceID = code.DeviceCode
	a.gl().mu.Unlock()
	return code, nil
}

// GitLabPollDeviceAuth checks once whether the user has finished
// authorizing.
func (a *App) GitLabPollDeviceAuth() (bool, error) {
	a.gl().mu.Lock()
	deviceID := a.gl().deviceID
	a.gl().mu.Unlock()
	if deviceID == "" {
		return false, errors.New("no authorization in progress")
	}

	ctx, cancel := context.WithTimeout(a.ctx, 30*time.Second)
	defer cancel()

	token, err := gitlab.PollDeviceFlow(ctx, GitLabClientID, deviceID)
	if errors.Is(err, gitlab.ErrAuthPending) || errors.Is(err, gitlab.ErrSlowDown) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if err := glSaveToken(token); err != nil {
		return false, err
	}
	a.gl().mu.Lock()
	a.gl().deviceID = ""
	a.gl().mu.Unlock()
	return true, nil
}

// GitLabSaveToken stores a pasted personal access token.
func (a *App) GitLabSaveToken(token string) error {
	token = strings.TrimSpace(token)
	if token == "" {
		return errors.New("token is empty")
	}
	ctx, cancel := context.WithTimeout(a.ctx, 20*time.Second)
	defer cancel()
	if _, err := gitlab.Whoami(ctx, token); err != nil {
		return err
	}
	return glSaveToken(token)
}

// GitLabDisconnect removes the stored GitLab token.
func (a *App) GitLabDisconnect() error {
	return glDeleteToken()
}

func (a *App) gl() *glState {
	a.glOnce.Do(func() {
		a.glShared = &glState{}
	})
	return a.glShared
}

func glHasToken() bool {
	token, err := secrets.GetToken(secrets.Service, secrets.KeyGitLabToken)
	return err == nil && token != ""
}

func glLoadToken() (string, error) {
	token, err := secrets.GetToken(secrets.Service, secrets.KeyGitLabToken)
	if err != nil {
		return "", err
	}
	if token == "" {
		return "", fmt.Errorf("not connected to GitLab, connect in Settings")
	}
	return token, nil
}

func glSaveToken(token string) error {
	return secrets.SaveToken(secrets.Service, secrets.KeyGitLabToken, strings.TrimSpace(token))
}

func glDeleteToken() error {
	return secrets.DeleteToken(secrets.Service, secrets.KeyGitLabToken)
}
