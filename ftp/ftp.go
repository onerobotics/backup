package ftp

import (
	"log"
 	"net"
	"net/textproto"
	"regexp"
	"strconv"
	"strings"
	"time"
	"os"
	"io"
	"bufio"
)

func check(e error) {
	if e != nil {
		log.Fatal("FATAL:", e)
	}
}

type Connection struct {
	c net.Conn
	conn *textproto.Conn
	addr string
	port string
	Debug bool
}

func NewConnection(addr string, port string) *Connection {
	var c Connection
	c.addr = addr
	c.port = port
	return &c
}

func (c *Connection) debug(v ...interface{}) {
	if c.Debug {
		log.Println(v...)
	}
}

func (c *Connection) debugf(format string, v ...interface{}) {
	if c.Debug {
		log.Printf(format, v...)
	}
}

func (c *Connection) debugResponse(code int, msg string) {
	if c.Debug {
	  log.Printf("code: %d, msg: %v\n", code, msg)
	}
}

func (c *Connection) Connect() {
	c.debugf("Connecting to", c.addr + ":" + c.port)
	conn, err := net.Dial("tcp", c.addr + ":" + c.port)
	check(err)
	c.c = conn

	c.conn = textproto.NewConn(conn)
	code, msg, err := c.conn.ReadResponse(2)
	check(err)
	c.debugResponse(code, msg)
}

func (c *Connection) Cmd(exp int, format string, args ...interface{}) (code int, msg string, err error) {
	err = c.conn.PrintfLine(format, args...)
	if err != nil {
		return 0, "", err
	}
	code, msg, err = c.conn.ReadResponse(exp)
	return
}

func (c *Connection) Quit() {
	code, msg, err := c.Cmd(221, "QUIT")
	check(err)
	c.debugResponse(code, msg)
}

func (c *Connection) Type(t string) {
	code, msg, err := c.Cmd(200, "TYPE %s", t)
	check(err)
	c.debugResponse(code, msg)
}

var passiveRegexp = regexp.MustCompile(`([\d]+),([\d]+),([\d]+),([\d]+),([\d]+),([\d]+)`)

func (c *Connection) Passive() net.Conn {
	code, msg, err := c.Cmd(227,"PASV")
	check(err)
	c.debugResponse(code, msg)

	matches := passiveRegexp.FindStringSubmatch(msg)
	if matches == nil {
		log.Fatal("Cannot parse PASV response", msg)
	}

	ph, err := strconv.Atoi(matches[5])
	check(err)
	pl, err := strconv.Atoi(matches[6])
	check(err)
	port := strconv.Itoa((ph << 8) | pl)
	addr := strings.Join(matches[1:5], ".") + ":" + port

	timeout := 10 * time.Second
	dconn, err := net.DialTimeout("tcp", addr, timeout)
	check(err)

	return dconn
}

// todo: support argument to namelist e.g. *.ls
func (c *Connection) NameList() []string {
	dconn := c.Passive()
	defer dconn.Close()

	code, msg, err := c.Cmd(1,"NLST")
	check(err)
	c.debugResponse(code, msg)

	var files []string
	scanner := bufio.NewScanner(dconn)
	c.debug("Getting list of files...")
	for scanner.Scan() {
		files = append(files, scanner.Text())
	}
	err = scanner.Err()
	check(err)
	dconn.Close()

	c.debugf("Received list of %d files\n", len(files))

	c.debug("Waiting for response from main connection...")
	code, msg, err = c.conn.ReadResponse(226)
	check(err)
	c.debugResponse(code, msg)

	return files
}

func (c *Connection) Download(filename string, dest string) {
	if filename[0] == '-' {
		return
	}

	fo, err := os.Create(dest + "/" + filename)
	check(err)
	defer fo.Close()

	w := bufio.NewWriter(fo)
	defer w.Flush()

	dconn := c.Passive()
	defer dconn.Close()

	code, msg, err := c.Cmd(1, "RETR %s", filename)

	_, err = io.Copy(w, dconn)
	check(err)

	dconn.Close()

	code, msg, err = c.conn.ReadResponse(2)
	check(err)
	c.debugResponse(code, msg)
}
