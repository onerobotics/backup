package ftp

import (
	"bufio"
	"errors"
	"io"
	"log"
	"net"
	"net/textproto"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
)

func check(e error) {
	if e != nil {
		log.Fatal("FATAL:", e)
	}
}

type Connection struct {
	c     net.Conn
	conn  *textproto.Conn
	addr  string
	port  string
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
	c.debugf("Connecting to", c.addr+":"+c.port)
	conn, err := net.Dial("tcp", c.addr+":"+c.port)
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

func (c *Connection) Passive() (net.Conn, error) {
	code, msg, err := c.Cmd(227, "PASV")
	if err != nil {
		return nil, err
	}
	c.debugResponse(code, msg)

	matches := passiveRegexp.FindStringSubmatch(msg)
	if matches == nil {
		return nil, errors.New("Cannot parse PASV response: " + msg)
	}

	ph, err := strconv.Atoi(matches[5])
	if err != nil {
		return nil, err
	}
	pl, err := strconv.Atoi(matches[6])
	if err != nil {
		return nil, err
	}
	port := strconv.Itoa((ph << 8) | pl)
	addr := strings.Join(matches[1:5], ".") + ":" + port

	timeout := 10 * time.Second
	dconn, err := net.DialTimeout("tcp", addr, timeout)
	if err != nil {
		return nil, err
	}

	return dconn, nil
}

// todo: support argument to namelist e.g. *.ls
func (c *Connection) NameList() ([]string, error) {
	dconn, err := c.Passive()
	if err != nil {
		return nil, err
	}
	defer dconn.Close()

	code, msg, err := c.Cmd(1, "NLST")
	if err != nil {
		return nil, err
	}
	c.debugResponse(code, msg)

	var files []string
	scanner := bufio.NewScanner(dconn)
	c.debug("Getting list of files...")
	for scanner.Scan() {
		files = append(files, scanner.Text())
	}
	err = scanner.Err()
	if err != nil {
		return nil, err
	}
	dconn.Close()

	c.debugf("Received list of %d files\n", len(files))

	c.debug("Waiting for response from main connection...")
	code, msg, err = c.conn.ReadResponse(226)
	if err != nil {
		return nil, err
	}
	c.debugResponse(code, msg)

	return files, nil
}

func (c *Connection) Download(filename string, dest string) error {
	if filename[0] == '-' {
		return nil
	}

	fo, err := os.Create(dest + "/" + filename)
	if err != nil {
		return err
	}
	defer fo.Close()

	w := bufio.NewWriter(fo)
	defer w.Flush()

	dconn, err := c.Passive()
	if err != nil {
		return err
	}
	defer dconn.Close()

	code, msg, err := c.Cmd(1, "RETR %s", filename)
	if err != nil {
		return err
	}

	_, err = io.Copy(w, dconn)
	if err != nil {
		return err
	}

	dconn.Close()

	code, msg, err = c.conn.ReadResponse(2)
	if err != nil {
		return err
	}
	c.debugResponse(code, msg)

	return nil
}
