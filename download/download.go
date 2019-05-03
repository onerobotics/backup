package download

import (
	"fmt"
	"log"

	"github.com/onerobotics/backup/ftp"
)

type Downloader struct {
	Id    int
	Dest  string
	Debug bool
	conn  *ftp.Connection
	Count int
}

func NewDownloader(id int, host string, port string, dest string) *Downloader {
	var d Downloader

	d.conn = ftp.NewConnection(host, port)
	//d.conn.Debug = true
	d.Id = id
	d.Dest = dest
	//d.Debug = true

	return &d
}

func (d *Downloader) debugf(format string, args ...interface{}) {
	if d.Debug {
		var a []interface{}
		a = append(a, d.Id)
		a = append(a, args...)
		log.Printf("Worker %d: "+format, a...)
	}
}

func (d *Downloader) Connect() <-chan error {
	out := make(chan error)

	go func() {
		defer close(out)

		d.debugf("Connecting...\n")
		err := d.conn.Connect()
		if err != nil {
			out <- err
			return
		}
		d.debugf("Connected.\n")

		err = d.conn.Type("I")
		if err != nil {
			out <- err
			return
		}
	}()

	return out
}

type FilterFunc func(filename string) bool

func AllFiles(filename string) bool { return true }

func (d *Downloader) Ls(filter FilterFunc) (<-chan string, <-chan error) {
	filec := make(chan string)
	errc := make(chan error)

	go func() {
		defer close(filec)

		d.debugf("Getting list of files...\n")
		files, err := d.conn.NameList()
		if err != nil {
			errc <- err
			return
		}
		close(errc)

		for _, f := range files {
			if f[0] == '-' {
				// don't download -bckedit- files
				continue
			}

			if filter(f) {
				filec <- f
			}
		}
	}()

	return filec, errc
}

func (d *Downloader) Download(filec <-chan string) (<-chan string, <-chan error) {
	outc := make(chan string)
	errc := make(chan error)

	go func() {
		defer close(outc)
		defer close(errc)

		for f := range filec {
			d.debugf("Downloading %s\n", f)
			err := d.conn.Download(f, d.Dest)
			if err != nil {
				d.debugf("Error downloading file '%s': %s\n", f, err)
				errc <- fmt.Errorf("failed to download file '%s': %s", f, err)
				continue
			}

			d.debugf("...done with %s\n", f)
			outc <- f

			d.Count++
		}
	}()

	return outc, errc
}

func (d *Downloader) Quit() <-chan error {
	out := make(chan error)

	go func() {
		defer close(out)

		err := d.conn.Quit()
		if err != nil {
			out <- err
		}
	}()

	return out
}
