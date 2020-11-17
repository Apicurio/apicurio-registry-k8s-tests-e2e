package utils

import (
	"os"
	"os/exec"
	"strings"

	logf "sigs.k8s.io/controller-runtime/pkg/log"

	. "github.com/onsi/gomega"
)

var log = logf.Log.WithName("utils")

//ExecuteCmdOrDie executes a command
func ExecuteCmdOrDie(logOutput bool, name string, arg ...string) {
	ExecuteCmdOrDieCore(logOutput, true, name, arg...)
}

func ExecuteCmdOrDieCore(logOutput bool, logCmd bool, name string, arg ...string) {
	err := ExecuteCmdCore(logOutput, logCmd, &Command{Cmd: append([]string{name}, arg...)})
	Expect(err).ToNot(HaveOccurred())
}

//Command encapsulates parameters of ExecuteCmd
type Command struct {
	Cmd []string
	Env []string
}

//ExecuteCmd executes a command
func ExecuteCmd(logOutput bool, command *Command) error {
	return ExecuteCmdCore(logOutput, true, command)
}

func ExecuteCmdCore(logOutput bool, logCmd bool, command *Command) error {
	var stdOutFile *os.File = nil
	var stdErrFile *os.File = nil
	if logOutput {
		stdOutFile = os.Stdout
		stdErrFile = os.Stderr
	}
	return Execute(command, stdOutFile, stdErrFile, logCmd)
}

func Execute(command *Command, stdOutFile *os.File, stdErrFile *os.File, logCmd bool) error {
	if logCmd {
		log.Info("Executing command ", "cmd", strings.Join(command.Cmd, " "))
	}
	cmd := exec.Command(command.Cmd[0], command.Cmd[1:]...)
	if command.Env != nil {
		if logCmd {
			log.Info("With env", "env", strings.Join(command.Env, " "))
		}
		cmd.Env = os.Environ()
		cmd.Env = append(cmd.Env, command.Env...)
	}
	if stdOutFile != nil {
		cmd.Stdout = stdOutFile
	}
	if stdErrFile != nil {
		cmd.Stderr = stdErrFile
	}
	err := cmd.Run()
	if err != nil {
		return err
	}
	return nil
}
