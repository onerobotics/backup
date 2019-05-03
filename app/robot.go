package app

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

type Robot struct {
	Name string
	Host string
}

func NewRobot() (*Robot, error) {
	r := bufio.NewReader(os.Stdin)
begin:
	fmt.Println("Please provide a name for the robot (e.g. R1):")
	name, err := r.ReadString('\n')
	if err != nil {
		return nil, err
	}
	name = strings.TrimSpace(name)

	fmt.Printf("What is %s's IP address?\n", name)
	host, err := r.ReadString('\n')
	if err != nil {
		return nil, err
	}
	host = strings.TrimSpace(host)

confirm:
	fmt.Printf("Name: %s\nIP:   %s\n", name, host)
	fmt.Println("Is this correct? (Y/N)")
	answer, err := r.ReadString('\n')
	if err != nil {
		return nil, err
	}

	answer = strings.ToLower(strings.TrimSpace(answer))
	switch answer {
	case "y":
		return &Robot{name, host}, nil
	case "n":
		goto begin
	default:
		goto confirm
	}
}
