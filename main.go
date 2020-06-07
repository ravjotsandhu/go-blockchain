package main

import (
	"os"

	"github.com/RavjotSandhu/GoBlockchain/cli"
)

func main() {
	defer os.Exit(0)
	cmd := cli.CommandLine{}
	cmd.Run()
}
