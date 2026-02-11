package main

import (
	"flag"
	"fmt"
)

func main() {
	fmt.Println("bingobernt")

	var id string
	flag.StringVar(&id, "id", "", "id of this peer")
	flag.Parse()
}
