package gocrawler

import (
	"fmt"
	"os"
)

func main() {

	args := os.Args

	if (len(args) != 2) {
		fmt.Println("Usage: gocrawler [domainName]\n")
		return
	}

	domainName := args[1]

	fmt.Println("Hello, world " + domainName)
}
