package opener

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// dirToken marks where the target directory goes in a candidate's arguments.
// Some launchers take it as a bare operand, others behind a flag, and a few
// glue it onto one (`--working-directory={dir}`), so substitution beats
// appending.
const dirToken = "{dir}"

// candidate is one launcher we are willing to try, named by the binary to
// look up on PATH plus the arguments that point it at a directory.
type candidate struct {
	bin  string
	args []string
	// viaConsole marks launchers that go through a console host (cmd.exe).
	// Those get their window suppressed on Windows; GUI launchers must not,
	// or the window we are trying to open is hidden along with it.
	viaConsole bool
}

// launch runs the first candidate whose binary is installed. notFound is the
// error message used when none of them are, phrased for the UI because it is
// shown verbatim.
func launch(dir, notFound string, candidates []candidate) error {
	for _, c := range candidates {
		path, err := exec.LookPath(c.bin)
		if err != nil {
			continue
		}
		return start(dir, path, c)
	}
	return errors.New(notFound)
}

// start launches the candidate with dir as its working directory and
// detaches. The child owns a desktop window that may live for hours, so it is
// only waited on in the background, to reap it. Its exit code is deliberately
// ignored: Windows Explorer, for one, reports failure on success.
func start(dir, path string, c candidate) error {
	args := make([]string, len(c.args))
	for i, a := range c.args {
		args[i] = strings.ReplaceAll(a, dirToken, dir)
	}

	cmd := exec.Command(path, args...)
	cmd.Dir = dir
	// Terminals that ignore their cwd flag still inherit PWD, and the shell
	// they launch reads it for its prompt.
	cmd.Env = append(os.Environ(), "PWD="+dir)
	if c.viaConsole {
		hideConsole(cmd)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("could not launch %s: %w", c.bin, err)
	}
	go func() { _ = cmd.Wait() }()
	return nil
}

// envCandidate turns an environment variable naming a program (e.g. $TERMINAL)
// into a candidate, so a user's explicit choice wins over our probe order. It
// returns nil when the variable is unset.
func envCandidate(key string, args ...string) []candidate {
	bin := strings.TrimSpace(os.Getenv(key))
	if bin == "" {
		return nil
	}
	return []candidate{{bin: bin, args: args}}
}
