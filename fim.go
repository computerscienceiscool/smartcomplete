package smartcomplete

import (
	"fmt"
	"strings"
)

// FIMFormatter formats Fill-in-Middle prompts
type FIMFormatter struct {
	instructionTemplate string
}

// FormatPrompt creates a FIM prompt from context
func (f *FIMFormatter) FormatPrompt(ctx *CompletionContext) string {
	var prompt strings.Builder

	// System instructions
	prompt.WriteString(fmt.Sprintf(
		"You are an expert %s programmer. Complete the code at the cursor position.\n\n",
		ctx.Language,
	))

	// AGENTS.md instructions (if present)
	if ctx.AgentsInstructions != "" {
		prompt.WriteString("PROJECT INSTRUCTIONS:\n")
		prompt.WriteString(ctx.AgentsInstructions)
		prompt.WriteString("\n\n")
	}

	// Recent discussion context (if present)
	if ctx.DiscussionContext != "" {
		prompt.WriteString("RECENT PROJECT DISCUSSION:\n")
		prompt.WriteString(ctx.DiscussionContext)
		prompt.WriteString("\n\n")
	}

	// Additional context files
	if len(ctx.AdditionalFiles) > 0 {
		prompt.WriteString("RELATED FILES:\n")
		for _, file := range ctx.AdditionalFiles {
			prompt.WriteString(fmt.Sprintf("\n--- %s ---\n%s\n", file.Path, file.Content))
		}
		prompt.WriteString("\n")
	}

	// Main FIM prompt
	prompt.WriteString("CODE BEFORE CURSOR:\n")
	prompt.WriteString(ctx.Prefix)
	prompt.WriteString("\n\n")

	prompt.WriteString("CODE AFTER CURSOR:\n")
	prompt.WriteString(ctx.Suffix)
	prompt.WriteString("\n\n")

	prompt.WriteString("INSTRUCTIONS:\n")
	prompt.WriteString("Complete only the code at the cursor position.\n")
	prompt.WriteString("Provide syntactically correct, idiomatic " + ctx.Language + " code.\n")
	prompt.WriteString("Do not repeat the prefix or suffix.\n")
	prompt.WriteString("Output only the completion, nothing else.\n")

	return prompt.String()
}
