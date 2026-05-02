package main

import "fmt"

func main() {
	s := NewService("test")
	result := s.Run()
	fmt.Println(result)
	helper()
}

func helper() string {
	return "ok"
}
