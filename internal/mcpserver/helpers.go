package mcpserver

import (
	"encoding/json"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func textResult(v any) (*mcp.CallToolResult, any, error) {
	var text string
	switch t := v.(type) {
	case string:
		text = t
	default:
		b, err := json.MarshalIndent(v, "", "  ")
		if err != nil {
			return nil, nil, err
		}
		text = string(b)
	}
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: text}},
	}, nil, nil
}

func toolError(err error) (*mcp.CallToolResult, any, error) {
	if err == nil {
		return textResult("ok")
	}
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: err.Error()}},
		IsError: true,
	}, nil, nil
}

func require(cond bool, msg string) error {
	if cond {
		return nil
	}
	return fmt.Errorf("%s", msg)
}
