// Package logging envuelve el canonical logger del lab (go-shared, ADR-001). P-20.
package logging

import gs "github.com/hornosg/go-shared/infrastructure/logging"

type Logger struct{ c *gs.CanonicalLogger }

func New(service string) *Logger { return &Logger{c: gs.NewCanonicalLogger(service)} }

func (l *Logger) Info(event string, fields map[string]any)  { l.c.Emit("info", event, fields) }
func (l *Logger) Warn(event string, fields map[string]any)  { l.c.Emit("warn", event, fields) }
func (l *Logger) Error(event string, fields map[string]any) { l.c.Emit("error", event, fields) }
