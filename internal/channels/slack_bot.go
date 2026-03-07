package channels

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	"github.com/gaurav3000R/neuron-cli/internal/llm"
	"github.com/gaurav3000R/neuron-cli/internal/skills"
	"github.com/slack-go/slack"
	"github.com/slack-go/slack/slackevents"
	"github.com/slack-go/slack/socketmode"
)

// RunSocketMode starts the Slack bot using Socket Mode instead of Webhooks
func (s *SlackChannel) RunSocketMode(ctx context.Context, botToken, appToken string) error {
	client := slack.New(botToken, slack.OptionAppLevelToken(appToken))
	socketClient := socketmode.New(client)

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case evt := <-socketClient.Events:
				switch evt.Type {
				case socketmode.EventTypeEventsAPI:
					eventsAPIEvent, ok := evt.Data.(slackevents.EventsAPIEvent)
					if !ok {
						continue
					}
					socketClient.Ack(*evt.Request)

					switch eventsAPIEvent.Type {
					case slackevents.CallbackEvent:
						innerEvent := eventsAPIEvent.InnerEvent
						switch ev := innerEvent.Data.(type) {
						case *slackevents.AppMentionEvent:
							slog.Info("Received App Mention in Slack", "channel", ev.Channel, "thread", ev.ThreadTimeStamp)
							go s.handleSlackMessage(ctx, client, ev.Channel, ev.ThreadTimeStamp, ev.TimeStamp, ev.Text)
						case *slackevents.MessageEvent:
							// Ignore messages from the bot itself
							if ev.BotID == "" && ev.ClientMsgID != "" {
								slog.Info("Received Message in Slack", "channel", ev.Channel, "thread", ev.ThreadTimeStamp)
								go s.handleSlackMessage(ctx, client, ev.Channel, ev.ThreadTimeStamp, ev.TimeStamp, ev.Text)
							}
						}
					}
				}
			}
		}
	}()

	slog.Info("Starting Slack Socket Mode listener...")
	return socketClient.Run()
}

func (s *SlackChannel) handleSlackMessage(ctx context.Context, client *slack.Client, channel, threadTS, ts, text string) {
	if threadTS == "" {
		threadTS = ts
	}

	threadID := fmt.Sprintf("slack:%s:%s", channel, threadTS)

	// Clean up user mention if it exists
	text = strings.TrimSpace(text)

	s.sessions.AppendHistory(threadID, llm.Message{Role: llm.RoleUser, Content: text})

	// Show typing indicator or initial response
	_, _, err := client.PostMessage(channel, slack.MsgOptionText("_Thinking..._", false), slack.MsgOptionTS(threadTS))
	if err != nil {
		slog.Error("Failed to send typing indicator", "error", err)
	}

	botCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	var process func([]llm.Message, int)

	process = func(history []llm.Message, depth int) {
		if depth > 3 {
			client.PostMessage(channel, slack.MsgOptionText("⚠️ Maximum tool execution limit reached. Please refine your question.", false), slack.MsgOptionTS(threadTS))
			return
		}

		toolDefs := s.registry.GetDefinitions()
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

		if depth > 0 {
			llmTools = nil
		}

		compReq := llm.CompletionRequest{
			Model:    s.modelID,
			Messages: history,
			Stream:   true,
			Tools:    llmTools,
		}

		tokenChan, errChan := s.provider.GenerateStream(botCtx, compReq)
		var fullResponse string
		var errorMessage string
		var toolCallRequested bool

		for {
			select {
			case <-botCtx.Done():
				return
			case token, ok := <-tokenChan:
				if !ok {
					// Stream finished
					if errorMessage != "" {
						client.PostMessage(channel, slack.MsgOptionText(fmt.Sprintf("❌ Error: %s", errorMessage), false), slack.MsgOptionTS(threadTS))
						return
					}

					if !toolCallRequested && fullResponse != "" {
						// Send final text
						s.sessions.AppendHistory(threadID, llm.Message{Role: llm.RoleAssistant, Content: fullResponse})
						client.PostMessage(channel, slack.MsgOptionText(fullResponse, false), slack.MsgOptionTS(threadTS))
					}
					return
				}

				if strings.HasPrefix(token, "__TOOL_CALL__") {
					toolCallRequested = true
					toolCallJSON := strings.TrimPrefix(token, "__TOOL_CALL__")

					var toolCalls []llm.ToolCall
					if err := json.Unmarshal([]byte(toolCallJSON), &toolCalls); err != nil {
						client.PostMessage(channel, slack.MsgOptionText(fmt.Sprintf("❌ Error parsing tool calls: %v", err), false), slack.MsgOptionTS(threadTS))
						return
					}

					if len(toolCalls) > 0 {
						tc := toolCalls[0]
						tool, ok := s.registry.Get(tc.Function.Name)
						if ok {
							client.PostMessage(channel, slack.MsgOptionText(fmt.Sprintf("🔧 Running tool `%s`...", tc.Function.Name), false), slack.MsgOptionTS(threadTS))

							result, err := tool.Execute(botCtx, tc.Function.Arguments)

							history = append(history, llm.Message{
								Role:      llm.RoleAssistant,
								Content:   "",
								ToolCalls: toolCalls,
							})

							if err != nil {
								history = append(history, llm.Message{
									Role:    llm.RoleUser,
									Content: fmt.Sprintf("[Tool Error]\n%v", err),
								})
							} else {
								history = append(history, llm.Message{
									Role:    llm.RoleUser,
									Content: fmt.Sprintf("[Tool Result]\n%s", result),
								})
							}

							process(history, depth+1)
							return
						} else {
							client.PostMessage(channel, slack.MsgOptionText(fmt.Sprintf("❌ Tool '%s' not found", tc.Function.Name), false), slack.MsgOptionTS(threadTS))
							return
						}
					}
				} else {
					fullResponse += token
				}

			case err := <-errChan:
				if err != nil {
					errorMessage = err.Error()
				}
			}
		}
	}

	runHistory := s.sessions.GetHistory(threadID)
	sysCtx := skills.LoadContext()
	if sysCtx != "" {
		// prepend system prompt dynamically without saving to persistent sessions
		runHistory = append([]llm.Message{{Role: llm.RoleSystem, Content: sysCtx}}, runHistory...)
	}

	process(runHistory, 0)
}
