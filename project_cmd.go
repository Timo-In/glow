package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

var projectCmd = &cobra.Command{
	Use:   "project",
	Short: "Manage and open Markdown projects",
	Args:  cobra.NoArgs,
	RunE: func(*cobra.Command, []string) error {
		store, err := configuredProjectStore()
		if err != nil {
			return err
		}
		projects, err := store.list()
		if err != nil {
			return err
		}
		if len(projects) == 0 {
			return fmt.Errorf("no projects configured; add one with %q", "glow project add NAME PATH")
		}

		selected, err := selectProject(projects)
		if err != nil {
			return err
		}
		if selected == nil {
			return nil
		}
		if err := validateProjectDirectory(selected.Path); err != nil {
			return err
		}
		return runTUI(selected.Path, "")
	},
}

var projectListCmd = &cobra.Command{
	Use:   "list",
	Short: "List configured Markdown projects",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, _ []string) error {
		store, err := configuredProjectStore()
		if err != nil {
			return err
		}
		projects, err := store.list()
		if err != nil {
			return err
		}
		if len(projects) == 0 {
			fmt.Fprintln(cmd.OutOrStdout(), "No projects configured.")
			return nil
		}
		for _, project := range projects {
			fmt.Fprintf(cmd.OutOrStdout(), "%s\t%s\n", project.Name, project.Path)
		}
		return nil
	},
}

var projectAddCmd = &cobra.Command{
	Use:   "add NAME PATH",
	Short: "Add a Markdown project",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		store, err := configuredProjectStore()
		if err != nil {
			return err
		}
		project, err := store.add(args[0], args[1])
		if err != nil {
			return err
		}
		fmt.Fprintf(cmd.OutOrStdout(), "Added project %q: %s\n", project.Name, project.Path)
		return nil
	},
}

var projectRemoveCmd = &cobra.Command{
	Use:               "remove NAME",
	Short:             "Remove a Markdown project",
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: completeProjectNames,
	RunE: func(cmd *cobra.Command, args []string) error {
		store, err := configuredProjectStore()
		if err != nil {
			return err
		}
		project, err := store.remove(args[0])
		if err != nil {
			return err
		}
		fmt.Fprintf(cmd.OutOrStdout(), "Removed project %q.\n", project.Name)
		return nil
	},
}

var projectOpenCmd = &cobra.Command{
	Use:               "open NAME",
	Short:             "Open a configured Markdown project",
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: completeProjectNames,
	RunE: func(_ *cobra.Command, args []string) error {
		store, err := configuredProjectStore()
		if err != nil {
			return err
		}
		project, err := store.find(args[0])
		if err != nil {
			return err
		}
		if err := validateProjectDirectory(project.Path); err != nil {
			return err
		}
		return runTUI(project.Path, "")
	},
}

func init() {
	projectCmd.AddCommand(projectListCmd, projectAddCmd, projectRemoveCmd, projectOpenCmd)
}

func completeProjectNames(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
	store, err := configuredProjectStore()
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}
	projects, err := store.list()
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}
	names := make([]string, 0, len(projects))
	for _, project := range projects {
		names = append(names, project.Name)
	}
	return names, cobra.ShellCompDirectiveNoFileComp
}
