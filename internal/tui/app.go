package tui

import (
	"context"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	
	"github.com/gaurav3000R/neuron-cli/internal/llm"
)

var (
	titleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFDF5")).
			Background(lipgloss.Color("#25A065")).
			Padding(0, 1)

	userRoleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#569CD6")).
			Bold(true).
			MarginTop(1)

	assistantRoleStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#4EC9B0")).
				Bold(true).
				MarginTop(1)

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#F44336")).
			Bold(true)
)

// tokenMsg is sent when a new token arrives from the LLM stream
type tokenMsg string

// streamDoneMsg is sent when the LLM stream completes successfully
type streamDoneMsg struct{}

// streamErrorMsg is sent when the LLM stream encounters an error
type streamErrorMsg struct{ err error }

// chatModel implements tea.Model
type chatModel struct {
	viewport    viewport.Model
	textarea    textarea.Model
	
	messages    []llm.Message
	provider    llm.Provider
	modelID     string
	
	isStreaming bool
	currentGen  strings.Builder
	
	err         error
	ctx         context.Context
	cancel      context.CancelFunc
}

// InitialModel configures the starting state of the TUI.
func InitialModel(provider llm.Provider, modelID string) chatModel {
	ta := textarea.New()
	ta.Placeholder = "Ask Neuron... (Ctrl+D to send, Ctrl+C to quit)"
	ta.Focus()
	ta.Prompt = "> "
	ta.CharLimit = 4096
	ta.SetWidth(80)
	ta.SetHeight(3)
	ta.FocusedStyle.CursorLine = lipgloss.NewStyle()
	ta.ShowLineNumbers = false
	ta.KeyMap.InsertNewline.SetEnabled(false) // We want Enter to submit, Shift+Enter for newline (not easily configurable without custom keymap, but we'll use Ctrl+D for send conventionally, or hijack Enter).

	vp := viewport.New(80, 20)
	vp.SetContent("Welcome to Neuron CLI.\nType a message to begin.")

	ctx, cancel := context.WithCancel(context.Background())

	return chatModel{
		textarea: ta,
		viewport: vp,
		provider: provider,
		modelID:  modelID,
		messages: make([]llm.Message, 0),
		ctx:      ctx,
		cancel:   cancel,
	}
}

func (m *chatModel) Init() tea.Cmd {
	return textarea.Blink
}

func (m *chatModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		tiCmd tea.Cmd
		vpCmd tea.Cmd
	)

	m.textarea, tiCmd = m.textarea.Update(msg)
	m.viewport, vpCmd = m.viewport.Update(msg)

	switch msg := msg.(type) {
	
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			return m, tea.Quit
		
		// Map 'Enter' to submit the prompt instead of newline
		case tea.KeyEnter:
			if m.isStreaming {
				return m, nil // Ignore input while generating
			}
			text := strings.TrimSpace(m.textarea.Value())
			if text == "" {
				return m, nil
			}
			
			// Reset textarea
			m.textarea.Reset()
			
			// Add user message to history
			m.messages = append(m.messages, llm.Message{
				Role:    llm.RoleUser,
				Content: text,
			})
			
			// Kick off LLM generation
			m.isStreaming = true
			m.currentGen.Reset()
			m.err = nil
			m.updateViewport()
			
			return m, m.generateResponse()
		}

	case tea.WindowSizeMsg:
		m.viewport.Width = msg.Width
		m.viewport.Height = msg.Height - m.textarea.Height() - 3 // Leave room for textarea and title
		m.textarea.SetWidth(msg.Width)
		m.updateViewport()

	case tokenResponseMsg:
		// Append token to current generation and re-render
		m.currentGen.WriteString(msg.token)
		m.updateViewport()
		m.viewport.GotoBottom()
		// Recursively wait for the next token using the provided channels
		return m, waitForNextToken(msg.tokenChan, msg.errChan)

	case streamDoneMsg:
		m.isStreaming = false
		// Save completed generation to history
		m.messages = append(m.messages, llm.Message{
			Role:    llm.RoleAssistant,
			Content: m.currentGen.String(),
		})
		m.currentGen.Reset()
		m.updateViewport()
		m.viewport.GotoBottom()
		return m, nil

	case streamErrorMsg:
		m.isStreaming = false
		m.err = msg.err
		m.currentGen.Reset()
		m.updateViewport()
		m.viewport.GotoBottom()
		return m, nil
	}

	return m, tea.Batch(tiCmd, vpCmd)
}

// generateResponse invokes the LLM Provider and returns the initial token pipeline command
func (m *chatModel) generateResponse() tea.Cmd {
	req := llm.CompletionRequest{
		Model:    m.modelID,
		Messages: m.messages,
		Stream:   true,
	}

	tokenChan, errChan := m.provider.GenerateStream(m.ctx, req)

	// We return a command that reads the first token from the channel
	return waitForNextToken(tokenChan, errChan)
}

// waitForNextToken is recursively called after each token update to pull the next
// token from the channel into the Bubbletea event loop.
func waitForNextToken(tokenChan <-chan string, errChan <-chan error) tea.Cmd {
	return func() tea.Msg {
		select {
		case err := <-errChan:
			if err != nil {
				return streamErrorMsg{err}
			}
			return streamDoneMsg{}
		case token, ok := <-tokenChan:
			if !ok {
				// Channel closed, check for final errors
				select {
				case err := <-errChan:
					if err != nil {
						return streamErrorMsg{err}
					}
				default:
				}
				return streamDoneMsg{}
			}
			// We received a token, we must return a command containing the token AND the channels 
			// so the Update loop can recurse.
			return tokenResponseMsg{
				token:     token,
				tokenChan: tokenChan,
				errChan:   errChan,
			}
		}
	}
}

type tokenResponseMsg struct {
	token     string
	tokenChan <-chan string
	errChan   <-chan error
}

func (m *chatModel) updateViewport() {
	var sb strings.Builder

	// Render conversation history
	for _, msg := range m.messages {
		if msg.Role == llm.RoleUser {
			sb.WriteString(userRoleStyle.Render("You\n"))
		} else {
			sb.WriteString(assistantRoleStyle.Render("Neuron\n"))
		}
		sb.WriteString(msg.Content + "\n")
	}

	// Render the currently generating stream
	if m.isStreaming {
		sb.WriteString(assistantRoleStyle.Render("Neuron\n"))
		sb.WriteString(m.currentGen.String() + "█\n")
	}

	// Render any stream errors
	if m.err != nil {
		sb.WriteString(errorStyle.Render(fmt.Sprintf("\n[Error: %v]\n", m.err)))
	}

	m.viewport.SetContent(sb.String())
}

func (m *chatModel) View() string {
	header := titleStyle.Render(fmt.Sprintf(" Neuron CLI (%s) ", m.modelID))
	
	historyView := m.viewport.View()
	
	inputLabel := "  "
	if m.isStreaming {
		inputLabel = "░ " 
	}
	
	return fmt.Sprintf(
		"%s\n%s\n\n%s%s",
		header,
		historyView,
		inputLabel,
		m.textarea.View(),
	)
}

// Run is the main entrypoint called by the Cobra command.
func Run(provider llm.Provider, modelID string) error {
	m := InitialModel(provider, modelID)
	p := tea.NewProgram(&m, tea.WithAltScreen())
	_, err := p.Run()
	return err
}
