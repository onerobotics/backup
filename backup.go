package main

import (
	"log"
	"bufio"
	"encoding/json"
	"fmt"
	"github.com/codegangsta/cli"
	"io/ioutil"
	"os"
	"strings"
	"sync"
	"time"
	"github.com/unreal/backup/ftp"
)


const VERSION = "0.1.0"

type Project struct {
	Destination string
	Version string
	Robots []Robot
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

func (r *Robot) Backup(destination string, wg *sync.WaitGroup) {
	defer wg.Done()
	t := time.Now()

	fmt.Println("Backing up", r.Name)
	dirname := destination + "/" + r.Name
	err := os.MkdirAll(dirname, os.ModePerm)
	check(err)

	c := ftp.NewConnection(r.Host, "21")
	c.Connect()
	defer c.Quit()

	files := c.NameList()
	
	c.Type("I")

	for _, file := range files {
		log.Printf("%s: Downloading %s", r.Name, file)
		c.Download(file, dirname)
	}

	fmt.Printf("Finished backing up %s in %v\n", r.Name, time.Since(t))
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

func (p *Project) Backup() {
	if len(p.Robots) <= 0 {
		fmt.Println("Your project does not have any robots. Please run `BackupTool add` to add one.")
		return
	}

	t := time.Now()

	fmt.Println("Backing up project...")
	dest := p.Destination + "/" + fmt.Sprintf("%d-%02d-%02dT%02d-%02d-%02d", t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second())

	var wg sync.WaitGroup
	for _, r := range p.Robots {
		wg.Add(1)
		go r.Backup(dest, &wg)
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
	app.Email = "jay.strybis@gmail.com"
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
						p.Backup()
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
