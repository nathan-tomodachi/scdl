package main

import "scdl/cmd"

var Version = "1.0.0"

func main() {
	cmd.SetVersion(Version)
	cmd.Execute()
}
