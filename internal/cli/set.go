package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

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
			a := requireApp()
			p, err := a.Repo.Get(args[0])
			if err != nil {
				return err
			}
			if cmd.Flags().Changed("name") {
				p.Name = name
			}
			if cmd.Flags().Changed("editor") {
				p.Editor = editor
			}
			if cmd.Flags().Changed("profile") {
				p.Profile = profile
			}
			if cmd.Flags().Changed("group") {
				if group == "-" || group == "none" {
					p.Group = ""
				} else {
					p.Group = group
				}
			}
			if cmd.Flags().Changed("favorite") {
				p.Favorite = favorite
			}
			for _, t := range tags {
				p.AddTag(t)
			}
			for _, t := range untag {
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
