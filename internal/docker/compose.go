package docker

import "errors"

// errNoCompose is returned when a compose action is requested for a directory
// that has no recognised compose file.
var errNoCompose = errors.New("no docker compose file found in project root")

// Container is a compose service container as shown in the Containers tab.
type Container struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Service string `json:"service"`
	Image   string `json:"image"`
	State   string `json:"state"`  // running | exited | created | ...
	Status  string `json:"status"` // human status, e.g. "Up 3 minutes"
	Ports   string `json:"ports"`
}

// Image is a compose service image.
type Image struct {
	ID         string `json:"id"`
	Repository string `json:"repository"`
	Tag        string `json:"tag"`
	Size       string `json:"size"`
}

// Volume is a named Docker volume belonging to the compose project.
type Volume struct {
	Name   string `json:"name"`
	Driver string `json:"driver"`
	Scope  string `json:"scope"`
}

// ComposeUp starts the project's stack in detached mode.
func ComposeUp(root string) error {
	if composeFile(root) == "" {
		return errNoCompose
	}
	_, err := run(root, "compose", "up", "-d")
	return err
}

// ComposeDown stops and removes the project's stack (containers and networks).
func ComposeDown(root string) error {
	if composeFile(root) == "" {
		return errNoCompose
	}
	_, err := run(root, "compose", "down")
	return err
}

// Containers lists all containers (running and stopped) for the compose
// project rooted at root.
func Containers(root string) ([]Container, error) {
	if composeFile(root) == "" {
		return nil, errNoCompose
	}
	rows, err := runJSONLines(root, "compose", "ps", "-a", "--format", "json")
	if err != nil {
		return nil, err
	}
	out := make([]Container, 0, len(rows))
	for _, r := range rows {
		out = append(out, Container{
			ID:      firstStr(r, "ID", "Id"),
			Name:    firstStr(r, "Name", "Names"),
			Service: str(r, "Service"),
			Image:   str(r, "Image"),
			State:   str(r, "State"),
			Status:  str(r, "Status"),
			Ports:   firstStr(r, "Ports", "Publishers"),
		})
	}
	return out, nil
}

// Images lists the images backing the compose project's services.
func Images(root string) ([]Image, error) {
	if composeFile(root) == "" {
		return nil, errNoCompose
	}
	rows, err := runJSONLines(root, "compose", "images", "--format", "json")
	if err != nil {
		return nil, err
	}
	out := make([]Image, 0, len(rows))
	for _, r := range rows {
		out = append(out, Image{
			ID:         firstStr(r, "ID", "Id"),
			Repository: str(r, "Repository"),
			Tag:        str(r, "Tag"),
			Size:       sizeMB(r, "Size"),
		})
	}
	return out, nil
}

// Volumes lists Docker volumes labelled as belonging to the compose project.
func Volumes(root string) ([]Volume, error) {
	name := projectName(root)
	if name == "" {
		return nil, errNoCompose
	}
	rows, err := runJSONLines(root, "volume", "ls",
		"--filter", "label=com.docker.compose.project="+name,
		"--format", "{{json .}}")
	if err != nil {
		return nil, err
	}
	out := make([]Volume, 0, len(rows))
	for _, r := range rows {
		out = append(out, Volume{
			Name:   str(r, "Name"),
			Driver: str(r, "Driver"),
			Scope:  str(r, "Scope"),
		})
	}
	return out, nil
}
