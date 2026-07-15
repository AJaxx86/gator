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
		"agg": handlerAgg,
		"addfeed": handlerAddFeed,
		"feeds": handlerListFeeds,
		"follow": handlerFollowFeed,
		"following": handlerListUserFeedFollows,
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


func handlerAgg(s *state, cmd command) error {
	// if len(cmd.args) == 0 {
	// 	return fmt.Errorf("no URL provided")
	// }
	url := "https://www.wagslane.dev/index.xml" //cmd.args[0]

	feed, err := fetchFeed(context.Background(), url)
	if err != nil {
		return fmt.Errorf("failed to fetch feed: %w", err)
	}

	fmt.Println(feed)
	return nil
}


func handlerAddFeed(s *state, cmd command) error {
	if len(cmd.args) < 2 {
		return fmt.Errorf("Not enough arguments. Enter a name and URL.")
	}

	userData, err := s.db.GetUser(context.Background(), s.cfg.UserName)
	if err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}

	feedID := uuid.New()
	userID := userData.ID
	feedName := cmd.args[0]
	feedURL := cmd.args[1]
	createdAt := time.Now()
	updatedAt := time.Now()
	params := database.SaveFeedParams{
		ID: feedID,
		UserID: userID,
		Name: feedName,
		Url: feedURL,
		CreatedAt: createdAt,
		UpdatedAt: updatedAt,
	}

	feedDetails, err := s.db.SaveFeed(context.Background(), params)
	if err != nil {
		return fmt.Errorf("failed to save feed: %w", err)
	}

	_, err = s.db.CreateFeedFollow(context.Background(), database.CreateFeedFollowParams{
		ID: uuid.New(),
		UserID: userID,
		FeedID: feedID,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	})
	if err != nil {
		return fmt.Errorf("failed to save feed follow: %w", err)
	}

	msg := fmt.Sprintf("Saved feed for %s: %s", s.cfg.UserName, feedDetails)
	fmt.Print(msg)
	return nil
}


func handlerListFeeds(s *state, cmd command) error {
	feeds, err := s.db.ListFeeds(context.Background())
	if err != nil {
		return fmt.Errorf("failed to list feeds: %w", err)
	}

	for _, feed := range feeds {
		fmt.Printf("%s:\n - URL: %s\n - Added by: %s\n", feed.Name, feed.Url, feed.AddedBy.String)
	}

	return nil
}


func handlerFollowFeed(s *state, cmd command) error {
	if len(cmd.args) != 1 {
		return fmt.Errorf("Invalid argument. Enter the URL you want to follow.")
	}

	followURL := cmd.args[0]
	feed, err := s.db.GetFeed(context.Background(), followURL)
	if err != nil {
		return fmt.Errorf("failed to get feed: %w", err)
	}

	user, err := s.db.GetUser(context.Background(), s.cfg.UserName)
	if err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}

	createdAt := time.Now()
	updatedAt := createdAt
	params := database.CreateFeedFollowParams{
		ID:        uuid.New(),
		CreatedAt: createdAt,
		UpdatedAt: updatedAt,
		UserID:    user.ID,
		FeedID:    feed.ID,
	}

	follow, err := s.db.CreateFeedFollow(context.Background(), params)
	if err != nil {
		return fmt.Errorf("failed to follow feed: %w", err)
	}

	fmt.Printf("Created feed follow: %s\n", follow)
	return nil
}


func handlerListUserFeedFollows(s *state, cmd command) error {
	user, err := s.db.GetUser(context.Background(), s.cfg.UserName)
	if err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}

	follows, err := s.db.GetFeedFollowsForUser(context.Background(), user.ID)
	if err != nil {
		return fmt.Errorf("failed to get feed follows: %w", err)
	}

	fmt.Println(s.cfg.UserName, "is following:")
	for _, follow := range follows {
		fmt.Printf("%s: %s\n", follow.Name, follow.Url)
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
