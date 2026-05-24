package prompt

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"aseek-orchestrator/internal/logging"
	"aseek-orchestrator/internal/pipeline"
)

type Builder struct {
	templatesDir string
	log          *logging.Logger
	templates    map[string]string
}

func New(templatesDir string, log *logging.Logger) *Builder {
	return &Builder{
		templatesDir: templatesDir,
		log:          log.WithModule("prompt"),
		templates:    make(map[string]string),
	}
}

func (b *Builder) LoadTemplates() error {
	names := []string{"system.txt", "rag.txt", "no_results.txt"}

	for _, name := range names {
		path := filepath.Join(b.templatesDir, name)
		data, err := os.ReadFile(path)
		if err != nil {
			b.log.Warn("template not found", "name", name, "error", err)
			continue
		}
		b.templates[name] = string(data)
	}

	return nil
}

func (b *Builder) BuildSystemPrompt(ctx context.Context) (string, error) {
	if tmpl, ok := b.templates["system.txt"]; ok {
		return tmpl, nil
	}
	return "Ты — полезный ассистент.", nil
}

func (b *Builder) BuildRAGPrompt(ctx context.Context, query string, docs []pipeline.Document) (string, error) {
	if len(docs) == 0 {
		if tmpl, ok := b.templates["no_results.txt"]; ok {
			return tmpl, nil
		}
		return fmt.Sprintf("Вопрос: %s\n\nОтвет: Не удалось найти релевантные документы.", query), nil
	}

	if tmpl, ok := b.templates["rag.txt"]; ok {
		return b.renderTemplate(tmpl, query, docs)
	}

	return b.buildDefaultRAG(query, docs), nil
}

func (b *Builder) buildDefaultRAG(query string, docs []pipeline.Document) string {
	prompt := "Ты отвечаешь только по предоставленным документам.\n\nДокументы:\n\n"
	for _, d := range docs {
		prompt += fmt.Sprintf("[%d]\n%s\n[/%d]\n\n", d.Index, d.Content, d.Index)
	}
	prompt += fmt.Sprintf("Вопрос:\n%s\n\nОтвет:\n", query)
	return prompt
}

func (b *Builder) renderTemplate(tmpl string, query string, docs []pipeline.Document) (string, error) {
	if !strings.Contains(tmpl, "{{.Query}}") && !strings.Contains(tmpl, "{{.Results}}") {
		return "", fmt.Errorf("template missing {{.Query}} or {{.Results}} variables")
	}

	result := strings.ReplaceAll(tmpl, "{{.Query}}", query)

	var docsBuilder strings.Builder
	for _, d := range docs {
		docsBuilder.WriteString(fmt.Sprintf("[%d]\n%s\n[/%d]\n\n", d.Index, d.Content, d.Index))
	}
	result = strings.ReplaceAll(result, "{{.Results}}", docsBuilder.String())

	return result, nil
}