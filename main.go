package main

import (
	"flag"
	"fmt"
)

var (
	sha1ver   string
	buildTime string
)

func main() {
	version := flag.Bool("version", false, "Print the program version")
	flag.Parse()

	if *version {
		fmt.Printf("Binary sha %s built on %s\n", sha1ver, buildTime)
		return
	}
}
