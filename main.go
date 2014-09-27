package main
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

	url := args[1]

	//fmt.Println("Hello, world " + url)

	Scan(url)


}
