// +build windows

package windows

import (
	"errors"
	"fmt"

	"github.com/Sirupsen/logrus"
	"github.com/docker/docker/daemon/execdriver"
	"github.com/microsoft/hcsshim"
)

// Exec implements the exec driver Driver interface.
func (d *Driver) Exec(c *execdriver.Command, processConfig *execdriver.ProcessConfig, pipes *execdriver.Pipes, hooks execdriver.Hooks) (int, error) {

	var (
		term     execdriver.Terminal
		err      error
		exitCode int32
		errno    uint32
	)

	active := d.activeContainers[c.ID]
	if active == nil {
		return -1, fmt.Errorf("Exec - No active container exists with ID %s", c.ID)
	}

	createProcessParms := hcsshim.CreateProcessParams{
		EmulateConsole:   processConfig.Tty, // Note NOT c.ProcessConfig.Tty
		WorkingDirectory: c.WorkingDir,
	}

	// Configure the environment for the process // Note NOT c.ProcessConfig.Tty
	createProcessParms.Environment = setupEnvironmentVariables(processConfig.Env)

	// While this should get caught earlier, just in case, validate that we
	// have something to run.
	if processConfig.Entrypoint == "" {
		err = errors.New("No entrypoint specified")
		logrus.Error(err)
		return -1, err
	}

	// Build the command line of the process
	createProcessParms.CommandLine = processConfig.Entrypoint
	for _, arg := range processConfig.Arguments {
		logrus.Debugln("appending ", arg)
		createProcessParms.CommandLine += " " + arg
	}
	logrus.Debugln("commandLine: ", createProcessParms.CommandLine)

	// Start the command running in the container.
	pid, stdin, stdout, stderr, err := hcsshim.CreateProcessInComputeSystem(c.ID, pipes.Stdin != nil, true, !processConfig.Tty, createProcessParms)
	if err != nil {
		logrus.Errorf("CreateProcessInComputeSystem() failed %s", err)
		return -1, err
	}

	// Now that the process has been launched, begin copying data to and from
	// the named pipes for the std handles.
	setupPipes(stdin, stdout, stderr, pipes)

	// Note NOT c.ProcessConfig.Tty
	if processConfig.Tty {
		term = NewTtyConsole(c.ID, pid)
	} else {
		term = NewStdConsole()
	}
	processConfig.Terminal = term

	// Invoke the start callback
	if hooks.Start != nil {
		// A closed channel for OOM is returned here as it will be
		// non-blocking and return the correct result when read.
		chOOM := make(chan struct{})
		close(chOOM)
		hooks.Start(&c.ProcessConfig, int(pid), chOOM)
	}

	if exitCode, errno, err = hcsshim.WaitForProcessInComputeSystem(c.ID, pid, hcsshim.TimeoutInfinite); err != nil {
		if errno == hcsshim.Win32PipeHasBeenEnded {
			logrus.Debugf("Exiting Run() after WaitForProcessInComputeSystem failed with recognised error 0x%X", errno)
			return hcsshim.WaitErrExecFailed, nil
		}
		logrus.Warnf("WaitForProcessInComputeSystem failed (container may have been killed): 0x%X %s", errno, err)
		return -1, err
	}

	logrus.Debugln("Exiting Run()", c.ID)
	return int(exitCode), nil
}
