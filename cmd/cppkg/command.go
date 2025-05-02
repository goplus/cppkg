package main

import (
	"os"
	"os/exec"
	"strings"
)

var (
	ErrNotFound = exec.ErrNotFound
)

type Command struct {
	cmd      string
	installs [][]string
}

func NewCommand(cmd string, installs []string) *Command {
	inst := make([][]string, len(installs))
	for i, install := range installs {
		inst[i] = strings.Split(install, " ")
	}
	return &Command{
		cmd:      cmd,
		installs: inst,
	}
}

func (p *Command) New(quietInstall bool, args ...string) (cmd *exec.Cmd, err error) {
	app, err := p.Get(quietInstall)
	if err != nil {
		return
	}
	return exec.Command(app, args...), nil
}

func (p *Command) Get(quietInstall bool) (app string, err error) {
	app, err = exec.LookPath(p.cmd)
	if err == nil {
		return
	}
	amPath, install, err := p.getAppManager()
	if err != nil {
		return
	}
	c := exec.Command(amPath, install[1:]...)
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	if err = c.Run(); err != nil {
		return
	}
	return exec.LookPath(p.cmd)
}

func (p *Command) getAppManager() (amPath string, install []string, err error) {
	for _, install = range p.installs {
		am := install[0]
		if amPath, err = exec.LookPath(am); err == nil {
			return
		}
	}
	err = ErrNotFound
	return
}
