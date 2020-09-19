package main

import "fmt"

var (
	version = "0.1.0"
	date    = "2020/09/19"
)

func init() {
	fmt.Printf("version: %s\ndate: %s\n\n", version, date)
}
