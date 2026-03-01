package tui

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/gaurav3000R/neuron-cli/internal/llm"
	"github.com/gaurav3000R/neuron-cli/internal/tools"
)

var (
	titleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFDF5")).
			Background(lipgloss.Color("#25A065")).
			Padding(0, 1).
			Bold(true)

	userStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#00D9FF")).
			Bold(true)

	assistantStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#50FA7B")).
				Bold(true)

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF5555")).
			Bold(true)

	dimStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#6272A4"))

	toolStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFB86C")).
			Italic(true)
)

// Messages for bubbletea
type tokenMsg string
type streamDoneMsg struct{}
type streamErrorMsg struct{ err error }
type toolCallMsg struct {
	ToolName string
	Args     json.RawMessage
}
type toolResultMsg struct {
	Result string
	Error  error
}

// chatModel implements tea.Model
type chatModel struct {
	viewport viewport.Model
	textarea textarea.Model
	spinner  spinner.Model

	messages []llm.Message
	provider llm.Provider
	modelID  string
	registry *tools.Registry

	isStreaming    bool
	currentGen     strings.Builder
	chatHistory    strings.Builder
	pendingToolMsg *toolCallMsg

	tokenChan <-chan string
	errChan   <-chan error

	err    error
	ctx    context.Context
	cancel context.CancelFunc
}

// InitialModel configures the starting state of the TUI.
func InitialModel(provider llm.Provider, modelID string, registry *tools.Registry) *chatModel {
	ta := textarea.New()
	ta.Placeholder = "Type your message and press Enter..."
	ta.Focus()
	ta.Prompt = "│ "
	ta.CharLimit = 8192
	ta.SetWidth(80)
	ta.SetHeight(3)
	ta.FocusedStyle.CursorLine = lipgloss.NewStyle()
	ta.ShowLineNumbers = false

	vp := viewport.New(80, 20)

	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF79C6"))

	ctx, cancel := context.WithCancel(context.Background())

	m := &chatModel{
		textarea: ta,
		viewport: vp,
		spinner:  sp,
		provider: provider,
		modelID:  modelID,
		registry: registry,
		messages: make([]llm.Message, 0),
		ctx:      ctx,
		cancel:   cancel,
	}

	m.addToChatHistory(dimStyle.Render("╭─ Welcome to Neuron CLI ─╮"))
	m.addToChatHistory(dimStyle.Render(fmt.Sprintf("│ Model: %s", modelID)))
	m.addToChatHistory(dimStyle.Render(fmt.Sprintf("│ Tools: %d available", len(registry.GetAll()))))
	m.addToChatHistory(dimStyle.Render("╰───────────────────────────╯"))
	m.addToChatHistory("")
	m.updateViewport()

	return m
}

func (m *chatModel) Init() tea.Cmd {
	return tea.Batch(textarea.Blink, m.spinner.Tick)
}

func (m *chatModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		tiCmd tea.Cmd
		vpCmd tea.Cmd
		spCmd tea.Cmd
	)

	m.textarea, tiCmd = m.textarea.Update(msg)
	m.viewport, vpCmd = m.viewport.Update(msg)
	m.spinner, spCmd = m.spinner.Update(msg)

	switch msg := msg.(type) {

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			m.cancel()
			return m, tea.Quit

		case "enter":
			if m.isStreaming {
				return m, nil
			}
			text := strings.TrimSpace(m.textarea.Value())
			if text == "" {
				return m, nil
			}

			m.textarea.Reset()
			m.handleUserMessage(text)
			return m, tea.Batch(m.generateResponse(), m.spinner.Tick)
		}

	case tea.WindowSizeMsg:
		m.viewport.Width = msg.Width
		headerHeight := 4
		footerHeight := 6
		m.viewport.Height = msg.Height - headerHeight - footerHeight
		m.textarea.SetWidth(msg.Width - 4)
		m.updateViewport()

	case tokenMsg:
		token := string(msg)
		
		// Log the exact token for debugging  
		prefix := token
		if len(token) > 20 {
			prefix = token[:20]
		}
		slog.Info("TUI received token", "token_prefix", prefix, "length", len(token))
		
		// Check for tool call marker using strings.HasPrefix
		if strings.HasPrefix(token, "__TOOL_CALL__") {
			slog.Info("TUI detected tool call marker - processing")
			toolCallJSON := strings.TrimPrefix(token, "__TOOL_CALL__")
			slog.Debug("Tool call JSON", "json", toolCallJSON)
			
			var toolCalls []llm.ToolCall
			if err := json.Unmarshal([]byte(toolCallJSON), &toolCalls); err != nil {
				slog.Error("Failed to parse tool calls", "error", err, "json", toolCallJSON)
				m.addToChatHistory(errorStyle.Render(fmt.Sprintf("Error parsing tool calls: %v", err)))
				m.updateViewport()
				return m, nil
			}
			
			// Execute the first tool call (handle multiple later)
			if len(toolCalls) > 0 {
				tc := toolCalls[0]
				slog.Info("Executing tool", "name", tc.Function.Name, "args", string(tc.Function.Arguments))
				
				// Add tool call to message history
				m.messages = append(m.messages, llm.Message{
					Role:     llm.RoleAssistant,
					Content:  "",
					ToolCalls: toolCalls,
				})
				
				return m, m.executeTool(toolCallMsg{
					ToolName: tc.Function.Name,
					Args:     tc.Function.Arguments,
				})
			}
			return m, nil
		}
		
		// Regular token - add to output
		m.currentGen.WriteString(token)
		m.updateViewport()
		return m, m.waitForNextToken()

	case streamDoneMsg:
		slog.Info("TUI received streamDone")
		content := m.currentGen.String()
		slog.Info("Final content", "content", content, "length", len(content))
		if content != "" {
			m.messages = append(m.messages, llm.Message{
				Role:    llm.RoleAssistant,
				Content: content,
			})
			m.addToChatHistory("")
			m.addToChatHistory(assistantStyle.Render("Assistant:"))
			m.addToChatHistory(content)
		}
		m.isStreaming = false
		m.currentGen.Reset()
		m.updateViewport()
		return m, nil

	case streamErrorMsg:
		slog.Error("TUI received streamError", "error", msg.err)
		m.isStreaming = false
		m.err = msg.err
		m.addToChatHistory("")
		m.addToChatHistory(errorStyle.Render(fmt.Sprintf("✗ Error: %v", msg.err)))
		m.currentGen.Reset()
		m.updateViewport()
		return m, nil

	case toolCallMsg:
		m.pendingToolMsg = &msg
		return m, m.executeTool(msg)

	case toolResultMsg:
		m.pendingToolMsg = nil
		
		if msg.Error != nil {
			// Tool failed - show error and continue with text response
			m.addToChatHistory(errorStyle.Render(fmt.Sprintf("❌ Error: %v", msg.Error)))
			m.messages = append(m.messages, llm.Message{
				Role:    llm.RoleUser,
				Content: fmt.Sprintf("[Tool Error]\n%v", msg.Error),
			})
		} else {
			// Tool succeeded - don't show raw result, let model explain it
			m.addToChatHistory(dimStyle.Render("✓ Done"))
			m.messages = append(m.messages, llm.Message{
				Role:    llm.RoleUser,
				Content: fmt.Sprintf("[Tool Result]\n%s", msg.Result),
			})
		}

		m.updateViewport()
		
		// Continue with text response (tools disabled to avoid loops)
		m.isStreaming = true
		return m, tea.Batch(m.generateResponseWithoutTools(), m.spinner.Tick)
	}

	return m, tea.Batch(tiCmd, vpCmd, spCmd)
}

func (m *chatModel) View() string {
	header := titleStyle.Render(fmt.Sprintf(" Neuron CLI (%s) ", m.modelID))

	var status string
	if m.isStreaming {
		status = fmt.Sprintf("%s Thinking...", m.spinner.View())
	} else if m.pendingToolMsg != nil {
		status = fmt.Sprintf("%s Executing %s...", m.spinner.View(), m.pendingToolMsg.ToolName)
	} else {
		status = dimStyle.Render("● Ready")
	}

	footer := lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#6272A4")).
		Padding(0, 1).
		Render(m.textarea.View())

	help := dimStyle.Render("Enter: send • Ctrl+C: quit")

	return fmt.Sprintf("%s\n%s\n\n%s\n%s\n%s",
		header,
		status,
		m.viewport.View(),
		footer,
		help,
	)
}

func (m *chatModel) handleUserMessage(text string) {
	slog.Info("User message received", "text", text)
	
	m.messages = append(m.messages, llm.Message{
		Role:    llm.RoleUser,
		Content: text,
	})

	m.addToChatHistory("")
	m.addToChatHistory(userStyle.Render("You:"))
	m.addToChatHistory(text)

	m.isStreaming = true
	m.currentGen.Reset()
	m.err = nil
	m.updateViewport()
}

func (m *chatModel) generateResponseWithoutTools() tea.Cmd {
	slog.Info("Generating LLM response without tools", "message_count", len(m.messages))

	req := llm.CompletionRequest{
		Model:    m.modelID,
		Messages: m.messages,
		Stream:   true,
		Tools:    nil, // No tools - force text response
	}

	m.tokenChan, m.errChan = m.provider.GenerateStream(m.ctx, req)
	slog.Debug("Stream channels created")
	return tea.Batch(m.waitForNextToken(), m.spinner.Tick)
}

func (m *chatModel) generateResponse() tea.Cmd {
	slog.Info("Generating LLM response", "message_count", len(m.messages))
	
	toolDefs := m.registry.GetDefinitions()
	var llmTools []llm.Definition
	for _, def := range toolDefs {
		llmTools = append(llmTools, llm.Definition{
			Type: def.Type,
			Function: llm.FunctionSchema{
				Name:        def.Function.Name,
				Description: def.Function.Description,
				Parameters:  def.Function.Parameters,
			},
		})
	}

	slog.Debug("Tools available", "count", len(llmTools))

	req := llm.CompletionRequest{
		Model:    m.modelID,
		Messages: m.messages,
		Stream:   true,
		Tools:    llmTools,
	}

	m.tokenChan, m.errChan = m.provider.GenerateStream(m.ctx, req)
	slog.Debug("Stream channels created")
	return m.waitForNextToken()
}

func (m *chatModel) waitForNextToken() tea.Cmd {
	return func() tea.Msg {
		select {
		case <-m.ctx.Done():
			slog.Error("Context cancelled")
			return streamErrorMsg{err: m.ctx.Err()}
		case token, ok := <-m.tokenChan:
			if !ok {
				// Token channel closed, check for errors
				select {
				case err, ok := <-m.errChan:
					if ok && err != nil {
						slog.Error("Stream error after tokens", "error", err)
						return streamErrorMsg{err}
					}
				default:
				}
				slog.Debug("Token channel closed, stream done")
				return streamDoneMsg{}
			}
			slog.Debug("Token received", "token", token)
			return tokenMsg(token)
		}
	}
}

func (m *chatModel) executeTool(msg toolCallMsg) tea.Cmd {
	return func() tea.Msg {
		tool, ok := m.registry.Get(msg.ToolName)
		if !ok {
			return toolResultMsg{Error: fmt.Errorf("tool not found: %s", msg.ToolName)}
		}

		m.addToChatHistory("")
		m.addToChatHistory(toolStyle.Render(fmt.Sprintf("🔧 %s...", tool.Name())))
		m.updateViewport()

		result, err := tool.Execute(m.ctx, msg.Args)
		return toolResultMsg{Result: result, Error: err}
	}
}

func (m *chatModel) addToChatHistory(line string) {
	m.chatHistory.WriteString(line)
	m.chatHistory.WriteString("\n")
}

func (m *chatModel) updateViewport() {
	m.viewport.SetContent(m.chatHistory.String())
	m.viewport.GotoBottom()
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// Run starts the TUI application
func Run(provider llm.Provider, modelID string, registry *tools.Registry) error {
	p := tea.NewProgram(
		InitialModel(provider, modelID, registry),
		tea.WithAltScreen(),
	)

	_, err := p.Run()
	return err
}
