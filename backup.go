package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/onerobotics/backup/app"
)

var filters map[string][]string
var robotNamelistFlag app.RobotNamelist

func init() {
	filters = make(map[string][]string)
	filters["all"] = []string{"*.*"}
	filters["tp"] = []string{"*.tp"}
	filters["ls"] = []string{"*.ls"}
	filters["vr"] = []string{"*.vr"}
	filters["va"] = []string{"*.va"}
	filters["sv"] = []string{"*.sv"}
	filters["vision"] = []string{"*.vd", "*.vda", "*.zip"}
	filters["app"] = []string{"*.tp", "numreg.vr", "posreg.vr"}
	filters["ascii"] = []string{"*.ls", "*.va", "*.dat", "*.dg", "*.xml"}
	filters["bin"] = []string{"*.zip", "*.sv", "*.tp", "*.vr"}

	flag.Var(&robotNamelistFlag, "r", "comma-separated list of robot names")
}

func usage() {
	fmt.Printf(`
BackupTool v%s
-----------------
FANUC robot backups made easy by ONE Robotics Company.

Author:		Jay Strybis <jstrybis@onerobotics.com>
Website:	https://www.onerobotics.com

Usage:
	
	backuptool command [arguments]

The commands are:

	add, a		Add a robot to a project
	backup, b	Perform a backup
	remove, r	Remove a robot from a project
	help, h		Show this screen or command-specific help
`, app.VERSION)
}

func addUsage() {
	fmt.Println(`
usage: backuptool add

Follow the instructions in the CLI wizard to add a robot
to the current project.`)
}

func backupUsage() {
	fmt.Println(`
usage: backuptool backup [flags] filter

The filters are:
	all	*.*
	tp	*.tp
	ls	*.ls
	vr	*.vr
	va	*.va
	sv	*.sv
	vision	*.vd, *.vda, *.zip
	app	*.tp, numreg.vr, posreg.vr
	ascii	*.ls, *.va, *.dat, *.dg, *.xml
	bin	*.zip, *.sv, *.tp, *.vr
	
The flags are:
	-r	comma-separated list of robot names
		Used to backup a subset of the project's robots
`)
}

func removeUsage() {
	fmt.Println(`
usage: backuptool remove

Follow the CLI wizard to remove a robot from your project`)
}

func main() {
	flag.Parse()
	args := flag.Args()

	p, err := app.NewProject()
	if err != nil {
		log.Fatal(err)
	}

	if len(args) < 1 {
		usage()
		os.Exit(0)
	}

	switch args[0] {
	case "add", "a":
		if len(args) > 1 {
			addUsage()
			os.Exit(1)
		}

		err := p.AddRobot()
		if err != nil {
			log.Fatal(err)
		}
	case "backup", "b":
		if len(args) < 2 {
			backupUsage()
			os.Exit(1)
		}

		filter, ok := filters[args[1]]
		if !ok {
			fmt.Printf("Invalid filter: %s\n", args[1])
			backupUsage()
			os.Exit(1)
		}

		err := p.Backup(robotNamelistFlag, func(filename string) bool {
			for _, f := range filter {
				if f[0] == '*' {
					return filepath.Ext(filename) == f[1:]
				} else {
					return filename == f
				}
			}

			return false
		}, args[1])
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	case "remove", "r":
		if len(args) > 1 {
			removeUsage()
			os.Exit(1)
		}

		err := p.RemoveRobot()
		if err != nil {
			log.Fatal(err)
		}
	case "help", "h":
		if len(args) < 2 {
			usage()
			return
		}

		switch args[1] {
		case "add", "a":
			addUsage()
		case "backup", "b":
			backupUsage()
		case "remove", "r":
			removeUsage()
		}
	default:
		usage()
	}
}
