package logging

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"
	"sync"
)

type Handler struct {
	out   io.Writer
	mu    *sync.Mutex
	attrs []slog.Attr
}

func New(out io.Writer) *Handler {
	return &Handler{out: out, mu: &sync.Mutex{}}
}

func Setup() {
	slog.SetDefault(slog.New(New(os.Stdout)))
}

func (h *Handler) Enabled(context.Context, slog.Level) bool {
	return true
}

// Handle logging style: [15:04:05] [relay/INFO]: message key=value
func (h *Handler) Handle(_ context.Context, r slog.Record) error {
	var sb strings.Builder
	_, err := fmt.Fprintf(&sb, "[%s] [relay/%s]: %s", r.Time.Format("15:04:05"), r.Level, r.Message)
	if err != nil {
		return err
	}

	for _, a := range h.attrs {
		writeAttr(&sb, a)
	}

	r.Attrs(func(a slog.Attr) bool {
		writeAttr(&sb, a)
		return true
	})
	sb.WriteByte('\n')

	h.mu.Lock()
	defer h.mu.Unlock()

	_, err = io.WriteString(h.out, sb.String())
	return err
}

func (h *Handler) WithAttrs(attrs []slog.Attr) slog.Handler {
	nh := *h
	nh.attrs = append(append([]slog.Attr{}, h.attrs...), attrs...)
	return &nh
}

func (h *Handler) WithGroup(string) slog.Handler {
	return h
}

func writeAttr(sb *strings.Builder, a slog.Attr) {
	v := a.Value.Resolve()
	sb.WriteByte(' ')
	sb.WriteString(a.Key)
	sb.WriteByte('=')
	if s, ok := v.Any().(string); ok && !strings.ContainsAny(s, " \t\"") {
		sb.WriteString(s)
		return
	}
	fmt.Fprintf(sb, "%v", v)
}
