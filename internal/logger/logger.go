package logger

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

// Logger estructura principal del logger
type Logger struct{}

// NewLogger crea una instancia de Logger
func NewLogger() *Logger {
	return &Logger{}
}

// logMessage estructura de un mensaje de log
type logMessage struct {
	Time    string `json:"time"`
	Level   string `json:"level"`
	Module  string `json:"module"`
	Func    string `json:"func"`
	Message string `json:"message"`
}

// Info genera un log de nivel INFO
func (l *Logger) Info(module, function, message string) {
	l.print("INFO", module, function, message)
}

// Error genera un log de nivel ERROR
func (l *Logger) Error(module, function, message string) {
	l.print("ERROR", module, function, message)
}

// Warning genera un log de nivel WARNING
func (l *Logger) Warning(module, function, message string) {
	l.print("WARNING", module, function, message)
}

// Debug genera un log de nivel DEBUG
func (l *Logger) Debug(module, function, message string) {
	l.print("DEBUG", module, function, message)
}

// print serializa el log en JSON y lo escribe a stdout
func (l *Logger) print(level, module, function, message string) {
	logEntry := logMessage{
		Time:    time.Now().Format(time.RFC3339),
		Level:   level,
		Module:  module,
		Func:    function,
		Message: message,
	}
	data, err := json.Marshal(logEntry)
	if err != nil {
		// fallback simple
		fmt.Fprintf(os.Stdout, "[%s] %s.%s: %s\n", level, module, function, message)
		return
	}
	fmt.Println(string(data))
}
