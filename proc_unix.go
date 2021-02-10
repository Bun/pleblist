package main

import (
	"os/exec"
	"syscall"
)

func closeChild(cmd *exec.Cmd) {
	// Ensure child is closed when this program exits
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}
	cmd.SysProcAttr.Pdeathsig = syscall.SIGINT
}
