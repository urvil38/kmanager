package manager

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os/exec"
	"strings"
)

type Command struct {
	name         string
	rootCmd      string
	args         []string
	stderr       error
	stdout       string
	internal     bool
	succeed      bool
	generateArgs func(cc *ClusterConfig) []string
	runFn        func(cmd *Command) error
}

func (c *Command) execute(ctx context.Context, cc *ClusterConfig) {
	c.reset()
	if c.generateArgs != nil {
		c.args = c.generateArgs(cc)
	}
	c.stdout, c.stderr = RunCommand(ctx, c.rootCmd, c.args...)
	if c.stderr == nil {
		c.succeed = true
	}

	if c.runFn != nil {
		c.runFn(c)
	}
}

func (c *Command) reset() {
	c.succeed = false
	c.stderr = nil
	c.stdout = ""
}

type commandNotFound struct {
	name string
}

func (c commandNotFound) Error() string {
	return fmt.Sprintf("no command found of name %s", c.name)
}

func (cs *cmdSet) getCommand(cmdName string) (*Command, error) {
	cmd, isthere := cs.cmdMap[cmdName]
	if !isthere {
		return nil, commandNotFound{name: cmdName}
	}

	return &cmd, nil
}

func newCmdSet(cc *ClusterConfig, name string) *cmdSet {
	return &cmdSet{
		name:   name,
		cmdMap: make(map[string]Command),
		cc:     cc,
	}
}

type cmdSet struct {
	name   string
	cmdMap map[string]Command
	cmds   []Command
	cc     *ClusterConfig
}

func (cs *cmdSet) AddCmd(cmd Command) error {
	_, isthere := cs.cmdMap[cmd.name]
	if isthere {
		log.Fatal(cmd.name, ": command redefined!!")
	}

	cs.cmdMap[cmd.name] = cmd
	cs.cmds = append(cs.cmds, cmd)

	return nil
}

func RunCommand(ctx context.Context, name string, args ...string) (output string, err error) {
	fmt.Println(name + " " + strings.Join(args, " "))
	cmd := exec.CommandContext(ctx, name, args...)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return "", err
	}

	// scannerStdout := bufio.NewScanner(stdout)
	// go func() {
	// 	for scannerStdout.Scan() {
	// 		fmt.Printf("%s\n", scannerStdout.Text())
	// 	}
	// }()

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return "", err
	}
	// scannerStderr := bufio.NewScanner(stdout)
	// go func() {
	// 	for scannerStderr.Scan() {
	// 		fmt.Printf("%s\n", scannerStderr.Text())
	// 	}
	// }()

	err = cmd.Start()
	if err != nil {
		return "", err
	}

	stderrorCmd, err := ioutil.ReadAll(stderr)
	if err != nil {
		return "", err
	}

	stdoutCmd, err := ioutil.ReadAll(stdout)
	if err != nil {
		return "", err
	}

	if err := cmd.Wait(); err != nil {
		if len(stderrorCmd) > 0 {
			return "", errors.New(string(stderrorCmd))
		}
		return "", err
	}

	return string(stdoutCmd), nil
}
