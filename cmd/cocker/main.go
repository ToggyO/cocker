package main

import (
	"os"

	"github.com/ToggyO/cocker/internal/cli"
	linuxprocess "github.com/ToggyO/cocker/internal/platform/linux/process"
)

func main() {
	runner := linuxprocess.NewRunner() // временное решение для linux имплементации
	os.Exit(cli.Run(os.Args[1:], runner))
}
