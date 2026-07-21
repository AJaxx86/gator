# Gator

Gator is a command-line RSS feed aggreGATOR I made following the aggregator boot.dev course. It stores users, feeds, and posts
in PostgreSQL so you can follow feeds and browse their latest entries from your
terminal.

## Prerequisites

To build and run Gator, you need:

- [Go](https://go.dev/doc/install)
- [PostgreSQL](https://www.postgresql.org/download/)

Make sure PostgreSQL is running before starting Gator.

## Install

Install the `gator` CLI with:

```sh
go install github.com/ajaxx86/gator@latest
```

The command installs the executable in Go's binary directory. If your shell
cannot find `gator`, add `$(go env GOPATH)/bin` to your `PATH`.

## Set up the database

Create a PostgreSQL database for Gator:

```sh
createdb gator
```

The repository's migrations use
[Goose](https://github.com/pressly/goose). From a clone of this repository,
install Goose and apply the migrations:

```sh
go install github.com/pressly/goose/v3/cmd/goose@latest
goose -dir sql/schema postgres "postgres://postgres:postgres@localhost:5432/gator?sslmode=disable" up
```

Change the connection string to match your PostgreSQL username, password,
host, port, and database.

## Configure Gator

Create a file named `.gatorconfig.json` in your home directory:

```json
{
  "DBURL": "postgres://postgres:postgres@localhost:5432/gator?sslmode=disable",
  "UserName": ""
}
```

Gator updates `UserName` when you run `register` or `login`, so it can be empty
initially. The `DBURL` value must point to the database you created above.

## Run

Every command follows this form:

```sh
gator <command> [arguments]
```

Start by registering a user, adding a feed, and running the aggregator:

```sh
gator register alice
gator addfeed "Hacker News" "https://hnrss.org/newest"
gator agg 1m
```

`agg` runs continuously and fetches one feed at the interval provided. Stop it
with `Ctrl+C`. In another terminal, browse the saved posts:

```sh
gator browse 10
```

Useful commands include:

| Command | Description |
| --- | --- |
| `gator register <name>` | Create a user and make it the current user. |
| `gator login <name>` | Switch to an existing user. |
| `gator users` | List all users. |
| `gator addfeed <name> <url>` | Add a feed and follow it. Quote names containing spaces. |
| `gator feeds` | List all feeds. |
| `gator follow <url>` | Follow an existing feed by URL. |
| `gator following` | List feeds followed by the current user. |
| `gator unfollow <url>` | Unfollow a feed by URL. |
| `gator agg <duration>` | Fetch feeds repeatedly (for example, `30s` or `1m`). |
| `gator browse [limit]` | Show posts for the current user; the default limit is 2. |
| `gator reset` | Delete all users and their associated data. |

**NOTE:** You can run the tool directly from the folder by adding `./` to the start.
