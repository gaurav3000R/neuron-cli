package tui

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/gaurav3000R/neuron-cli/internal/llm"
	"github.com/gaurav3000R/neuron-cli/internal/skills"
	"github.com/gaurav3000R/neuron-cli/internal/tools"
)

var (
	brandStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#BD93F9")).
			Bold(true).
			SetString("✧ NEURON")

	managedByStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#6272A4")).
			Bold(true).
			Italic(true).
			SetString("managed by GAURAV")

	headerStyle = lipgloss.NewStyle().
			Border(lipgloss.NormalBorder(), false, false, true, false).
			BorderForeground(lipgloss.Color("#44475A")).
			MarginBottom(1)

	userStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#8BE9FD")).
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

	infoStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#BD93F9"))
)

// Messages for bubbletea
type tokenMsg string
type tickMsg time.Time
type streamDoneMsg struct{}
type streamErrorMsg struct{ err error }
type toolCallMsg struct {
	ID       string
	ToolName string
	Args     json.RawMessage
}
type toolResultMsg struct {
	ToolCallID string
	Result     string
	Error      error
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

	width  int
	height int
	frame  int
}

func tick() tea.Cmd {
	return tea.Tick(time.Millisecond*100, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

// InitialModel configures the starting state of the TUI.
func InitialModel(provider llm.Provider, modelID string, registry *tools.Registry) *chatModel {
	ta := textarea.New()
	ta.Placeholder = "Type your message and press Enter..."
	ta.Focus()
	ta.Prompt = "┃ "
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

	sysCtx := skills.LoadContext()
	if sysCtx != "" {
		m.messages = append(m.messages, llm.Message{Role: llm.RoleSystem, Content: sysCtx})
	}

	m.updateViewport()

	return m
}

func (m *chatModel) Init() tea.Cmd {
	return tea.Batch(textarea.Blink, m.spinner.Tick, tick())
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

	case tickMsg:
		m.frame++
		return m, tick()

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
		m.width = msg.Width
		m.height = msg.Height
		m.viewport.Width = msg.Width
		headerHeight := 4
		footerHeight := 8
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
				slog.Info("Executing tool", "name", tc.Function.Name, "args", string(tc.Function.Arguments), "id", tc.ID)

				// Add tool call to message history
				m.messages = append(m.messages, llm.Message{
					Role:      llm.RoleAssistant,
					Content:   "",
					ToolCalls: toolCalls,
				})

				return m, m.executeTool(toolCallMsg{
					ID:       tc.ID,
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
				Role:       llm.RoleTool,
				Content:    fmt.Sprintf("Error: %v", msg.Error),
				ToolCallID: msg.ToolCallID,
			})
		} else {
			// Tool succeeded - don't show raw result, let model explain it
			m.addToChatHistory(dimStyle.Render("✓ Done"))
			m.messages = append(m.messages, llm.Message{
				Role:       llm.RoleTool,
				Content:    msg.Result,
				ToolCallID: msg.ToolCallID,
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
	// Animate the brand name in the header
	brandColors := []string{"#BD93F9", "#8BE9FD", "#FF79C6", "#50FA7B"}
	brandColor := brandColors[(m.frame/5)%len(brandColors)]

	brand := lipgloss.NewStyle().
		Foreground(lipgloss.Color(brandColor)).
		Bold(true).
		Render(fmt.Sprintf("✧ NEURON"))

	managedBy := managedByStyle.Render()

	gapWidth := m.width - lipgloss.Width(brand) - lipgloss.Width(managedBy) - 2
	if gapWidth < 0 {
		gapWidth = 0
	}
	gap := strings.Repeat(" ", gapWidth)

	header := headerStyle.Width(m.width).Render(
		lipgloss.JoinHorizontal(lipgloss.Center, brand, gap, managedBy),
	)

	var status string
	if m.isStreaming {
		status = fmt.Sprintf(" %s Thinking...", m.spinner.View())
	} else if m.pendingToolMsg != nil {
		status = fmt.Sprintf(" %s Executing %s...", m.spinner.View(), m.pendingToolMsg.ToolName)
	} else {
		status = dimStyle.Render(" ● Ready")
	}

	// If no messages yet, show the animated banner
	var mainView string
	if len(m.messages) == 0 {
		banner := getAnimatedBanner(m.frame)
		welcomeInfo := lipgloss.NewStyle().
			MarginTop(1).
			Padding(1, 2).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#44475A")).
			Render(
				lipgloss.JoinVertical(lipgloss.Left,
					infoStyle.Render(fmt.Sprintf("• Model: %s", m.modelID)),
					dimStyle.Render(fmt.Sprintf("• Tools: %d enabled", len(m.registry.GetAll()))),
					dimStyle.Render("• System Status: Online"),
				),
			)

		mainView = lipgloss.Place(m.width, m.viewport.Height,
			lipgloss.Center, lipgloss.Center,
			lipgloss.JoinVertical(lipgloss.Center, banner, welcomeInfo),
		)
	} else {
		mainView = m.viewport.View()
	}

	footer := lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#44475A")).
		Padding(0, 1).
		Render(m.textarea.View())

	help := dimStyle.Render(" Enter: send • Ctrl+C: quit")

	return lipgloss.JoinVertical(lipgloss.Left,
		header,
		status,
		"",
		mainView,
		"",
		footer,
		help,
	)
}

func getAnimatedBanner(frame int) string {
	lines := []string{
		"███╗   ██╗███████╗██╗   ██╗██████╗  ██████╗ ██╗   ██╗",
		"████╗  ██║██╔════╝██║   ██║██╔══██╗██╔═══██╗████╗  ██║",
		"██╔██╗ ██║█████╗  ██║   ██║██████╔╝██║   ██║██╔██╗ ██║",
		"██║╚██╗██║██╔══╝  ██║   ██║██╔══██╗██║   ██║██║╚██╗██║",
		"██║ ╚████║███████╗╚██████╔╝██║  ██║╚██████╔╝██║ ╚████║",
		"╚═╝  ╚═══╝╚══════╝ ╚═════╝ ╚═╝  ╚═╝ ╚═════╝ ╚═╝  ╚═══╝",
	}

	// Neural connection decorations
	topDeco := "   ○───✧───●──────────✧──────────●───✧───○"
	botDeco := "   ○───✧───●──────────✧──────────●───✧───○"

	gradient := []lipgloss.Color{
		lipgloss.Color("#8BE9FD"), // Cyan
		lipgloss.Color("#BD93F9"), // Purple
		lipgloss.Color("#FF79C6"), // Pink
		lipgloss.Color("#BD93F9"), // Purple
		lipgloss.Color("#8BE9FD"), // Cyan
	}

	var styledBanner strings.Builder

	styledBanner.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#6272A4")).Render(topDeco) + "\n\n")

	for _, line := range lines {
		var styledLine strings.Builder
		runes := []rune(line)
		for j, r := range runes {
			// Shift the gradient using the frame counter for a "working" effect
			colorIdx := ((j + frame) * len(gradient) / 10) % len(gradient)
			style := lipgloss.NewStyle().Foreground(gradient[colorIdx]).Bold(true)
			styledLine.WriteString(style.Render(string(r)))
		}
		styledBanner.WriteString(styledLine.String() + "\n")
	}

	styledBanner.WriteString("\n" + lipgloss.NewStyle().Foreground(lipgloss.Color("#6272A4")).Render(botDeco) + "\n")

	return styledBanner.String()
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
			return toolResultMsg{ToolCallID: msg.ID, Error: fmt.Errorf("tool not found: %s", msg.ToolName)}
		}

		m.addToChatHistory("")
		m.addToChatHistory(toolStyle.Render(fmt.Sprintf("🔧 %s...", tool.Name())))
		m.updateViewport()

		result, err := tool.Execute(m.ctx, msg.Args)
		return toolResultMsg{ToolCallID: msg.ID, Result: result, Error: err}
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
