package main

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

	"github.com/codegangsta/cli"
	"github.com/unreal/backup/ftp"
)

const VERSION = "1.0.0"

type Project struct {
	Destination string
	Version     string
	Robots      []Robot
}

type Robot struct {
	Name string
	Host string
}

func check(e error) {
	if e != nil {
		panic(e)
	}
}

func (r Robot) Backup(filter func(filename string) bool, destination string, wg *sync.WaitGroup) {
	defer wg.Done()
	t := time.Now()

	fmt.Println("Backing up", r.Name, "at", r.Host)
	dirname := destination + "/" + r.Name
	err := os.MkdirAll(dirname, os.ModePerm)
	check(err)

	c := ftp.NewConnection(r.Host, "21")
	c.Connect()
	defer c.Quit()

	files, err := c.NameList()
	if err != nil {
		log.Println("error getting list of files")
		return
	}

	c.Type("I")

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
}

func InitProject() *Project {
	var p Project
	data, err := ioutil.ReadFile("backup_tool.json")
	if os.IsNotExist(err) {
		f, err := os.Create("backup_tool.json")
		check(err)
		fmt.Println("backup_tool.json config file created.")
		f.Close()

		reader := bufio.NewReader(os.Stdin)
	Questions:
		fmt.Println("Where should backups be stored?")
		dest, err := reader.ReadString('\n')
		check(err)
		dest = strings.TrimSpace(dest)

	confirm:
		fmt.Printf("Destination: %s\n", dest)
		fmt.Println("Is this correct? (Y/N)")
		answer, err := reader.ReadString('\n')
		check(err)
		answer = strings.TrimSpace(answer)
		if answer != "Y" && answer != "y" && answer != "N" && answer != "n" {
			goto confirm
		}

		if answer == "N" || answer == "n" {
			goto Questions
		}

		p.Destination = dest
		p.Version = VERSION
		p.Save()
	} else {
		json.Unmarshal([]byte(data), &p)
	}

	return &p
}

func (p *Project) AddRobot() {
	reader := bufio.NewReader(os.Stdin)
begin:
	fmt.Println("Please provide a name for the robot (e.g. R1):")
	name, err := reader.ReadString('\n')
	check(err)
	name = strings.TrimSpace(name)

	fmt.Printf("What is %s's IP address?\n", name)
	ip, err := reader.ReadString('\n')
	check(err)
	ip = strings.TrimSpace(ip)

confirm:
	fmt.Printf("Name: %s\nIP:   %s\n", name, ip)
	fmt.Println("Is this correct? (Y/N)")
	answer, err := reader.ReadString('\n')
	check(err)
	answer = strings.TrimSpace(answer)
	switch answer {
	case "Y", "y":
		r := Robot{Name: name, Host: ip}
		p.Robots = append(p.Robots, r)
		p.Save()
	case "N", "n":
		goto begin
	default:
		goto confirm
	}

}

func (p *Project) Backup(filter func(string) bool, name string) {
	if len(p.Robots) <= 0 {
		fmt.Println("Your project does not have any robots. Please run `BackupTool add` to add one.")
		return
	}

	t := time.Now()

	fmt.Println("Backing up project...")
	dest := p.Destination + "/" + fmt.Sprintf("%d-%02d-%02dT%02d-%02d-%02d_%s", t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second(), name)

	var wg sync.WaitGroup
	for _, r := range p.Robots {
		wg.Add(1)
		go r.Backup(filter, dest, &wg)
	}
	wg.Wait()
	fmt.Printf("Backed up all robots in %v", time.Since(t))
}

func (p *Project) Save() {
	b, err := json.Marshal(p)
	check(err)
	err = ioutil.WriteFile("backup_tool.json", b, 0644)
	check(err)
	fmt.Println("Project saved.")
}

func (p *Project) RemoveRobot() {
	if len(p.Robots) <= 0 {
		fmt.Println("Your project does not have any robots. Please run `BackupTool add` to add one.")
		return
	}

list:
	for id, robot := range p.Robots {
		fmt.Printf("%d. %s %s\n", id+1, robot.Name, robot.Host)
	}
	fmt.Println("\nWhich robot do you want to remove?")
	var id int
	_, err := fmt.Scanf("%d", &id)
	check(err)
	id = id - 1
	if id < 0 || id > len(p.Robots)-1 {
		goto list
	}

	fmt.Printf("Removing robot #%d\n", id+1)
	p.Robots = append(p.Robots[:id], p.Robots[id+1:]...)
	p.Save()
}

func main() {
	p := InitProject()

	app := cli.NewApp()
	app.Name = "BackupTool"
	app.Usage = "Robot backups made easy by ONE Robotics Company."
	app.Version = VERSION
	app.Author = "Jay Strybis"
	app.Email = "jstrybis@onerobotics.com"
	app.Commands = []cli.Command{
		{
			Name:      "add",
			ShortName: "a",
			Usage:     "add a robot to a project",
			Action: func(c *cli.Context) {
				p.AddRobot()
			},
		},
		{
			Name:      "backup",
			ShortName: "b",
			Usage:     "Backup files on all robots",
			Subcommands: []cli.Command{
				{
					Name:  "all",
					Usage: "*.*",
					Action: func(c *cli.Context) {
						p.Backup(func(filename string) bool { return true }, "all")
					},
				},
				{
					Name:  "tp",
					Usage: "*.tp",
					Action: func(c *cli.Context) {
						p.Backup(func(filename string) bool {
							if filepath.Ext(filename) == ".tp" {
								return true
							} else {
								return false
							}
						}, "tp")
					},
				},
				{
					Name:  "ls",
					Usage: "*.ls",
					Action: func(c *cli.Context) {
						p.Backup(func(filename string) bool {
							if filepath.Ext(filename) == ".ls" {
								return true
							} else {
								return false
							}
						}, "ls")
					},
				},
				{
					Name:  "vr",
					Usage: "*.vr",
					Action: func(c *cli.Context) {
						p.Backup(func(filename string) bool {
							if filepath.Ext(filename) == ".vr" {
								return true
							} else {
								return false
							}
						}, "vr")
					},
				},
				{
					Name:  "va",
					Usage: "*.va",
					Action: func(c *cli.Context) {
						p.Backup(func(filename string) bool {
							if filepath.Ext(filename) == ".va" {
								return true
							} else {
								return false
							}
						}, "va")
					},
				},
				{
					Name:  "sv",
					Usage: "*.sv",
					Action: func(c *cli.Context) {
						p.Backup(func(filename string) bool {
							if filepath.Ext(filename) == ".sv" {
								return true
							} else {
								return false
							}
						}, "sv")
					},
				},
				{
					Name:  "vision",
					Usage: "*.vd, *.vda, *.zip",
					Action: func(c *cli.Context) {
						p.Backup(func(filename string) bool {
							switch filepath.Ext(filename) {
							case ".vd", ".vda", ".zip":
								return true
							}
							return false
						}, "vision")
					},
				},
				{
					Name:  "app",
					Usage: "*.tp, numreg.vr, posreg.vr",
					Action: func(c *cli.Context) {
						p.Backup(func(filename string) bool {
							switch filepath.Ext(filename) {
							case ".tp":
								return true
							}
							switch filename {
							case "numreg.vr", "posreg.vr":
								return true
							}
							return false
						}, "app")
					},
				},
				{
					Name:  "ascii",
					Usage: "*.ls, *.va, *.dat, *.dg, *.xml",
					Action: func(c *cli.Context) {
						p.Backup(func(filename string) bool {
							switch filepath.Ext(filename) {
							case ".ls", ".va", ".dat", ".dg", ".xml":
								return true
							}
							return false
						}, "ascii")
					},
				},
				{
					Name:  "bin",
					Usage: "*.zip, *.sv, *.tp, *.vr",
					Action: func(c *cli.Context) {
						p.Backup(func(filename string) bool {
							switch filepath.Ext(filename) {
							case ".zip", ".sv", ".tp", ".vr":
								return true
							}
							return false
						}, "bin")
					},
				},
			},
		},
		{
			Name:      "remove",
			ShortName: "r",
			Usage:     "remove a robot from a project",
			Action: func(c *cli.Context) {
				p.RemoveRobot()
			},
		},
	}

	app.Run(os.Args)
}
