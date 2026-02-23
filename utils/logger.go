package utils

import (
  "fmt"
  "os"
  "time"
)

var logFile = "tuik.log"

// Log writes a message to tuik.log in the current directory.
func Log(format string, v ...any) {
  f, err := os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
  if err != nil {
    return
  }
  defer f.Close()

  timestamp := time.Now().Format("15:04:05")
  msg := fmt.Sprintf(format, v...)
  fmt.Fprintf(f, "[%s] %s\n", timestamp, msg)
}
