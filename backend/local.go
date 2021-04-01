package backend

import (
	"io"
	"os/exec"

	"github.com/brad-jones/goerr/v2"
	"github.com/brad-jones/goexec/v2"
)

type Local struct {
	command    *exec.Cmd
	decorators []func(*exec.Cmd) error
	stderr     io.ReadCloser
	stdin      io.WriteCloser
	stdout     io.ReadCloser
	exited     bool
}

func (b *Local) init() {
	if b.decorators == nil {
		b.decorators = []func(*exec.Cmd) error{}
	}
}

func (b *Local) LookPath(file string) (string, error) {
	return exec.LookPath(file)
}

func (b *Local) SetEnv(values map[string]string, combined bool) {
	b.init()
	if values == nil {
		values = map[string]string{}
	}
	e := goexec.Env(values)
	if combined {
		e = goexec.EnvCombined(values)
	}
	b.decorators = append(b.decorators, e)
}

func (b *Local) SetWorkingDir(v string) {
	b.init()
	if v != "" {
		b.decorators = append(b.decorators, goexec.Cwd(v))
	}
}

func (b *Local) StartProcess(cmd string, args ...string) (err error) {
	defer goerr.Handle(func(e error) { err = e })

	b.init()
	b.decorators = append(b.decorators, goexec.Args(args...))
	c, err := goexec.Cmd(cmd, b.decorators...)
	goerr.Check(err, "failed to create exec.Cmd")

	b.command = c
	b.command.Stdin = nil
	b.command.Stdout = nil
	b.command.Stderr = nil

	stdin, err := b.command.StdinPipe()
	goerr.Check(err, "Could not get hold of the PowerShell's stdin stream")
	b.stdin = stdin

	stdout, err := b.command.StdoutPipe()
	goerr.Check(err, "Could not get hold of the PowerShell's stdout stream")
	b.stdout = stdout

	stderr, err := b.command.StderrPipe()
	goerr.Check(err, "Could not get hold of the PowerShell's stderr stream")
	b.stderr = stderr

	goerr.Check(b.command.Start(), "Could not spawn PowerShell process")
	return
}

func (b *Local) Stderr() io.Reader {
	return b.stderr
}

func (b *Local) Stdin() io.Writer {
	return b.stdin
}

func (b *Local) Stdout() io.Reader {
	return b.stdout
}

func (b *Local) Wait() error {
	return b.command.Wait()
}
