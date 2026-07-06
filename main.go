package main
// db connection: "postgres://postgres:postgres@localhost:5432/gator"

import (
	"github.com/ajaxx86/gator/internal/config"
	"fmt"
	"os"
)

type state struct {
	cfg *config.Config
}

type command struct {
	name string
	args []string
}

type commands struct {
	list map[string]func(*state, command) error
}


func main() {
	if len(os.Args) < 2 {
		fmt.Println("Not enough args. Usage: gator <command> [args]")
		os.Exit(1)
	}
	cfg, err := config.Read()
	if err != nil {
		fmt.Println("Error reading config:", err)
		os.Exit(1)
	}

	cliState := state{cfg: &cfg}
	cmdList := map[string]func(*state, command) error{
		"login": handlerLogin,
	}
	cmds := commands{list: cmdList}
	for cmdName, cmdFunc := range cmds.list {
		cmds.list[cmdName] = cmdFunc
	}

	cmd := command{name: os.Args[1], args: os.Args[2:]}
	cmdErr := cmds.run(&cliState, cmd)
	if cmdErr != nil {
		fmt.Println("Error:", cmdErr)
		os.Exit(1)
	}
	os.Exit(0)
}


func handlerLogin(s *state, cmd command) error {
	if len(cmd.args) == 0 {
		return fmt.Errorf("Please enter a username.")
	}
	user := cmd.args[0]

	err := s.cfg.SetUser(user)
	if err != nil {
		return err
	}

	return nil
}


func (c *commands) run(s *state, cmd command) error {
	if command, ok := c.list[cmd.name]; ok {
		if err := command(s, cmd); err != nil {
			return err
		}
		return nil
	}

	return fmt.Errorf("unknown command: %s", cmd.name)
}


func (c *commands) register(name string, f func(*state, command) error) {
	c.list[name] = f
}
