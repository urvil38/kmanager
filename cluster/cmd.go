package cluster

import (
	"bytes"
	"context"
	"fmt"
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
		c.Stdout, c.Stderr = RunCommand(ctx, c.Name, c.RootCmd, c.Args...)
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

func RunCommand(ctx context.Context, name, rootCmd string, args ...string) (output string, err error) {
	fmt.Println(name + ": " + rootCmd + " " + strings.Join(args, " "))
	var stdout, stderr bytes.Buffer
	cmd := exec.CommandContext(ctx, rootCmd, args...)
	cmd.Stderr = &stderr
	cmd.Stdout = &stdout

	err = cmd.Run()
	if err != nil {
		return "", err
	}
	if stderr.Len() > 0 {
		return "", err
	}
	return stdout.String(), nil
}
