package main

import (
	"fmt"

	"github.com/brad-jones/gopwsh"
)

func main() {
	shell := gopwsh.MustNew()
	defer shell.Exit()
	stdout, _, err := shell.Execute("Get-ComputerInfo")
	if err != nil {
		panic(err)
	}
	fmt.Println(stdout)
}
