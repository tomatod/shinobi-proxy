package main

import (
	"fmt"
)

const (
	LOG_LEVEL = 0 // 0 is debug
)

func printf(level int, str string, params ...interface{}) {
	if level >= LOG_LEVEL {
		fmt.Printf(str, params...)
	}
}
