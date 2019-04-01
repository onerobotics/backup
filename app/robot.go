package app

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/onerobotics/backup/ftp"
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

func (r Robot) Backup(filter func(filename string) bool, destination string, wg *sync.WaitGroup) error {
	defer wg.Done()

	t := time.Now()

	log.Println("Backing up", r.Name, "at", r.Host)
	dirname := destination + "/" + r.Name
	err := os.MkdirAll(dirname, os.ModePerm)
	if err != nil {
		return err
	}

	c := ftp.NewConnection(r.Host, "21")
	c.Connect()
	defer c.Quit()

	files, err := c.NameList()
	if err != nil {
		log.Println("error getting list of files", err)
		return err
	}

	err = c.Type("I")
	if err != nil {
		return err
	}

	var errorList []error
	for _, file := range files {
		if filter(file) {
			log.Printf("%s: Downloading %s", r.Name, file)
			err := c.Download(file, dirname)
			if err != nil {
				errorList = append(errorList, err)
			}
		} else {
			//log.Printf("%s: Skipping %s", r.Name, file)
		}
	}

	if len(errorList) > 0 {
		log.Printf("There were %d errors.\n", len(errorList))
		for _, err := range errorList {
			log.Println(err)
		}
	}

	log.Printf("Finished backing up %s in %v\n", r.Name, time.Since(t))

	return nil
}

