package main

import "scdl/cmd"

var Version = "1.0.1"

func main() {
	cmd.SetVersion(Version)
	cmd.Execute()
}
