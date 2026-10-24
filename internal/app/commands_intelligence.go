package app

import (
	"context"
	"errors"
	"io"
	"strconv"
	"strings"

	"github.com/WolfCanCode/personal-ai-brain/internal/intelligence"
	"github.com/WolfCanCode/personal-ai-brain/internal/memory"
)

// runProfileCmd detects and prints the project stack profile.
// It rejects any positional arguments since the profile command takes none.
func runProfileCmd(project memory.Project, args []string, out io.Writer) error {
	if len(args) != 0 {
		return errors.New("profile accepts no arguments")
	}
	v, err := intelligence.Profile(project.Root)
	if err != nil {
		return err
	}
	return printJSON(out, v)
}

// runAdviseCmd compiles a full intelligence pack for a task.
func runAdviseCmd(store *memory.Store, project memory.Project, args []string, out io.Writer) error {
	pos, f, err := flags(args, []string{"--budget"}, nil)
	if err != nil {
		return err
	}
	budget := 0
	if f["--budget"] != "" {
		budget, err = strconv.Atoi(f["--budget"])
		if err != nil {
			return errors.New("budget must be an integer")
		}
	}
	v, err := intelligence.Compile(store, project, strings.Join(pos, " "), budget)
	if err != nil {
		return err
	}
	return printJSON(out, v)
}

// runFeedbackCmd records whether a context pack was useful.
func runFeedbackCmd(store *memory.Store, args []string, out io.Writer) error {
	pos, f, err := flags(args, []string{"--outcome", "--tokens", "--notes"}, nil)
	if err != nil {
		return err
	}
	if len(pos) != 1 {
		return errors.New("feedback requires one context pack ID")
	}
	tokens := 0
	if f["--tokens"] != "" {
		tokens, err = strconv.Atoi(f["--tokens"])
		if err != nil {
			return errors.New("tokens must be an integer")
		}
	}
	v, err := intelligence.Feedback(store, pos[0], f["--outcome"], tokens, f["--notes"])
	if err != nil {
		return err
	}
	return printJSON(out, v)
}

func sources(store *memory.Store, args []string, out io.Writer) error {
	if len(args) == 0 {
		return errors.New("use sources seed|add|list|sync|pause")
	}
	sub := args[0]
	args = args[1:]
	switch sub {
	case "seed":
		if len(args) != 0 {
			return errors.New("sources seed accepts no arguments")
		}
		v, err := intelligence.SeedSources(store)
		if err != nil {
			return err
		}
		return printJSON(out, v)
	case "list":
		if len(args) != 0 {
			return errors.New("sources list accepts no arguments")
		}
		v, err := intelligence.ListSources(store)
		if err != nil {
			return err
		}
		return printJSON(out, v)
	case "add":
		pos, f, err := flags(args, []string{"--name", "--tags"}, nil)
		if err != nil {
			return err
		}
		if len(pos) != 1 {
			return errors.New("sources add requires one URL")
		}
		v, err := intelligence.AddSource(store, f["--name"], pos[0], f["--tags"])
		if err != nil {
			return err
		}
		return printJSON(out, v)
	case "sync":
		if len(args) != 0 {
			return errors.New("sources sync accepts no arguments")
		}
		v, err := intelligence.SyncSources(context.Background(), store)
		if err != nil {
			return err
		}
		return printJSON(out, v)
	case "pause":
		if len(args) != 1 {
			return errors.New("sources pause requires one source ID")
		}
		ok, err := intelligence.PauseSource(store, args[0])
		if err != nil {
			return err
		}
		if !ok {
			return errors.New("active source not found")
		}
		return printJSON(out, map[string]any{"paused": args[0]})
	}
	return errors.New("use sources seed|add|list|sync|pause")
}
