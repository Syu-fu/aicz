package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"html/template"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/manifoldco/promptui"
	"github.com/sashabaranov/go-openai"
)

type commitType struct {
	Type        string
	Description string
}

type input struct {
	Type             string
	Scope            string
	IsBreakingChange bool
	Description      string
	Reason           string
	IssueNumber      string
}

func main() {
	var input input

	types := []commitType{
		{Type: "feat", Description: "feature"},
		{Type: "fix", Description: "bug fix"},
		{Type: "deps", Description: "dependencies"},
		{Type: "breaking", Description: "breaking changes"},
		{Type: "docs", Description: "documentation"},
		{Type: "style", Description: "formatting, missing semi colons, etc"},
		{Type: "refactor", Description: "refactoring"},
		{Type: "test", Description: "when adding missing tests"},
		{Type: "chore", Description: "maintain"},
	}

	templates := &promptui.SelectTemplates{
		Label:    "{{ . }}?",
		Active:   "> {{ .Type | cyan }}: {{ .Description }}",
		Inactive: "  {{ .Type | cyan }}: {{ .Description }}",
		Selected: "> {{ .Type | red | cyan }}",
	}

	searcher := func(input string, index int) bool {
		ctype := types[index]
		name := strings.Replace(strings.ToLower(ctype.Type), " ", "", -1)
		input = strings.Replace(strings.ToLower(input), " ", "", -1)

		return strings.Contains(name, input)
	}

	ctPrompt := promptui.Select{
		Label:     "Spicy Level",
		Items:     types,
		Templates: templates,
		Size:      4,
		Searcher:  searcher,
	}

	i, _, err := ctPrompt.Run()
	if err != nil {
		fmt.Printf("Prompt failed %v\n", err)
		return
	}
	input.Type = types[i].Type

	brPrompt := promptui.Prompt{
		Label:     "has breaking changes",
		IsConfirm: true,
	}
	_, err = brPrompt.Run()
	input.IsBreakingChange = true
	if err != nil {
		input.IsBreakingChange = false
	}

	scPrompt := promptui.Prompt{
		Label: "scope(optional)",
	}
	input.Scope, err = scPrompt.Run()
	if err != nil {
		fmt.Printf("Prompt failed %v\n", err)
		return
	}

	sbPrompt := promptui.Prompt{
		Label: "subject",
	}
	input.Description, err = sbPrompt.Run()
	if err != nil {
		fmt.Printf("Prompt failed %v\n", err)
		return
	}

	rsPrompt := promptui.Prompt{
		Label: "reason(optional)",
	}
	input.Reason, err = rsPrompt.Run()
	if err != nil {
		fmt.Printf("Prompt failed %v\n", err)
		return
	}

	issuePrompt := promptui.Prompt{
		Label:    "resolve issue number(optional)",
		Validate: issueNumberValidate,
	}
	input.IssueNumber, err = issuePrompt.Run()
	if err != nil {
		fmt.Printf("Prompt failed %v\n", err)
		return
	}
	if input.IssueNumber != "" {
		input.Description = input.Description + "(#" + input.IssueNumber + ")"
	}

	t, err := template.New("aiprompt.tmpl").ParseFiles("aiprompt.tmpl")
	if err != nil {
		fmt.Printf("Error parsing template: %v\n", err)
		return
	}
	var aiPrompt bytes.Buffer
	_ = t.Execute(&aiPrompt, input)

	OPENAI_KEY := os.Getenv("OPENAI_KEY")
	client := openai.NewClient(OPENAI_KEY)
	resp, err := client.CreateChatCompletion(
		context.Background(),
		openai.ChatCompletionRequest{
			Model: openai.GPT3Dot5Turbo,
			Messages: []openai.ChatCompletionMessage{
				{
					Role:    openai.ChatMessageRoleUser,
					Content: aiPrompt.String(),
				},
			},
		},
	)
	if err != nil {
		fmt.Printf("ChatCompletion error: %v\n", err)
		return
	}

	fmt.Println(resp.Choices[0].Message.Content)

	conPrompt := promptui.Prompt{
		Label:     "Commit this message",
		IsConfirm: true,
	}
	_, err = conPrompt.Run()
	if err != nil {
		return
	}

	cmdResult, err := exec.Command("git", "commit", "-m", resp.Choices[0].Message.Content).CombinedOutput()
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(string(cmdResult))
}

func allowEmptyValidate(input string) error {
	if input == "" {
		return nil
	}
	return errors.New("Invalid input")
}

func numberValidate(input string) error {
	_, err := strconv.Atoi(input)
	if err != nil {
		return errors.New("Invalid number")
	}
	return nil
}

func issueNumberValidate(input string) error {
	if allowEmptyValidate(input) == nil {
		return nil
	}
	if numberValidate(input) == nil {
		return nil
	}
	return errors.New("Invalid issue number")
}
