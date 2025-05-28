package logger

import (
	"fmt"
	"log"
	"time"
)

// Error wrapper for fmt.Errorf
func Error(f, op string, err error) error {
	return fmt.Errorf("%v -> %v: %v", f, op, err)
}

// Add wrapper for log.Printf
func Add(f, op string, err error) {
	log.Printf("[%s] %v -> %v: %v", time.Now().Format(time.DateTime), f, op, err)
}
