package smartcomplete

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// CompletionContext contains all context for a completion
type CompletionContext struct {
	Prefix              string
	Suffix              string
	AgentsInstructions  string
	DiscussionContext   string
	AdditionalFiles     []FileContext
	Language            string
}

// FileContext represents content from an additional file
type FileContext struct {
	Path    string
	Content string
}

// ContextGatherer collects relevant context for completions
type ContextGatherer struct {
	maxTokens int
}

// GatherContext collects all relevant context for the completion
func (g *ContextGatherer) GatherContext(
	req CompletionRequest,
	fileContent string,
	projectGetter ProjectGetter,
) (*CompletionContext, error) {
	baseDir, err := projectGetter.GetProjectBaseDir(req.ProjectID)
	if err != nil {
		return nil, err
	}

	// Extract prefix/suffix at cursor position
	prefix, suffix := extractPrefixSuffix(fileContent, req.CursorLine, req.CursorColumn)

	// Gather AGENTS.md instructions
	agentsInstructions := g.gatherAgentsInstructions(baseDir, req.FilePath, projectGetter)

	// Gather recent discussion context
	discussionContext := g.gatherDiscussionContext(req.ProjectID, projectGetter)

	// Gather additional context files
	additionalContext := g.gatherAdditionalFiles(req, baseDir, projectGetter)

	ctx := &CompletionContext{
		Prefix:              prefix,
		Suffix:              suffix,
		AgentsInstructions:  agentsInstructions,
		DiscussionContext:   discussionContext,
		AdditionalFiles:     additionalContext,
		Language:            detectLanguage(req.FilePath),
	}

	// Trim to fit within token budget
	g.trimToTokenBudget(ctx)

	return ctx, nil
}

// extractPrefixSuffix splits file content at cursor position
func extractPrefixSuffix(content string, line, col int) (prefix, suffix string) {
	lines := strings.Split(content, "\n")
	
	if line < 0 {
		line = 0
	}
	if line >= len(lines) {
		line = len(lines) - 1
	}
	if col < 0 {
		col = 0
	}

	// Prefix: everything before cursor
	prefixLines := lines[:line]
	if line < len(lines) && col < len(lines[line]) {
		prefixLines = append(prefixLines, lines[line][:col])
	}
	prefix = strings.Join(prefixLines, "\n")

	// Suffix: everything after cursor
	var suffixLines []string
	if line < len(lines) && col < len(lines[line]) {
		suffixLines = append(suffixLines, lines[line][col:])
	}
	if line+1 < len(lines) {
		suffixLines = append(suffixLines, lines[line+1:]...)
	}
	suffix = strings.Join(suffixLines, "\n")

	return prefix, suffix
}

// gatherAgentsInstructions finds and reads AGENTS.md files
func (g *ContextGatherer) gatherAgentsInstructions(
	baseDir, targetFile string,
	projectGetter ProjectGetter,
) string {
	dir := filepath.Dir(filepath.Join(baseDir, targetFile))
	var instructions []string

	for {
		agentsPath := filepath.Join(dir, "AGENTS.md")
		if content, err := projectGetter.ReadFile(agentsPath); err == nil {
			instructions = append(instructions, string(content))
		}

		if dir == baseDir || dir == "/" || dir == "." {
			break
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	if len(instructions) == 0 {
		return ""
	}

	return strings.Join(instructions, "\n\n---\n\n")
}

// gatherDiscussionContext extracts recent discussion rounds
func (g *ContextGatherer) gatherDiscussionContext(
	projectID string,
	projectGetter ProjectGetter,
) string {
	discussionFile, err := projectGetter.GetProjectDiscussionFile(projectID)
	if err != nil {
		return ""
	}

	content, err := projectGetter.ReadFile(discussionFile)
	if err != nil {
		return ""
	}

	// Extract last 3000 characters as simplified approach
	str := string(content)
	if len(str) > 3000 {
		str = str[len(str)-3000:]
	}

	return str
}

// gatherAdditionalFiles collects context from additional files
func (g *ContextGatherer) gatherAdditionalFiles(
	req CompletionRequest,
	baseDir string,
	projectGetter ProjectGetter,
) []FileContext {
	var contexts []FileContext

	for _, filePath := range req.ContextFiles {
		absPath := resolveFilePath(baseDir, filePath)
		content, err := projectGetter.ReadFile(absPath)
		if err != nil {
			continue
		}

		contexts = append(contexts, FileContext{
			Path:    filePath,
			Content: string(content),
		})
	}

	return contexts
}

// trimToTokenBudget ensures context fits within token budget
func (g *ContextGatherer) trimToTokenBudget(ctx *CompletionContext) {
	// Simplified: estimate tokens as ~4 chars per token
	estimateTokens := func(s string) int {
		return len(s) / 4
	}

	currentTokens := estimateTokens(ctx.Prefix) + 
		estimateTokens(ctx.Suffix) +
		estimateTokens(ctx.AgentsInstructions) +
		estimateTokens(ctx.DiscussionContext)

	for _, f := range ctx.AdditionalFiles {
		currentTokens += estimateTokens(f.Content)
	}

	if currentTokens <= g.maxTokens {
		return
	}

	// Priority: Keep prefix/suffix, trim discussion and agents
	if estimateTokens(ctx.DiscussionContext) > 1000 {
		ctx.DiscussionContext = ctx.DiscussionContext[len(ctx.DiscussionContext)-1000:]
	}
	if estimateTokens(ctx.AgentsInstructions) > 2000 {
		ctx.AgentsInstructions = ctx.AgentsInstructions[:2000]
	}
}

// detectLanguage infers programming language from file extension
func detectLanguage(filePath string) string {
	ext := strings.ToLower(filepath.Ext(filePath))
	langMap := map[string]string{
		".go":   "Go",
		".py":   "Python",
		".js":   "JavaScript",
		".ts":   "TypeScript",
		".java": "Java",
		".c":    "C",
		".cpp":  "C++",
		".rs":   "Rust",
		".rb":   "Ruby",
		".php":  "PHP",
		".sh":   "Shell",
	}

	if lang, ok := langMap[ext]; ok {
		return lang
	}
	return "code"
}
