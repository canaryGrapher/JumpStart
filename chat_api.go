package main

import (
	"encoding/json"
	"fmt"
	"strings"

	"devdeck/internal/ai"
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
func (a *App) SendChatMessage(host, model, projectID, sessionID, text string) (*chatstore.Session, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil, fmt.Errorf("message is empty")
	}

	session, err := a.resolveSession(projectID, sessionID)
	if err != nil {
		return nil, err
	}

	history := toAIHistory(session.Messages)
	history = append(history, ai.ChatMessage{Role: "user", Content: text})

	result, chatErr := a.OllamaChat(host, model, history, projectID)

	turns := []chatstore.Message{{Role: "user", Content: text}}
	if chatErr != nil {
		// Persist the user turn so the thread is not lost, and surface the
		// failure as an assistant message rather than an empty screen.
		turns = append(turns, chatstore.Message{
			Role:    "assistant",
			Content: "Couldn't reach the model: " + chatErr.Error(),
		})
		if _, err := chatstore.Append(projectID, session.ID, turns); err != nil {
			return nil, err
		}
		return nil, chatErr
	}

	reply := chatstore.Message{
		Role:    "assistant",
		Content: result.Reply,
		Sources: result.Sources,
	}
	if len(result.Stories) > 0 {
		if raw, err := json.Marshal(result.Stories); err == nil {
			reply.Stories = raw
		}
	}
	turns = append(turns, reply)

	return chatstore.Append(projectID, session.ID, turns)
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
