package logger

import (
	"fmt"
	"log"
	"time"
)

// Error обертка для fmt.Errorf
func Error(f, op string, err error) error {
	return fmt.Errorf("%v -> %v error: %v", f, op, err)
}

// Add обертка для log.Printf
func Add(f, op string, err error) {
	log.Printf("[%s] %v -> %v error: %v", time.Now().Format(time.DateTime), f, op, err)
}
