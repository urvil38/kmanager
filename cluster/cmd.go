package cluster

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"strings"
)

type Command struct {
	Name         string
	RootCmd      string
	Args         []string
	Stderr       error
	Stdout       string
	Internal     bool
	InterActive  bool
	Succeed      bool
	GenerateArgs func(*Cluster) []string
	AfterFn      func(*Command) error
}

func (c *Command) Execute(ctx context.Context, cc *Cluster) {
	c.reset()
	if c.GenerateArgs != nil {
		c.Args = c.GenerateArgs(cc)
	}

	if c.InterActive {
		cmd := exec.CommandContext(ctx, c.RootCmd, c.Args...)
		cmd.Stderr = os.Stderr
		cmd.Stdout = os.Stdout
		cmd.Stdin = os.Stdin

		err := cmd.Run()
		if err != nil {
			c.Stderr = err
		} else {
			c.Succeed = true
		}

	} else {
		c.Stdout, c.Stderr = RunCommand(ctx, c.RootCmd, c.Args...)
		if c.Stderr == nil {
			c.Succeed = true
		}
	}

	if c.AfterFn != nil {
		c.AfterFn(c)
	}
}

func (c *Command) reset() {
	c.Succeed = false
	c.Stderr = nil
	c.Stdout = ""
}

type commandNotFound struct {
	name string
}

func (c commandNotFound) Error() string {
	return fmt.Sprintf("no command found of name %s", c.name)
}

func (cs *CmdSet) GetCommand(cmdName string) (Command, error) {
	cmd, isthere := cs.CmdMap[cmdName]
	if !isthere {
		return Command{}, commandNotFound{name: cmdName}
	}

	return cmd, nil
}

func NewCmdSet(cc *Cluster, name string) *CmdSet {
	return &CmdSet{
		Name:   name,
		CmdMap: make(map[string]Command),
		C:      cc,
	}
}

type CmdSet struct {
	Name   string
	CmdMap map[string]Command
	Cmds   []Command
	C      *Cluster
}

func (cs *CmdSet) AddCmd(cmd Command) error {
	_, isthere := cs.CmdMap[cmd.Name]
	if isthere {
		log.Fatal(cmd.Name, ": command redefined!!")
	}

	cs.CmdMap[cmd.Name] = cmd
	cs.Cmds = append(cs.Cmds, cmd)

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
