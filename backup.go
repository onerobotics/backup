package main

import (
	"log"
	"os"
	"path/filepath"

	"github.com/codegangsta/cli"
	"github.com/onerobotics/backup/project"
)



func main() {
	p, err := project.Init()
	if err != nil {
		log.Fatal(err)
	}

	app := cli.NewApp()
	app.Name = "BackupTool"
	app.Usage = "Robot backups made easy by ONE Robotics Company."
	app.Version = project.VERSION
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
