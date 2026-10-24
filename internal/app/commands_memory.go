package app

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/WolfCanCode/personal-ai-brain/internal/intelligence"
	"github.com/WolfCanCode/personal-ai-brain/internal/memory"
)

// runInitCmd initialises (or re-initialises) the memory store.
func runInitCmd(store *memory.Store, args []string, home string, out, errOut io.Writer) error {
	pos, f, err := flags(args, nil, []string{"--schedule"})
	if err != nil {
		return err
	}
	if len(pos) > 0 {
		return errors.New("init accepts only --schedule")
	}
	if isTTY() {
		printBanner()
	}
	// Auto-seed the embedded AAS skills catalog on first init.
	// Report in the JSON result (not as bare stdout) so smoke tests
	// and MCP clients can still JSON.parse the init output cleanly.
	catalogSeeded := 0
	if n, seedErr := intelligence.SeedEmbeddedCatalog(store); seedErr != nil {
		fmt.Fprintf(errOut, "brain: catalog seed: %v\n", seedErr)
	} else {
		catalogSeeded = n
	}
	result := map[string]any{
		"database":  store.Path,
		"scheduler": schedulerStatus(),
	}
	if catalogSeeded > 0 {
		result["catalogSeeded"] = catalogSeeded
	}
	if f["--schedule"] != "" {
		v, e := installScheduler(home)
		if e != nil {
			return e
		}
		result["scheduler"] = v
	}
	conf, err := mcpConfig()
	if err != nil {
		return err
	}
	result["mcp"] = conf
	return printJSON(out, result)
}

// runRememberCmd stores a verified project memory.
func runRememberCmd(store *memory.Store, project memory.Project, args []string, out io.Writer) error {
	pos, f, err := flags(args, []string{"--scope", "--type"}, nil)
	if err != nil {
		return err
	}
	v, err := store.Remember(memory.Input{
		Content:   strings.Join(pos, " "),
		Scope:     f["--scope"],
		Type:      f["--type"],
		ProjectID: project.ID,
	})
	if err != nil {
		return err
	}
	return printJSON(out, v)
}

// runQueryCmd handles both "recall" and "context" with a shared flag set.
func runQueryCmd(cmd string, store *memory.Store, project memory.Project, args []string, out io.Writer) error {
	pos, _, err := flags(args, nil, nil)
	if err != nil {
		return err
	}
	q := strings.Join(pos, " ")
	var v any
	if cmd == "recall" {
		v, err = store.Recall(q, project.ID, 8)
	} else {
		v, err = store.Context(q, project.ID, 8)
	}
	if err != nil {
		return err
	}
	return printJSON(out, v)
}

// runApproveCmd promotes a candidate memory to verified.
func runApproveCmd(store *memory.Store, args []string, out io.Writer) error {
	if len(args) != 1 {
		return errors.New("approve requires one memory ID")
	}
	ok, err := store.Approve(args[0])
	if err != nil {
		return err
	}
	if !ok {
		return errors.New("candidate memory not found")
	}
	return printJSON(out, map[string]any{"approved": args[0]})
}

// runStatusCmd prints memory counts, scheduler state, and active knowledge.
func runStatusCmd(store *memory.Store, project memory.Project, out io.Writer) error {
	v, err := store.Stats()
	if err != nil {
		return err
	}
	v["project"] = project
	v["scheduler"] = schedulerStatus()
	if count, countErr := intelligence.KnowledgeCount(store); countErr == nil {
		v["activeKnowledge"] = count
	}
	return printJSON(out, v)
}

// runDigestCmd prints recent memories and events.
func runDigestCmd(store *memory.Store, args []string, out io.Writer) error {
	pos, f, err := flags(args, []string{"--days"}, nil)
	if err != nil {
		return err
	}
	if len(pos) > 0 {
		return errors.New("digest accepts only --days")
	}
	days := 1
	if f["--days"] != "" {
		days, err = strconv.Atoi(f["--days"])
		if err != nil {
			return errors.New("days must be an integer")
		}
	}
	v, err := store.Digest(days)
	if err != nil {
		return err
	}
	return printJSON(out, v)
}

// runSyncCmd runs local maintenance and optionally syncs sources and skills.
func runSyncCmd(store *memory.Store, args []string, out io.Writer) error {
	pos, f, err := flags(args, nil, []string{"--sources", "--skills", "--online"})
	if err != nil {
		return err
	}
	if len(pos) > 0 {
		return errors.New("sync accepts only --sources, --skills or --online")
	}
	v, err := store.Sync()
	if err != nil {
		return err
	}
	if f["--sources"] != "" || f["--online"] != "" {
		srcs, sourceErr := intelligence.SyncSources(context.Background(), store)
		v["sources"] = srcs
		if sourceErr != nil {
			v["sourceError"] = sourceErr.Error()
		}
	}
	if f["--skills"] != "" || f["--online"] != "" {
		r, e := external("npx", []string{"--yes", "skills", "check"})
		v["skillCheck"] = map[string]any{"ok": e == nil, "output": r}
		if e != nil {
			v["skillCheckError"] = e.Error()
		}
	}
	return printJSON(out, v)
}
