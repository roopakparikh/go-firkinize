// Copyright 2020 Platform9 Systems Inc.

package main

import (
  "github.com/platform9/go-firkinize/cmd"
  "os"
)

func main() {
  err := cmd.Execute()
  if err != nil {
    os.Exit(1)
  }
}
