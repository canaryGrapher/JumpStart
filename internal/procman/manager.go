package procman

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"sort"
	"sync"
	"syscall"
	"time"

	"devdeck/internal/model"
)

// Emitter pushes events to the frontend (wired to Wails EventsEmit).
type Emitter func(event string, data ...interface{})

type runningProc struct {
	cmd       *exec.Cmd
	logs      *logBuf
	ports     map[int]bool
	startedAt time.Time
	exitCode  int
	done      chan struct{}
	mu        sync.Mutex
}

// Manager owns all running subprocesses.
type Manager struct {
	mu    sync.Mutex
	procs map[string]*runningProc
	emit  Emitter
}

func New(emit Emitter) *Manager {
	return &Manager{procs: map[string]*runningProc{}, emit: emit}
}

// Start launches a process in its own process group.
func (m *Manager) Start(p model.Process) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if rp, ok := m.procs[p.ID]; ok {
		select {
		case <-rp.done:
		default:
			return fmt.Errorf("%s is already running", p.Name)
		}
	}

	cmd := exec.Command("/bin/sh", "-c", p.Command)
	cmd.Dir = p.Dir
	cmd.Env = os.Environ()
	for k, v := range p.Env {
		cmd.Env = append(cmd.Env, k+"="+v)
	}
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}
	if err := cmd.Start(); err != nil {
		return err
	}

	rp := &runningProc{
		cmd:       cmd,
		logs:      &logBuf{},
		ports:     map[int]bool{},
		startedAt: time.Now(),
		exitCode:  -1,
		done:      make(chan struct{}),
	}
	m.procs[p.ID] = rp

	go m.pipe(p.ID, rp, stdout)
	go m.pipe(p.ID, rp, stderr)
	go m.pollPorts(p.ID, rp)
	go m.wait(p.ID, rp)
	return nil
}

func (m *Manager) pipe(id string, rp *runningProc, r interface{ Read([]byte) (int, error) }) {
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 64*1024), 1024*1024)
	for sc.Scan() {
		line := sc.Text()
		rp.logs.add(line)
		m.emit("log:"+id, line)
		if ports := portsFromLine(line); len(ports) > 0 {
			m.addPorts(id, rp, ports)
		}
	}
}

func (m *Manager) pollPorts(id string, rp *runningProc) {
	pgid := rp.cmd.Process.Pid
	for i := 0; i < 15; i++ {
		select {
		case <-rp.done:
			return
		case <-time.After(2 * time.Second):
		}
		if ports := portsFromLsof(pgid); len(ports) > 0 {
			m.addPorts(id, rp, ports)
		}
	}
}

func (m *Manager) addPorts(id string, rp *runningProc, ports []int) {
	rp.mu.Lock()
	changed := false
	for _, p := range ports {
		if !rp.ports[p] {
			rp.ports[p] = true
			changed = true
		}
	}
	list := rp.portList()
	rp.mu.Unlock()
	if changed {
		m.emit("ports:"+id, list)
	}
}

func (m *Manager) wait(id string, rp *runningProc) {
	err := rp.cmd.Wait()
	rp.mu.Lock()
	rp.exitCode = 0
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			rp.exitCode = ee.ExitCode()
		} else {
			rp.exitCode = 1
		}
	}
	rp.mu.Unlock()
	close(rp.done)
	m.emit("exit:"+id, rp.exitCode)
}

// Stop terminates the whole process group (SIGTERM, then SIGKILL).
func (m *Manager) Stop(id string) error {
	m.mu.Lock()
	rp, ok := m.procs[id]
	m.mu.Unlock()
	if !ok {
		return nil
	}
	select {
	case <-rp.done:
		return nil
	default:
	}
	pgid := rp.cmd.Process.Pid
	_ = syscall.Kill(-pgid, syscall.SIGTERM)
	select {
	case <-rp.done:
	case <-time.After(5 * time.Second):
		_ = syscall.Kill(-pgid, syscall.SIGKILL)
	}
	return nil
}

// StopAll stops every running process.
func (m *Manager) StopAll() {
	m.mu.Lock()
	ids := make([]string, 0, len(m.procs))
	for id := range m.procs {
		ids = append(ids, id)
	}
	m.mu.Unlock()
	for _, id := range ids {
		_ = m.Stop(id)
	}
}

// RunningPIDs returns procID -> process group ID for running processes.
func (m *Manager) RunningPIDs() map[string]int {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := map[string]int{}
	for id, rp := range m.procs {
		select {
		case <-rp.done:
		default:
			out[id] = rp.cmd.Process.Pid
		}
	}
	return out
}

// Logs returns the buffered log lines for a process.
func (m *Manager) Logs(id string) []string {
	m.mu.Lock()
	rp, ok := m.procs[id]
	m.mu.Unlock()
	if !ok {
		return []string{}
	}
	return rp.logs.all()
}

// Status reports the live state of a process.
func (m *Manager) Status(id string) model.Status {
	m.mu.Lock()
	rp, ok := m.procs[id]
	m.mu.Unlock()
	st := model.Status{ProcID: id, ExitCode: -1}
	if !ok {
		return st
	}
	rp.mu.Lock()
	defer rp.mu.Unlock()
	running := true
	select {
	case <-rp.done:
		running = false
	default:
	}
	st.Running = running
	st.PID = rp.cmd.Process.Pid
	st.Ports = rp.portList()
	st.ExitCode = rp.exitCode
	if running {
		st.StartedAt = rp.startedAt.UnixMilli()
	}
	return st
}

func (rp *runningProc) portList() []int {
	out := make([]int, 0, len(rp.ports))
	for p := range rp.ports {
		out = append(out, p)
	}
	sort.Ints(out)
	return out
}
