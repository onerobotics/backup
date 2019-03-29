package project

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/onerobotics/backup/robot"
)

const VERSION = "1.0.0"
const JSON_FILENAME = "backup_tool.json"

type Project struct {
	Destination string
	Version     string
	Robots      []robot.Robot
}

func (p *Project) fromJSON() error {
	data, err := ioutil.ReadFile(JSON_FILENAME)
	if err != nil {
		return err
	}

	json.Unmarshal([]byte(data), p)

	return nil
}

func (p *Project) fromWizard() error {
	r := bufio.NewReader(os.Stdin)

	questions:
	fmt.Println("Where should backups be stored?")
	dest, err := r.ReadString('\n')
	if err != nil {
		return err
	}
	dest = strings.TrimSpace(dest)

	confirm:
	fmt.Printf("Destination: %s\n", dest)
	fmt.Println("Is this correct? (Y/N)")

	answer, err := r.ReadString('\n')
	if err != nil {
		return err
	}

	answer = strings.ToLower(strings.TrimSpace(answer))
	switch answer {
	case "y":
		// noop
	case "n":
		goto questions
	default:
		goto confirm
	}

	p.Destination = dest
	p.Version = VERSION

	return p.Save()
}


func Init() (*Project, error) {
	p := &Project{}

	err := p.fromJSON()
	if os.IsNotExist(err) {
		err = p.fromWizard()
	}
	if err != nil {
		return nil, err
	}

	return p, nil
}

func (p *Project) Save() error {
	b, err := json.Marshal(p)
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(JSON_FILENAME, b, 0644)
	if err != nil {
		return err
	}

	fmt.Println("Project saved.")

	return nil
}

func (p *Project) AddRobot() error {
	r, err := robot.FromWizard()
	if err != nil {
		return err
	}

	p.Robots = append(p.Robots, *r)
	return p.Save()
}

func (p *Project) RemoveRobot() error {
	if len(p.Robots) < 1 {
		fmt.Println("Your project does not have any robots. Please run `BackupTool add` to add one.")
		return nil
	}


	list:
	for id, robot := range p.Robots {
		fmt.Printf("%d. %s %s\n", id+1, robot.Name, robot.Host)
	}

	fmt.Println("\nWhich robot do you want to remove?")

	var id int
	for _, err := fmt.Scanf("%d", &id); err != nil; {
		fmt.Println("Invalid id. Try again.")
	}

	id = id - 1
	if id < 0 || id > len(p.Robots)-1 {
		fmt.Println("Id out of range")
		goto list
	}

	fmt.Printf("Removing robot #%d\n", id+1)

	p.Robots = append(p.Robots[:id], p.Robots[id+1:]...)

	return p.Save()
}

func (p *Project) Backup(filter func(string) bool, name string) {
	if len(p.Robots) <= 0 {
		fmt.Println("Your project does not have any robots. Please run `BackupTool add` to add one.")
		return
	}

	t := time.Now()

	log.Println("Backing up project...")

	dest := filepath.Join(p.Destination, fmt.Sprintf("%d-%02d-%02dT%02d-%02d-%02d_%s", t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second(), name))

	var wg sync.WaitGroup
	for _, r := range p.Robots {
		wg.Add(1)
		go r.Backup(filter, dest, &wg)
	}
	wg.Wait()

	log.Printf("Backed up all robots in %v", time.Since(t))
}
