package app

import (
	"bufio"
	"fmt"
	"io"
	"strings"

	"github.com/WolfCanCode/personal-ai-brain/internal/intelligence"
	"github.com/charmbracelet/huh"
)

func promptConfirm(in io.Reader, out io.Writer, msg string) (bool, error) {
	if isTTY() {
		var ok bool
		err := huh.NewConfirm().
			Title(msg).
			Affirmative("Yes, overwrite").
			Negative("No, keep existing").
			Value(&ok).
			Run()
		return ok, err
	}
	// Non-TTY fallback
	fmt.Fprintf(out, "%s [y/N]: ", msg)
	scanner := bufio.NewScanner(in)
	if scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		return line == "y" || line == "Y", nil
	}
	return false, scanner.Err()
}

func promptSelection(in io.Reader, out io.Writer, catalog []intelligence.ProvisionableCapability, liveGroups []skillGroup) ([]intelligence.ProvisionableCapability, error) {
	all := make([]intelligence.ProvisionableCapability, 0, len(catalog)+20)
	all = append(all, catalog...)
	for _, g := range liveGroups {
		all = append(all, g.caps...)
	}
	if len(all) == 0 {
		return nil, nil
	}

	if isTTY() {
		return promptSelectionInteractive(catalog, liveGroups, all)
	}
	return promptSelectionText(in, out, catalog, liveGroups, all)
}

func promptSelectionInteractive(catalog []intelligence.ProvisionableCapability, liveGroups []skillGroup, all []intelligence.ProvisionableCapability) ([]intelligence.ProvisionableCapability, error) {
	// Build huh options with badge labels
	opts := make([]huh.Option[string], 0, len(all))

	// Section: built-in catalog
	if len(catalog) > 0 {
		for _, cap := range catalog {
			e := capEntry{Ref: cap.Ref, Kind: cap.Kind, Name: cap.Name, Network: string(cap.Network), Reason: cap.Reason}
			label := capOptionLabel(e)
			opts = append(opts, huh.NewOption(label, cap.Ref))
		}
	}
	// Sections: per-keyword groups
	for _, g := range liveGroups {
		for _, cap := range g.caps {
			e := capEntry{Ref: cap.Ref, Kind: cap.Kind, Name: cap.Name, Network: string(cap.Network), Reason: cap.Reason, InstallCount: cap.InstallCount}
			label := capOptionLabel(e)
			opts = append(opts, huh.NewOption(label, cap.Ref))
		}
	}

	var selectedRefs []string
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewMultiSelect[string]().
				Title(styleAccent.Render("Select capabilities to provision")).
				Description(styleMuted.Render("↑↓ navigate  ·  Space select  ·  Enter confirm  ·  a toggle all")).
				Options(opts...).
				Value(&selectedRefs),
		),
	).WithTheme(huh.ThemeDracula())

	if err := form.Run(); err != nil {
		return nil, err
	}

	refMap := map[string]intelligence.ProvisionableCapability{}
	for _, cap := range all {
		refMap[cap.Ref] = cap
	}
	selected := make([]intelligence.ProvisionableCapability, 0, len(selectedRefs))
	for _, ref := range selectedRefs {
		if cap, ok := refMap[ref]; ok {
			selected = append(selected, cap)
		}
	}
	return selected, nil
}

func promptSelectionText(in io.Reader, out io.Writer, catalog []intelligence.ProvisionableCapability, liveGroups []skillGroup, all []intelligence.ProvisionableCapability) ([]intelligence.ProvisionableCapability, error) {
	fmt.Fprintln(out, "\nAvailable capabilities:")
	if len(catalog) > 0 {
		fmt.Fprintln(out, "\n  Built-in recommendations:")
		for i, cap := range catalog {
			fmt.Fprintf(out, "    %2d. [%s][%s] %s\n        %s\n", i+1, cap.Kind, string(cap.Network), cap.Name, cap.Reason)
		}
	}
	offset := len(catalog)
	for _, g := range liveGroups {
		if len(g.caps) == 0 {
			continue
		}
		fmt.Fprintf(out, "\n  %s:\n", g.label)
		for i, cap := range g.caps {
			line := fmt.Sprintf("    %2d. [skill][network] %s", offset+i+1, cap.Name)
			if cap.InstallCount != "" {
				line += fmt.Sprintf(" · 📦 %s", cap.InstallCount)
			}
			fmt.Fprintln(out, line)
			fmt.Fprintf(out, "        %s\n", cap.Reason)
		}
		offset += len(g.caps)
	}
	fmt.Fprint(out, "\nSelect capabilities to provision (all/none/1,2,3): ")
	scanner := bufio.NewScanner(in)
	if !scanner.Scan() {
		return nil, scanner.Err()
	}
	line := strings.TrimSpace(scanner.Text())
	if line == "" || line == "none" {
		return nil, nil
	}
	if line == "all" {
		return all, nil
	}
	selected := []intelligence.ProvisionableCapability{}
	for _, part := range strings.Split(line, ",") {
		part = strings.TrimSpace(part)
		var idx int
		if _, err := fmt.Sscanf(part, "%d", &idx); err != nil || idx < 1 || idx > len(all) {
			fmt.Fprintf(out, "  Skipping invalid selection: %q\n", part)
			continue
		}
		selected = append(selected, all[idx-1])
	}
	return selected, nil
}
