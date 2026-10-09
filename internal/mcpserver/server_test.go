package mcpserver

import (
	"io"
	"net/http"
	"testing"
	"time"

	"devdeck/internal/gitops"
	"devdeck/internal/model"
)

type stubHost struct{}

func (stubHost) ListProjects() ([]model.Project, error)                   { return nil, nil }
func (stubHost) GetProject(string) (*model.Project, error)                { return nil, errNotFound }
func (stubHost) SaveProject(model.Project) error                          { return nil }
func (stubHost) StartProcess(string, string) error                        { return nil }
func (stubHost) StopProcess(string) error                                 { return nil }
func (stubHost) StartAll(string) []string                                 { return nil }
func (stubHost) StopAll(string)                                           {}
func (stubHost) ProcessStatus(string) model.Status                        { return model.Status{} }
func (stubHost) ProcessLogs(string) []string                              { return nil }
func (stubHost) UpdateTasks(string, []model.Task) error                   { return nil }
func (stubHost) GlobalQuarters() []model.QuarterRange                      { return nil }
func (stubHost) GitStatus(string) (*gitops.Status, error)                 { return nil, nil }
func (stubHost) GitDiff(string, string) (*gitops.DiffResult, error)       { return nil, nil }
func (stubHost) GitWorkingChanges(string) ([]gitops.FileChange, error)    { return nil, nil }

var errNotFound = errString("not found")

type errString string

func (e errString) Error() string { return string(e) }

func TestServerAuth(t *testing.T) {
	dir := t.TempDir()
	s := New(stubHost{}, dir, "test")
	cfg := Config{Enabled: true, Port: 18787, Token: "secrettoken123456"}
	if err := s.Apply(cfg); err != nil {
		t.Fatal(err)
	}
	defer s.Stop()
	time.Sleep(50 * time.Millisecond)

	resp, err := http.Get("http://127.0.0.1:18787/mcp")
	if err != nil {
		t.Fatal(err)
	}
	io.Copy(io.Discard, resp.Body)
	resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("want 401 got %d", resp.StatusCode)
	}

	req, err := http.NewRequest(http.MethodGet, "http://127.0.0.1:18787/mcp", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer secrettoken123456")
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	io.Copy(io.Discard, resp.Body)
	resp.Body.Close()
	if resp.StatusCode == http.StatusUnauthorized {
		t.Fatal("authorized request still rejected")
	}
}
