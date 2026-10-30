package utils

import (
	"fmt"
	"time"
)

type log struct {
}

var Logger log

func (l log) Info(msg string) {
	fmt.Printf("[DMP] %s [I] %s\n", time.Now().Format(time.DateTime), msg)
}

func (l log) Error(msg string) {
	fmt.Printf("[DMP] %s [E] %s\n", time.Now().Format(time.DateTime), msg)
}
