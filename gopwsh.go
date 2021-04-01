package gopwsh

import (
	"fmt"
	"io"
	"runtime"
	"strings"
	"time"

	"github.com/brad-jones/goasync/v2/await"
	"github.com/brad-jones/goasync/v2/task"
	"github.com/brad-jones/goerr/v2"
	"github.com/brad-jones/gopwsh/backend"
	"github.com/thanhpk/randstr"
)

var newLine string

const bufferSize int = 64

func init() {
	newLine = "\n"
	if runtime.GOOS == "windows" {
		newLine = "\r\n"
	}
}

// Starter describes what we use to actually "start" a powershell process.
//
// This module includes an implementation for running processes locally.
// Other implementations such as starting a process via SSH are possible
// but "at this stage" are left as an exercise for the reader - PRs welcome :)
type Starter interface {
	LookPath(file string) (string, error)
	SetEnv(values map[string]string, combined bool)
	SetWorkingDir(v string)
	StartProcess(cmd string, args ...string) error
	Stderr() io.Reader
	Stdin() io.Writer
	Stdout() io.Reader
	Wait() error
}

// Shell is the primary object that represents a running PowerShell process.
//
// Create new instances of this with the "New()" function.
type Shell struct {
	backend      Starter
	env          map[string]string
	envCombined  bool
	pwshLocation string
	sudoLocation string
	wd           string
}

// Backend allows you set a custom backend or "Starter".
func Backend(b Starter) func(*Shell) error {
	return func(s *Shell) error {
		s.backend = b
		return nil
	}
}

// Elevated will create an elevated PowerShell process.
// This functionality assumes a "sudo" command exists on the system.
//
// On *nix systems this can probably be taken for granted.
// On Windows systems a package like the following should be installed:
//
// * https://github.com/gerardog/gsudo
// * https://github.com/brad-jones/winsudo
//
// Calling this function without any arguments, tells us you want an elevated
// session & we will do our best to locate a "sudo" binary for you. Otherwise
// you may provide a single argument of the path to a "sudo" binary.
func Elevated(sudoLocation ...string) func(*Shell) error {
	return func(s *Shell) error {
		if len(sudoLocation) == 1 {
			s.sudoLocation = sudoLocation[0]
			return nil
		}
		s.sudoLocation = "sudo"
		return nil
	}
}

// WorkingDir allows you to set a custom initial working directory for the
// PowerShell process.
func WorkingDir(wd string) func(*Shell) error {
	return func(s *Shell) error {
		s.wd = wd
		return nil
	}
}

// Env allows you to set custom environment variable for the PowerShell process.
func Env(env map[string]string) func(*Shell) error {
	return func(s *Shell) error {
		s.env = env
		return nil
	}
}

// EnvCombined is set to true will instruct the backend to combine these
// variables with the ones already define on the backend's environments.
func EnvCombined(v bool) func(*Shell) error {
	return func(s *Shell) error {
		s.envCombined = v
		return nil
	}
}

// PwshLocation allows you supply a custom path to a PowerShell executeable.
func PwshLocation(path string) func(*Shell) error {
	return func(s *Shell) error {
		s.pwshLocation = path
		return nil
	}
}

// New is a constructor like function for the Shell struct.
//
// All configuration is done through the functional options pattern.
// see: https://dave.cheney.net/2014/10/17/functional-options-for-friendly-apis
//
// e.g:
//	gopwsh.New(gopwsh.PwshLocation("/some/path/pwsh.exe"), ...)
//
// Some Defaults
//
// If no backend is set we will use the Local one.
//
// If no pwshLocation is set we will use the backend's LookPath method to first
// look for an executebale named "pwsh". On failure of that we will look for an
// executable named "powershell".
//
// envCombined is set to true
func New(decorators ...func(*Shell) error) (s *Shell, err error) {
	defer goerr.Handle(func(e error) { s = nil; err = e })

	s = &Shell{
		envCombined: true,
	}
	for _, decorator := range decorators {
		goerr.Check(decorator(s))
	}

	if s.backend == nil {
		s.backend = &backend.Local{}
	}

	s.backend.SetEnv(s.env, s.envCombined)
	s.backend.SetWorkingDir(s.wd)

	if s.pwshLocation == "" {
		if path, err := s.backend.LookPath("pwsh"); err == nil {
			s.pwshLocation = path
		} else {
			path, err := s.backend.LookPath("powershell")
			if err != nil {
				goerr.Check(goerr.New("Failed to locate a PowerShell binary"))
			}
			s.pwshLocation = path
		}
	}

	if s.sudoLocation != "" {
		if s.sudoLocation == "sudo" {
			path, err := s.backend.LookPath("sudo")
			if err != nil {
				goerr.Check(goerr.New("Failed to locate a sudo binary"))
			}
			s.sudoLocation = path
		}
		goerr.Check(
			s.backend.StartProcess(s.sudoLocation,
				s.pwshLocation, "-NoExit", "-Command", "-",
			),
			"Failed to start powershell process with sudo",
			s.sudoLocation,
			s.pwshLocation,
		)
		return
	}

	goerr.Check(
		s.backend.StartProcess(s.pwshLocation, "-NoExit", "-Command", "-"),
		"Failed to start powershell process",
		s.pwshLocation,
	)
	return
}

// MustNew is the same as New but panics on error instead of returning an error.
func MustNew(decorators ...func(*Shell) error) *Shell {
	s, err := New(decorators...)
	goerr.Check(err)
	return s
}

// Execute is what you can use to actually execute arbitrary powershell
// commands & script.
//
// It will return the STDOUT & STDERR in 2 separate strings.
// Even if STDERR is not empty then the err value will still be nil.
// Only errors associated with this gopwsh module will be returned.
//
// Just because a command returns STDERR doesn't necessarily mean failure.
// ie: some commands log progress messages / extra debugging info to STDERR
// but still successfully perform their task.
//
// ParserErrors are however considered fatal and will result in an error value
// being returned. The underlying PowerShell process will be killed and you
// won't be able to use this instance of the Shell any longer.
func (s *Shell) Execute(cmds ...string) (string, string, error) {
	stdout := ""
	stderr := ""

	for _, cmd := range cmds {
		o, e, err := s.execute(cmd)
		stdout = stdout + o
		stderr = stderr + e
		if err != nil {
			return stdout, stderr, goerr.Wrap(err, "failed to execute", cmd)
		}
	}

	return stdout, stderr, nil
}

func (s *Shell) execute(cmd string) (string, string, error) {
	if s.backend == nil {
		return "", "", goerr.Wrap("Cannot execute commands on closed shells.", cmd)
	}

	// Wrap the command in special markers so we know when to stop reading from the pipes
	outBoundary := createBoundary()
	errBoundary := createBoundary()
	full := fmt.Sprintf("%s; echo '%s'; [Console]::Error.WriteLine('%s')%s",
		cmd, outBoundary, errBoundary, newLine,
	)

	// Send the command to the running powershell process via STDIN
	_, err := s.backend.Stdin().Write([]byte(full))
	if err != nil {
		return "", "", goerr.Wrap(err, "Could not send PowerShell command", cmd)
	}

	// Read stdout and stderr
	results, err := await.FastAllOrError(
		streamReader(s.backend.Stdout(), outBoundary),
		streamReader(s.backend.Stderr(), errBoundary),
	)
	if err != nil {
		if strings.Contains(err.Error(), "ParserError") {
			s.Exit()
		}
		return "", "", goerr.Wrap(err, "Failed to read stdout/stderr steams")
	}
	sout := results[0].(string)
	serr := results[1].(string)

	return sout, serr, nil
}

// Exit is used to kill the powershell process.
//
// Typical usage might look like:
// 	shell := gopwsh.New()
// 	defer shell.Exit()
func (s *Shell) Exit() {
	if s.backend == nil {
		return
	}

	s.backend.Stdin().Write([]byte("exit" + newLine))

	// If it's possible to close stdin, do so.
	// Some backends, like the local one, do support it.
	closer, ok := s.backend.Stdin().(io.Closer)
	if ok {
		closer.Close()
	}

	s.backend.Wait()
	s.backend = nil
}

// QuoteArg can be used to escape string literals that you want to ensure
// don't get mangled between your Go code and PowerShell.
func QuoteArg(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "''") + "'"
}

func streamReader(stream io.Reader, boundary string) *task.Task {
	return task.New(func(t *task.Internal) {
		output := ""
		marker := boundary + newLine

		_, err := await.FastAny(
			task.New(func(t *task.Internal) {
				for {
					if t.ShouldStop() {
						return
					}

					if strings.Contains(output, "ParserError") {
						time.Sleep(time.Millisecond * 1)
						break
					}

					time.Sleep(time.Millisecond * 1)
				}

				t.Reject(output)
			}),
			task.New(func(t *task.Internal) {
				for {
					if t.ShouldStop() {
						return
					}

					buf := make([]byte, bufferSize)
					read, err := stream.Read(buf)
					if err != nil {
						t.Reject(err, "failed to read stream")
						return
					}

					output = output + string(buf[:read])

					if strings.HasSuffix(output, marker) {
						break
					}
				}
			}),
		)
		if err != nil {
			t.Reject(err)
			return
		}

		t.Resolve(strings.TrimSuffix(output, marker))
	})
}

func createBoundary() string {
	return "$gopwsh" + randstr.Hex(12) + "$"
}
