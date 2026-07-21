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
	"errors"
	"strconv"
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
	cmds := commands{
		list: make(map[string]func(*state, command) error),
	}
	cmds.register("login", handlerLogin)
	cmds.register("register", handlerRegister)
	cmds.register("reset", reset)
	cmds.register("users", handlerGetUsers)
	cmds.register("agg", handlerAgg)
	cmds.register("addfeed", middlewareLoggedIn(handlerAddFeed))
	cmds.register("feeds", handlerListFeeds)
	cmds.register("follow", middlewareLoggedIn(handlerFollowFeed))
	cmds.register("following", middlewareLoggedIn(handlerListUserFeedFollows))
	cmds.register("unfollow", middlewareLoggedIn(handlerDeleteFeedFollow))
	cmds.register("browse", middlewareLoggedIn(handlerBrowsePosts))

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
	if len(cmd.args) == 0 {
		return fmt.Errorf("Please enter a username.")
	}

	ctx := context.Background()
	id := uuid.New()
	created_at := time.Now()
	updated_at := time.Now()
	name := cmd.args[0]

	_, err := s.db.GetUser(ctx, name)
	if err == nil {
		return fmt.Errorf("User already exists: %s", name)
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("failed to check user: %w", err)
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


func handlerBrowsePosts(s *state, cmd command, user database.User) error {
	postLimit := int64(2)
	if len(cmd.args) > 0 {
		parsedLimit, err := strconv.ParseInt(cmd.args[0], 10, 64)
		if err != nil {
			fmt.Printf("Post limit invalid. Using default (2).")
		} else {
			postLimit = parsedLimit
		}
	}

	params := database.GetPostsForUserParams{
		UserID: user.ID,
		Limit: postLimit,
	}
	posts, err := s.db.GetPostsForUser(context.Background(), params)
	if err != nil {
		return err
	}

	for _, p := range posts {
		msg := fmt.Sprintf("Title: %s\nDescription: %s\nURL: %s\n -----\n", p.Title, p.Description.String, p.Url)
		fmt.Println(msg)
	}

	return nil
}


func handlerAgg(s *state, cmd command) error {
	if len(cmd.args) != 1 {
		return fmt.Errorf("Invalid usage. Include the time (i.e. 1s, 30m, 1.5h).")
	}
	timeBetweenReqs, err := time.ParseDuration(cmd.args[0])
	if err != nil {
		return err
	}
	fmt.Println("Fetching feeds every " + cmd.args[0])

	ticker := time.NewTicker(timeBetweenReqs)
	for ; ; <-ticker.C {
		fmt.Println("Fetching posts...")
		err := scrapeFeeds(s)
		if err != nil {
			fmt.Printf("scrapeFeeds err: %s\n", err)
		}
	}
}


func handlerAddFeed(s *state, cmd command, user database.User) error {
	if len(cmd.args) < 2 {
		return fmt.Errorf("Not enough arguments. Enter a name and URL.")
	}

	feedID := uuid.New()
	userID := user.ID
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

	msg := fmt.Sprintf("Saved feed for %s: %v\n", s.cfg.UserName, feedDetails)
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


func handlerFollowFeed(s *state, cmd command, user database.User) error {
	if len(cmd.args) != 1 {
		return fmt.Errorf("Invalid argument. Enter the URL you want to follow.")
	}

	followURL := cmd.args[0]
	feed, err := s.db.GetFeed(context.Background(), followURL)
	if err != nil {
		return fmt.Errorf("failed to get feed: %w", err)
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


func handlerListUserFeedFollows(s *state, cmd command, user database.User) error {
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


func handlerDeleteFeedFollow(s *state, cmd command, user database.User) error {
	if len(cmd.args) != 1 {
		return fmt.Errorf("Invalid arguments. Usage: unfollow <feed_name>")
	}

	url := cmd.args[0]
	feed, err := s.db.GetFeed(context.Background(), url)
	if err != nil {
		return fmt.Errorf("failed to get feed: %w", err)
	}

	params := database.DeleteFeedFollowParams{
		FeedID: feed.ID,
		UserID: user.ID,
	}
	if err := s.db.DeleteFeedFollow(context.Background(), params); err != nil {
		return fmt.Errorf("failed to delete feed follow: %w", err)
	}

	fmt.Printf("Unfollowed %s for user %s\n", feed.Name, user.Name)
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


func (c *commands) register(name string, f func(s *state, cmd command) error) {
	c.list[name] = f
}


func scrapeFeeds(s *state) error {
	nextFeed, err := s.db.GetNextFeedToFetch(context.Background())
	if err != nil {
		return err
	}

	params := database.MarkFeedFetchedParams{
		ID: nextFeed.ID,
		LastFetchedAt: sql.NullTime{
			Time: time.Now(),
			Valid: true,
		},
		UpdatedAt: time.Now(),
	}

	feed, err := fetchFeed(context.Background(), nextFeed.Url)
	if err != nil {
		return err
	}
	err = s.db.MarkFeedFetched(context.Background(), params)
	if err != nil {
		return err
	}

	feedPosts := feed.Channel.Item
	for _, post := range feedPosts {
		publishedAt, err := time.Parse(time.RFC1123, post.PubDate)
		if err != nil {
			fmt.Printf("Couldn't parse time of publish: %s\n", err)
		}

		postParams := database.CreatePostParams{
			ID: uuid.New(),
			CreatedAt: time.Now(),
			UpdatedAt: sql.NullTime{
				Time: time.Now(),
				Valid: true,
			},
			Title: post.Title,
			Url: post.Link,
			Description: sql.NullString{
				String: post.Description,
				Valid: true,
			},
			PublishedAt: sql.NullTime{
				Time: publishedAt,
				Valid: true,
			},
			FeedID: nextFeed.ID,
		}
		_, err = s.db.CreatePost(context.Background(), postParams)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			fmt.Printf("Error creating post: %s\n", err)
		}
	}

	return nil
}


func middlewareLoggedIn(handler func(s *state, cmd command, user database.User) error) func(*state, command) error {
	return func(s *state, cmd command) error {
		user, err := s.db.GetUser(context.Background(), s.cfg.UserName)
		if err != nil {
			return fmt.Errorf("failed to get user: %w", err)
		}
		return handler(s, cmd, user)
	}
}
