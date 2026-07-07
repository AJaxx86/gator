package main
// db connection: "postgres://postgres:postgres@localhost:5432/gator"

import (
	"github.com/ajaxx86/gator/internal/config"
	"github.com/ajaxx86/gator/internal/database"
	"database/sql"
	"github.com/google/uuid"
	"fmt"
	"os"
	"context"
	"time"
	_ "github.com/lib/pq"
)

type state struct {
	cfg *config.Config
	db *database.Queries
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

	db, err := sql.Open("postgres", cfg.DBURL)
	if err != nil {
		fmt.Println("Error opening database: %w", err)
		os.Exit(1)
	}
	dbQueries := database.New(db)

	cliState := state{cfg: &cfg, db: dbQueries}
	cmdList := map[string]func(*state, command) error{
		"login": handlerLogin,
		"register": handlerRegister,
		"reset": reset,
		"users": handlerGetUsers,
	}
	cmds := commands{list: cmdList}

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
	_, err := s.db.GetUser(context.Background(), user)
	if err != nil {
		return fmt.Errorf("User doesn't exist: %s", user)
	}

	err = s.cfg.SetUser(user)
	if err != nil {
		return err
	}

	fmt.Println("User logged in:", user)
	return nil
}


func handlerRegister(s *state, cmd command) error {
	ctx := context.Background()
	id := uuid.New()
	created_at := time.Now()
	updated_at := time.Now()
	name := cmd.args[0]

	_, err := s.db.GetUser(ctx, name)
	if err == nil {
		fmt.Println("User already exists:", name)
		os.Exit(1)
	}

	params := database.CreateUserParams{
		ID: id,
		CreatedAt: created_at,
		UpdatedAt: updated_at,
		Name: name,
	}
	_, err = s.db.CreateUser(ctx, params)
	if err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}

	err = s.cfg.SetUser(name)
	if err != nil {
		return fmt.Errorf("failed to set user config: %s\nDetails: %v", err, params)
	}

	registerMessage := fmt.Sprintf("User registered: %s\nID: %s\nCreated at: %s\n", params.Name, params.ID, params.CreatedAt.Format(time.DateTime))
	fmt.Println(registerMessage)
	return nil
}


func handlerGetUsers(s *state, cmd command) error {
	users, err := s.db.GetUsers(context.Background())
	if err != nil {
		return fmt.Errorf("failed to get users: %w", err)
	}

	for _, user := range users {
		if user.Name == s.cfg.UserName {
			fmt.Println("*", user.Name, "(current)")
			continue
		}
		fmt.Println("*", user.Name)
	}
	return nil
}


func reset(s *state, cmd command) error {
	err := s.db.Reset(context.Background())
	if err != nil {
		return fmt.Errorf("failed to reset: %w", err)
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
