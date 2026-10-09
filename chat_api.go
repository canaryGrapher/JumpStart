package main

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"devdeck/internal/ai"
	"devdeck/internal/analytics"
	"devdeck/internal/chatstore"
)

// --- Chat sessions ---
//
// A session is one conversation thread with the story assistant. Users can
// continue an existing thread (the model keeps the earlier turns) or start
// a fresh one to reset the framing without losing the old thread.

// ListChats returns a project's conversations, newest first.
func (a *App) ListChats(projectID string) ([]chatstore.Summary, error) {
	return chatstore.List(projectID)
}

// GetChat returns one full conversation including its messages.
func (a *App) GetChat(projectID, sessionID string) (*chatstore.Session, error) {
	return chatstore.Get(projectID, sessionID)
}

// NewChat starts an empty conversation.
func (a *App) NewChat(projectID, title string) (*chatstore.Session, error) {
	return chatstore.Create(projectID, title)
}

// RenameChat sets a conversation's title.
func (a *App) RenameChat(projectID, sessionID, title string) error {
	return chatstore.Rename(projectID, sessionID, title)
}

// DeleteChat removes one conversation.
func (a *App) DeleteChat(projectID, sessionID string) error {
	return chatstore.Delete(projectID, sessionID)
}

// SendChatMessage is the single entry point the UI calls to talk to the
// assistant. It loads the session, replays its history to the model with
// code context attached, then persists both the user turn and the reply.
//
// Passing an empty sessionID creates a new session first, so the caller
// never has to make two round trips to start a chat.
// SendChatMessage is instrumented rather than OllamaChat, because the UI
// reaches the model through this one entry point and instrumenting both
// would double-count every message.
//
// Zero chat content is reported: not the message, not the reply, not the
// files the retriever surfaced. Only length, latency, and whether code
// context was available.
func (a *App) SendChatMessage(host, model, projectID, sessionID, text string, opts AIRequestOptions) (session *chatstore.Session, err error) {
	start := time.Now()
	messageCount := 0
	storyCount := 0
	sourceCount := 0
	defer func() {
		a.track("ai_chat_message_sent", outcome(start, err, map[string]any{
			"model_family":          analytics.AIModelFamily(model),
			"param_size":            analytics.AIParamSize(model),
			"session_message_count": messageCount,
			"used_code_context":     sourceCount > 0,
			"context_chunk_count":   sourceCount,
			"stories_generated":     storyCount,
			"message_length":        len(text),
			"is_new_session":        strings.TrimSpace(sessionID) == "",
			"project_ref":           a.ref(projectID),
		}))
	}()

	text = strings.TrimSpace(text)
	if text == "" {
		err = fmt.Errorf("message is empty")
		return nil, err
	}

	session, err = a.resolveSession(projectID, sessionID)
	if err != nil {
		return nil, err
	}
	messageCount = len(session.Messages)

	history := toAIHistory(session.Messages)
	history = append(history, ai.ChatMessage{Role: "user", Content: text})

	result, chatErr := a.OllamaChat(host, model, history, projectID, opts)

	turns := []chatstore.Message{{Role: "user", Content: text}}
	if chatErr != nil {
		// Persist the user turn so the thread is not lost, and surface the
		// failure as an assistant message rather than an empty screen.
		msg := "Couldn't reach the model: " + chatErr.Error()
		if chatErr == ai.ErrCanceled {
			msg = "Stopped before the model finished."
		}
		turns = append(turns, chatstore.Message{Role: "assistant", Content: msg})
		saved, aerr := chatstore.Append(projectID, session.ID, turns)
		if aerr != nil {
			err = aerr
			return nil, err
		}
		if chatErr == ai.ErrCanceled {
			// Stopping is a user choice, not a failure: show the thread.
			return saved, nil
		}
		err = chatErr
		return nil, err
	}

	storyCount = len(result.Stories)
	sourceCount = len(result.Sources)

	reply := chatstore.Message{
		Role:    "assistant",
		Content: result.Reply,
		Sources: result.Sources,
	}
	if len(result.Stories) > 0 {
		if raw, merr := json.Marshal(result.Stories); merr == nil {
			reply.Stories = raw
		}
	}
	turns = append(turns, reply)

	session, err = chatstore.Append(projectID, session.ID, turns)
	return session, err
}

// resolveSession returns the named session, creating one when sessionID is
// empty or no longer exists.
func (a *App) resolveSession(projectID, sessionID string) (*chatstore.Session, error) {
	if strings.TrimSpace(sessionID) != "" {
		if s, err := chatstore.Get(projectID, sessionID); err == nil {
			return s, nil
		}
	}
	return chatstore.Create(projectID, "")
}

// toAIHistory converts stored messages into the model's wire format.
// Stories are dropped: the model re-derives them and replaying the JSON
// would waste the context window.
func toAIHistory(msgs []chatstore.Message) []ai.ChatMessage {
	out := make([]ai.ChatMessage, 0, len(msgs))
	for _, m := range msgs {
		if m.Role != "user" && m.Role != "assistant" {
			continue
		}
		if strings.TrimSpace(m.Content) == "" {
			continue
		}
		out = append(out, ai.ChatMessage{Role: m.Role, Content: m.Content})
	}
	return trimHistory(out, 20)
}

// trimHistory keeps the most recent n turns so a long thread does not
// overflow a local model's context window.
func trimHistory(msgs []ai.ChatMessage, n int) []ai.ChatMessage {
	if len(msgs) <= n {
		return msgs
	}
	return msgs[len(msgs)-n:]
}
