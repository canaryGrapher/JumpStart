package codectx

import "strings"

const (
	chunkLines   = 70 // lines per chunk
	chunkOverlap = 12 // lines repeated at the chunk boundary
	maxChunkRune = 6000
)

// chunkFile splits a file's text into overlapping line windows. Overlap
// keeps a function that straddles a boundary readable in at least one
// chunk.
func chunkFile(rel, lang, text string) []Chunk {
	lines := strings.Split(text, "\n")
	if len(lines) == 0 {
		return nil
	}

	step := chunkLines - chunkOverlap
	if step < 1 {
		step = chunkLines
	}

	var out []Chunk
	for start := 0; start < len(lines); start += step {
		end := start + chunkLines
		if end > len(lines) {
			end = len(lines)
		}
		body := strings.Join(lines[start:end], "\n")
		if strings.TrimSpace(body) == "" {
			if end == len(lines) {
				break
			}
			continue
		}
		if len(body) > maxChunkRune {
			body = body[:maxChunkRune]
		}
		out = append(out, Chunk{
			Path:  rel,
			Lang:  lang,
			Start: start + 1,
			End:   end,
			Text:  body,
		})
		if end == len(lines) {
			break
		}
	}
	return out
}

// Render formats a chunk for inclusion in a prompt.
func (c Chunk) Render() string {
	var b strings.Builder
	b.WriteString("--- ")
	b.WriteString(c.Path)
	b.WriteString(":")
	b.WriteString(itoa(c.Start))
	b.WriteString("-")
	b.WriteString(itoa(c.End))
	b.WriteString(" ---\n")
	b.WriteString(c.Text)
	b.WriteString("\n")
	return b.String()
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
}
