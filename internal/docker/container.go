package docker

import (
	"errors"
	"strings"
)

// errNoID guards the per-container operations against empty identifiers.
var errNoID = errors.New("container id is required")

// StartContainer starts a single stopped container by ID or name.
func StartContainer(id string) error {
	if strings.TrimSpace(id) == "" {
		return errNoID
	}
	_, err := run("", "start", id)
	return err
}

// StopContainer stops a single running container by ID or name.
func StopContainer(id string) error {
	if strings.TrimSpace(id) == "" {
		return errNoID
	}
	_, err := run("", "stop", id)
	return err
}

// RemoveContainer force-removes a single container by ID or name.
func RemoveContainer(id string) error {
	if strings.TrimSpace(id) == "" {
		return errNoID
	}
	_, err := run("", "rm", "-f", id)
	return err
}
