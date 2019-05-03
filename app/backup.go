package app

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/onerobotics/backup/download"
)

const MAX_WORKERS = 1

func BackupRobots(robots []Robot, destination string, filter download.FilterFunc) (<-chan string, <-chan error) {
	outc := make(chan string)
	outerrc := make(chan error)

	go func() {
		defer close(outc)
		defer close(outerrc)

		var wg sync.WaitGroup

		for _, r := range robots {
			wg.Add(1)
			go func(r Robot) {
				defer wg.Done()

				dest := filepath.Join(destination, r.Name)
				err := os.MkdirAll(dest, os.ModePerm)
				if err != nil {
					log.Fatal(err)
				}

				resultsc, errc := backupRobot(r, dest, filter)
				var nwg sync.WaitGroup
				nwg.Add(1)
				go func() {
					defer nwg.Done()
					for result := range resultsc {
						outc <- result
					}
				}()

				nwg.Add(1)
				go func() {
					defer nwg.Done()
					for err := range errc {
						outerrc <- err
					}
				}()
				nwg.Wait()
			}(r)
		}

		wg.Wait()
	}()

	return outc, outerrc
}

func backupRobot(r Robot, dest string, filter download.FilterFunc) (<-chan string, <-chan error) {
	outc := make(chan string)
	outerrc := make(chan error)

	go func() {
		t := time.Now()
		ls := make(chan int)
		go func() {
			ls <- 1
			close(ls)
		}()

		defer func() {
			outc <- fmt.Sprintf("%s: Finished in %s.", r.Name, time.Since(t))
			close(outc)
			close(outerrc)
		}()

		filesc := make(<-chan string)

		outc <- fmt.Sprintf("%s: Backup started with up to %d workers.", r.Name, MAX_WORKERS)

		var wg sync.WaitGroup
		for i := 0; i < MAX_WORKERS; i++ {
			wg.Add(1)
			go func(i int) {
				defer wg.Done()

				d := download.NewDownloader(i+1, r.Host, "21", dest)
				errc := d.Connect()
				err := <-errc
				if err != nil {
					outerrc <- fmt.Errorf("%s: Worker %d failed to connect.", r.Name, d.Id)
					return
				}

				// make sure all downloaders are finished
				// before issuing QUIT signal
				go func(d *download.Downloader) {
					wg.Wait()
					outc <- fmt.Sprintf("%s: Worker %d retrieved %d files.", r.Name, d.Id, d.Count)
					d.Quit()
				}(d)

				select {
				case val := <-ls:
					if val > 0 {
						filesc, errc = d.Ls(filter)
						err = <-errc
						if err != nil {
							outerrc <- fmt.Errorf("%s: Error getting list of files", r.Name)
							log.Fatal(err)
						}
					}
				}

				resultsc, errc := d.Download(filesc)
				var nwg sync.WaitGroup
				nwg.Add(1)
				go func() {
					defer nwg.Done()
					for result := range resultsc {
						outc <- fmt.Sprintf("%s: Worker %d downloaded: %s", r.Name, d.Id, result)
					}
				}()
				nwg.Add(1)
				go func() {
					defer nwg.Done()
					for err = range errc {
						outerrc <- fmt.Errorf("%s: Worker %d error: %s", r.Name, d.Id, err)
					}
				}()
				nwg.Wait()
			}(i)
		}

		wg.Wait()
	}()

	return outc, outerrc
}
