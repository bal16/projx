package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

type setParams struct {
	name       string
	editor     string
	profile    string
	group      string
	favorite   bool
	favSet     bool
	tags       []string
	untag      []string
	nameSet    bool
	editorSet  bool
	profileSet bool
	groupSet   bool
}

func runSet(a *App, name string, sp setParams) error {
	p, err := a.Repo.Get(name)
	if err != nil {
		return err
	}
	if sp.nameSet {
		p.Name = sp.name
	}
	if sp.editorSet {
		p.Editor = sp.editor
	}
	if sp.profileSet {
		if sp.profile == "" || sp.profile == "-" || sp.profile == "none" {
			p.Profile = ""
		} else {
			p.Profile = sp.profile
			if p.Editor == "code" || p.Editor == "codium" || p.Editor == "cursor" {
				found := false
				for _, vp := range listVSCodeProfiles() {
					if vp.Name == sp.profile || vp.ID == sp.profile {
						found = true
						break
					}
				}
				if !found {
					fmt.Printf("warning: profile %q not found among known VS Code profiles (will be created on open)\n", sp.profile)
				}
			}
		}
	}
	if sp.groupSet {
		if sp.group == "-" || sp.group == "none" {
			p.Group = ""
		} else {
			p.Group = sp.group
		}
	}
	if sp.favSet {
		p.Favorite = sp.favorite
	}
	for _, t := range sp.tags {
		p.AddTag(t)
	}
	for _, t := range sp.untag {
		p.RemoveTag(t)
	}
	if err := a.Repo.Update(p); err != nil {
		return err
	}
	fmt.Printf("updated %s\n", p.Name)
	if p.Editor != "" {
		fmt.Printf("  editor:  %s\n", p.Editor)
	}
	if p.Profile != "" {
		fmt.Printf("  profile: %s\n", p.Profile)
	}
	if len(p.Tags) > 0 {
		fmt.Printf("  tags:    %s\n", joinStrings(p.Tags, ", "))
	}
	return nil
}

func newSetCmd() *cobra.Command {
	var name, editor, profile, group string
	var favorite bool
	var tags []string
	var untag []string

	cmd := &cobra.Command{
		Use:   "set <project>",
		Short: "Update project fields (editor, profile, name, group, favorite, tags)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSet(requireApp(), args[0], setParams{
				name:       name,
				editor:     editor,
				profile:    profile,
				group:      group,
				favorite:   favorite,
				favSet:     cmd.Flags().Changed("favorite"),
				tags:       tags,
				untag:      untag,
				nameSet:    cmd.Flags().Changed("name"),
				editorSet:  cmd.Flags().Changed("editor"),
				profileSet: cmd.Flags().Changed("profile"),
				groupSet:   cmd.Flags().Changed("group"),
			})
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "new display name")
	cmd.Flags().StringVar(&editor, "editor", "", "preferred editor (zed, code, nvim, ...)")
	cmd.Flags().StringVar(&profile, "profile", "", "editor profile (VS Code profiles); empty clears")
	cmd.Flags().StringVar(&group, "group", "", "group name ('-' or 'none' to clear)")
	cmd.Flags().BoolVar(&favorite, "favorite", false, "mark as favorite (use --favorite=false to unset)")
	cmd.Flags().StringSliceVar(&tags, "tag", nil, "add tag (repeatable)")
	cmd.Flags().StringSliceVar(&untag, "untag", nil, "remove tag (repeatable)")
	return cmd
}
