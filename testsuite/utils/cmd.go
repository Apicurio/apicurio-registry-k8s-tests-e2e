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
	err := ExecuteCmd(logOutput, &Command{Cmd: append([]string{name}, arg...)})
	Expect(err).ToNot(HaveOccurred())
}

//Command encapsulates parameters of ExecuteCmd
type Command struct {
	Cmd []string
	Env []string
}

//ExecuteCmd executes a command
func ExecuteCmd(logOutput bool, command *Command) error {
	log.Info("Executing command ", "cmd", strings.Join(command.Cmd, " "))
	cmd := exec.Command(command.Cmd[0], command.Cmd[1:]...)
	if command.Env != nil {
		log.Info("With env", "env", strings.Join(command.Env, " "))
		cmd.Env = os.Environ()
		cmd.Env = append(cmd.Env, command.Env...)
	}
	if logOutput {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		err := cmd.Run()
		if err != nil {
			return err
		}
	} else {
		err := cmd.Run()
		if err != nil {
			return err
		}
	}
	return nil
}
