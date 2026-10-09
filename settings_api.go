package main

import (
	"context"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"devdeck/internal/ai"
	"devdeck/internal/analytics"
	"devdeck/internal/appsettings"
)

// --- App settings shared by the UI, MCP and Raycast ---

// GetAppSettings returns the persisted app-wide preferences.
func (a *App) GetAppSettings() appsettings.Settings {
	return appsettings.Load(analytics.DataDir())
}

// SetAppSettings validates and saves app-wide preferences and returns the
// stored copy (blanks filled with defaults).
func (a *App) SetAppSettings(s appsettings.Settings) (appsettings.Settings, error) {
	before := appsettings.Load(analytics.DataDir())
	saved, err := appsettings.Save(analytics.DataDir(), s)
	if err == nil {
		a.applyHotkeySettings(saved)
		// New OCR settings deserve another try at images that failed before.
		if ocrConfig(before) != ocrConfig(saved) {
			a.textWorker().Reset(true)
		}
	}
	return saved, err
}

// --- AI request options, progress and cancellation ---

// AIRequestOptions accompanies one AI call from the UI.
type AIRequestOptions struct {
	// RequestID lets the UI match progress events and cancel the call.
	RequestID string `json:"requestId"`
	// Think overrides the default effort for this call: auto|off|low|medium|high.
	// Empty uses Settings > AI.
	Think string `json:"think"`
}

// AIDelta is emitted as "ai:delta" while a reply streams.
type AIDelta struct {
	RequestID string `json:"requestId"`
	Thinking  string `json:"thinking,omitempty"`
	Content   string `json:"content,omitempty"`
}

// OllamaModelInfo reports what a model supports (thinking, effort levels,
// vision) so Settings can explain the effort slider.
func (a *App) OllamaModelInfo(host, model string) (ai.ModelInfo, error) {
	return a.modelInfoFor(host, model)
}

func (a *App) modelInfoFor(host, model string) (ai.ModelInfo, error) {
	key := host + "|" + model
	if v, ok := a.modelInfo.Load(key); ok {
		return v.(ai.ModelInfo), nil
	}
	ctx, cancel := context.WithTimeout(a.ctx, 5*time.Second)
	defer cancel()
	info, err := ai.New(host).ShowModel(ctx, model)
	if err == nil {
		a.modelInfo.Store(key, info)
	}
	return info, err
}

// CancelAIRequest stops an in-flight AI call started with this request id.
// Unknown or finished ids are ignored.
func (a *App) CancelAIRequest(requestID string) {
	if v, ok := a.aiRequests.LoadAndDelete(requestID); ok {
		v.(context.CancelFunc)()
	}
}

// aiCall prepares the context and chat options for one AI request: the
// effective thinking effort, live "ai:delta" events, and a cancel hook.
// Call done() when the request finishes.
func (a *App) aiCall(host, model string, o AIRequestOptions) (ctx context.Context, opts ai.ChatOptions, done func()) {
	ctx, cancel := context.WithCancel(a.ctx)
	if o.RequestID != "" {
		a.aiRequests.Store(o.RequestID, cancel)
	}
	think := o.Think
	if think == "" {
		think = appsettings.Load(analytics.DataDir()).ThinkLevel
	}
	opts.Think = think
	if think != ai.ThinkAuto && think != ai.ThinkOff {
		if info, err := a.modelInfoFor(host, model); err == nil {
			opts.ThinkLevels = info.ThinkLevels
			if !info.Thinking {
				// Non-thinking models reject `think`; let them run normally.
				opts.Think = ai.ThinkAuto
			}
		}
	}
	if o.RequestID != "" && a.ctx != nil {
		opts.OnDelta = func(d ai.Delta) {
			runtime.EventsEmit(a.ctx, "ai:delta", AIDelta{RequestID: o.RequestID, Thinking: d.Thinking, Content: d.Content})
		}
	}
	done = func() {
		if o.RequestID != "" {
			a.aiRequests.Delete(o.RequestID)
		}
		cancel()
	}
	return ctx, opts, done
}
