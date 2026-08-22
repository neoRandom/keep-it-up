package cliadapter

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strconv"

	"keep-it-up/internal/core/port"
	"keep-it-up/internal/core/service"
)

var (
	ErrNoCommand         = errors.New("no command given")
	ErrNoSubcommand      = errors.New("no subcommand given")
	ErrUnknownCommand    = errors.New("unknown command")
	ErrUnknownSubcommand = errors.New("unknown subcommand")
	ErrWrongArgCount     = errors.New("wrong number of arguments")
)

const usage = `keep-it-up --- system management CLI

Usage:
  keepitup <noun> <verb> [args...]

Nouns:
  game     add    <name>
           update <id> <name>
           delete <id>
  access   grant  <game id> <player id>
           revoke <game id> <player id>
           check  <game id> <player id>
  player   add          <name> <username> <password>
           rename       <id> <name>
           passwd       <username> <current password> <new password>
           passwd-force <username> <password>
           delete       <id>
  auth     validate-passwd <password>
           hash-passwd     <password>
           check-passwd    <username> <password>
  fetch    games        <player id>
           shared       <game id>
           interactions <game id> <limit>
  command  save   <game id> <player id> <duration in seconds>
           resume <game id> <player id>
           pause  <game id> <player id>
`

// Deps groups the driver ports and I/O streams the CLI needs. It is a
// struct rather than positional constructor params so call sites stay
// self-documenting as the dependency count grows.
//
// The six fields mirror driver.go's interface segregation exactly (one
// field per port) rather than collapsing them into a single composed
// interface. See the design-notes reply for the trade-off against
// composition — this file makes the conservative choice since it assumes
// nothing about how internal/application/usecase structures its types.
type Deps struct {
	Games    port.GameManagement
	Access   port.AccessManagement
	Players  port.PlayerManagement
	Auth     port.Authentication
	Fetch    port.DataFetching
	Commands port.GameCommands

	Stdout io.Writer
	Stderr io.Writer
}

// CLI is the driving adapter itself.
type CLI struct {
	d Deps
}

func New(d Deps) *CLI {
	return &CLI{d: d}
}

func (c *CLI) Run(ctx context.Context, args []string) error {
	if len(args) == 0 {
		fmt.Fprint(c.d.Stderr, usage)
		return ErrNoCommand
	}

	noun, rest := args[0], args[1:]
	switch noun {
	case "game":
		return c.runGame(ctx, rest)
	case "access":
		return c.runAccess(ctx, rest)
	case "player":
		return c.runPlayer(ctx, rest)
	case "auth":
		return c.runAuth(ctx, rest)
	case "data":
		return c.runFetch(ctx, rest)
	case "command":
		return c.runCommand(ctx, rest)
	case "help", "-h", "--help":
		fmt.Fprint(c.d.Stdout, usage)
		return nil
	default:
		fmt.Fprint(c.d.Stderr, usage)
		return fmt.Errorf("%w: %q", ErrUnknownCommand, noun)
	}
}

// parseID parses a decimal int64 identifier from a CLI argument, wrapping
// the error with the offending value for a legible message.
func parseID(s string) (int64, error) {
	id, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid id %q: %w", s, err)
	}
	return id, nil
}

func wrongArgs(cmd, usage string) error {
	return fmt.Errorf("%s: %w (usage: %s)", cmd, ErrWrongArgCount, usage)
}

// --- game: port.GameManagement ---------------------------------------

func (c *CLI) runGame(ctx context.Context, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("game: %w", ErrNoSubcommand)
	}
	verb, rest := args[0], args[1:]
	switch verb {
	case "add":
		if len(rest) != 1 {
			return wrongArgs("game add", "game add <name>")
		}
		game, err := c.d.Games.AddGame(ctx, rest[0])
		if err != nil {
			return fmt.Errorf("game add: %w", err)
		}
		fmt.Fprintf(c.d.Stdout, "%+v\n", game)
		return nil

	case "update":
		if len(rest) != 2 {
			return wrongArgs("game update", "game update <id> <name>")
		}
		id, err := parseID(rest[0])
		if err != nil {
			return fmt.Errorf("game update: %w", err)
		}
		if err := c.d.Games.UpdateGame(ctx, id, rest[1]); err != nil {
			return fmt.Errorf("game update: %w", err)
		}
		fmt.Fprintf(c.d.Stdout, "game %d updated\n", id)
		return nil

	case "delete":
		if len(rest) != 1 {
			return wrongArgs("game delete", "game delete <id>")
		}
		id, err := parseID(rest[0])
		if err != nil {
			return fmt.Errorf("game delete: %w", err)
		}
		if err := c.d.Games.DeleteGame(ctx, id); err != nil {
			return fmt.Errorf("game delete: %w", err)
		}
		fmt.Fprintf(c.d.Stdout, "game %d deleted\n", id)
		return nil

	default:
		return fmt.Errorf("game %s: %w", verb, ErrUnknownSubcommand)
	}
}

// --- access: port.AccessManagement -----------------------------------

func (c *CLI) runAccess(ctx context.Context, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("access: %w", ErrNoSubcommand)
	}
	verb, rest := args[0], args[1:]
	switch verb {
	case "grant":
		if len(rest) != 2 {
			return wrongArgs("access grant", "access grant <game id> <player id>")
		}
		gameID, err := parseID(rest[0])
		if err != nil {
			return fmt.Errorf("access grant: %w", err)
		}
		playerID, err := parseID(rest[1])
		if err != nil {
			return fmt.Errorf("access grant: %w", err)
		}
		if err := c.d.Access.GrantPlayerAccess(ctx, gameID, playerID); err != nil {
			return fmt.Errorf("access grant: %w", err)
		}
		fmt.Fprintf(c.d.Stdout, "player %d granted access to game %d\n", playerID, gameID)
		return nil

	case "revoke":
		if len(rest) != 2 {
			return wrongArgs("access revoke", "access revoke <game id> <player id>")
		}
		gameID, err := parseID(rest[0])
		if err != nil {
			return fmt.Errorf("access revoke: %w", err)
		}
		playerID, err := parseID(rest[1])
		if err != nil {
			return fmt.Errorf("access revoke: %w", err)
		}
		if err := c.d.Access.RevokePlayerAccess(ctx, gameID, playerID); err != nil {
			return fmt.Errorf("access revoke: %w", err)
		}
		fmt.Fprintf(c.d.Stdout, "player %d access to game %d revoked\n", playerID, gameID)
		return nil

	case "check":
		if len(rest) != 2 {
			return wrongArgs("access check", "access check <game id> <player id>")
		}
		gameID, err := parseID(rest[0])
		if err != nil {
			return fmt.Errorf("access check: %w", err)
		}
		playerID, err := parseID(rest[1])
		if err != nil {
			return fmt.Errorf("access check: %w", err)
		}

		access, err := c.d.Access.CheckPlayerAccess(ctx, gameID, playerID)
		if err != nil {
			return fmt.Errorf("access check: %w", err)
		}
		fmt.Fprintf(
			c.d.Stdout, 
			"player %d access to game %d: %v\n", 
			playerID, gameID, access,
		)
		return nil

	default:
		return fmt.Errorf("access %s: %w", verb, ErrUnknownSubcommand)
	}
}

// --- player: port.PlayerManagement -----------------------------------

func (c *CLI) runPlayer(ctx context.Context, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("player: %w", ErrNoSubcommand)
	}
	verb, rest := args[0], args[1:]
	switch verb {
	case "add":
		if len(rest) != 3 {
			return wrongArgs("player add", "player add <name> <username> <password>")
		}
		player, err := c.d.Players.AddPlayer(ctx, rest[0], rest[1], rest[2])
		if err != nil {
			return fmt.Errorf("player add: %w", err)
		}
		fmt.Fprintf(c.d.Stdout, "%+v\n", player)
		return nil

	case "rename":
		if len(rest) != 2 {
			return wrongArgs("player rename", "player rename <id> <name>")
		}
		id, err := parseID(rest[0])
		if err != nil {
			return fmt.Errorf("player rename: %w", err)
		}
		if err := c.d.Players.UpdatePlayerName(ctx, id, rest[1]); err != nil {
			return fmt.Errorf("player rename: %w", err)
		}
		fmt.Fprintf(c.d.Stdout, "player %d renamed\n", id)
		return nil

	case "passwd":
		if len(rest) != 3 {
			return wrongArgs("player passwd", "player passwd <username> <current password> <new password>")
		}
		if err := c.d.Players.UpdatePlayerPassword(ctx, rest[0], rest[1], rest[2]); err != nil {
			return fmt.Errorf("player passwd: %w", err)
		}
		fmt.Fprintf(c.d.Stdout, "player '%s' password updated\n", rest[0])
		return nil

	case "passwd-force":
		if len(rest) != 2 {
			return wrongArgs("player passwd-force", "player passwd-force <username> <password>")
		}
		if err := c.d.Players.UpdatePlayerPasswordForce(ctx, rest[0], rest[1]); err != nil {
			return fmt.Errorf("player passwd-force: %w", err)
		}
		fmt.Fprintf(c.d.Stdout, "player '%s' password forcibly set\n", rest[0])
		return nil

	case "delete":
		if len(rest) != 1 {
			return wrongArgs("player delete", "player delete <id>")
		}
		id, err := parseID(rest[0])
		if err != nil {
			return fmt.Errorf("player delete: %w", err)
		}
		if err := c.d.Players.DeletePlayer(ctx, id); err != nil {
			return fmt.Errorf("player delete: %w", err)
		}
		fmt.Fprintf(c.d.Stdout, "player %d deleted\n", id)
		return nil

	default:
		return fmt.Errorf("player %s: %w", verb, ErrUnknownSubcommand)
	}
}

// --- auth: port.Authentication, minus LoginPlayer --------------------

func (c *CLI) runAuth(ctx context.Context, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("auth: %w", ErrNoSubcommand)
	}
	verb, rest := args[0], args[1:]
	switch verb {
	case "validate-passwd":
		if len(rest) != 1 {
			return wrongArgs("auth validate-passwd", "auth validate-passwd <password>")
		}
		if err := service.IsPasswordValid(rest[0]); err != nil {
			return fmt.Errorf("auth validate-passwd: %w", err)
		}
		fmt.Fprintln(c.d.Stdout, "valid")
		return nil

	case "hash-passwd":
		if len(rest) != 1 {
			return wrongArgs("auth hash-passwd", "auth hash-passwd <password>")
		}
		hash, err := service.GeneratePasswordHash(rest[0])
		if err != nil {
			return fmt.Errorf("auth hash-passwd: %w", err)
		}
		fmt.Fprintln(c.d.Stdout, hash)
		return nil

	case "check-passwd":
		if len(rest) != 2 {
			return wrongArgs("auth check-passwd", "auth check-passwd <username> <password>")
		}
		ok, err := c.d.Auth.CheckPlayerPassword(ctx, rest[0], rest[1])
		if err != nil {
			return fmt.Errorf("auth check-passwd: %w", err)
		}
		fmt.Fprintln(c.d.Stdout, ok)
		return nil

	default:
		return fmt.Errorf("auth %s: %w", verb, ErrUnknownSubcommand)
	}
}

// --- data: port.DataFetching ------------------------------------------

func (c *CLI) runFetch(ctx context.Context, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("data: %w", ErrNoSubcommand)
	}
	verb, rest := args[0], args[1:]
	switch verb {
	case "games":
		if len(rest) != 1 {
			return wrongArgs("fetch games", "fetch games <player id>")
		}
		playerID, err := parseID(rest[0])
		if err != nil {
			return fmt.Errorf("fetch games: %w", err)
		}
		games, err := c.d.Fetch.ListPlayerGames(ctx, playerID)
		if err != nil {
			return fmt.Errorf("fetch games: %w", err)
		}
		for _, g := range games {
			fmt.Fprintf(c.d.Stdout, "%+v\n", g)
		}
		return nil

	case "shared":
		if len(rest) != 1 {
			return wrongArgs("fetch shared", "fetch shared <game id>")
		}
		gameID, err := parseID(rest[0])
		if err != nil {
			return fmt.Errorf("fetch shared: %w", err)
		}
		shared, err := c.d.Fetch.GetSharedData(ctx, gameID)
		if err != nil {
			return fmt.Errorf("fetch shared: %w", err)
		}
		fmt.Fprintf(c.d.Stdout, "%+v\n", shared)
		return nil

	case "interactions":
		if len(rest) != 2 {
			return wrongArgs("fetch interactions", "fetch interactions <game id> <limit>")
		}
		gameID, err := parseID(rest[0])
		if err != nil {
			return fmt.Errorf("fetch interactions: %w", err)
		}
		limit, err := strconv.ParseInt(rest[1], 10, 64)
		if err != nil {
			return fmt.Errorf("fetch interactions: invalid limit %q: %w", rest[1], err)
		}
		interactions, err := c.d.Fetch.ListInteractions(ctx, gameID, limit)
		if err != nil {
			return fmt.Errorf("fetch interactions: %w", err)
		}
		for _, i := range interactions {
			fmt.Fprintf(c.d.Stdout, "%+v\n", i)
		}
		return nil

	default:
		return fmt.Errorf("fetch %s: %w", verb, ErrUnknownSubcommand)
	}
}

// --- command: port.GameCommands ---------------------------------------
//
// Named "command" rather than "game-cmd" to keep the noun namespace
// readable; it maps 1:1 onto GameCommands (save/resume/pause) and doesn't
// overlap with the "game" noun (GameManagement: add/update/delete).

func (c *CLI) runCommand(ctx context.Context, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("command: %w", ErrNoSubcommand)
	}
	verb, rest := args[0], args[1:]
	switch verb {
	case "save":
		if len(rest) != 3 {
			return wrongArgs("command save", "command save <game id> <player id> <duration in seconds>")
		}
		gameID, err := parseID(rest[0])
		if err != nil {
			return fmt.Errorf("command save: %w", err)
		}
		playerID, err := parseID(rest[1])
		if err != nil {
			return fmt.Errorf("command save: %w", err)
		}
		duration, err := strconv.ParseInt(rest[2], 10, 64)
		if err != nil {
			return fmt.Errorf("command save: invalid timestamp %q: %w", rest[2], err)
		}
		if err := c.d.Commands.SaveGame(ctx, gameID, playerID, duration); err != nil {
			return fmt.Errorf("command save: %w", err)
		}
		fmt.Fprintf(c.d.Stdout, "game %d saved by player %d for %d min\n", gameID, playerID, duration)
		return nil

	case "resume":
		if len(rest) != 2 {
			return wrongArgs("command resume", "command resume <game id> <player id>")
		}
		gameID, err := parseID(rest[0])
		if err != nil {
			return fmt.Errorf("command resume: %w", err)
		}
		playerID, err := parseID(rest[1])
		if err != nil {
			return fmt.Errorf("command resume: %w", err)
		}
		if err := c.d.Commands.ResumeGame(ctx, gameID, playerID); err != nil {
			return fmt.Errorf("command resume: %w", err)
		}
		fmt.Fprintf(c.d.Stdout, "game %d resumed by player %d\n", gameID, playerID)
		return nil

	case "pause":
		if len(rest) != 2 {
			return wrongArgs("command pause", "command pause <game id> <player id>")
		}
		gameID, err := parseID(rest[0])
		if err != nil {
			return fmt.Errorf("command pause: %w", err)
		}
		playerID, err := parseID(rest[1])
		if err != nil {
			return fmt.Errorf("command pause: %w", err)
		}
		if err := c.d.Commands.PauseGame(ctx, gameID, playerID); err != nil {
			return fmt.Errorf("command pause: %w", err)
		}
		fmt.Fprintf(c.d.Stdout, "game %d paused by player %d\n", gameID, playerID)
		return nil

	default:
		return fmt.Errorf("command %s: %w", verb, ErrUnknownSubcommand)
	}
}
