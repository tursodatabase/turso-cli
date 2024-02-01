package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/tursodatabase/turso-cli/internal/prompt"
)

var accountFeedbackCmd = &cobra.Command{
	Use:               "feedback",
	Short:             "Tell us how can we help you, how we can improve, or what you'd like to see next.",
	Args:              cobra.NoArgs,
	ValidArgsFunction: noFilesArg,
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true

		client, err := authedTursoClient()
		if err != nil {
			return fmt.Errorf("could not create turso client: %w", err)
		}

		summary, err := prompt.TextInput(
			"Could you summarize your feedback in a few words?",
			"Can we have a turso location in Helsinki?",
			"",
		)
		if err != nil {
			return err
		}

		feedback, err := prompt.TextArea(
			"Tell us more about it.",
			"",
			"",
		)
		if err != nil {
			return err
		}

		if summary == "" && feedback == "" {
			return fmt.Errorf("you must provide a summary or feedback")
		}

		summary = strings.TrimSpace(summary)
		feedback = strings.TrimSpace(feedback)

		if err := client.Feedback.Submit(summary, feedback); err != nil {
			return fmt.Errorf("error submitting feedback: %w", err)
		}

		fmt.Println("Thank you for your feedback! We'll get back to you as soon as possible.")
		return nil
	},
}
