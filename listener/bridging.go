package listener

import (
	"io"
	"time"
)

func bridging(c0, c1 io.ReadWriteCloser, duration time.Duration) {
	defer c0.Close()
	defer c1.Close()

	done := make(chan bool, 1)
	go forwarding(c0, c1, done)
	go forwarding(c1, c0, done)
	<-done
	if duration > 0 {
		time.Sleep(duration)
	}
}
func forwarding(w io.WriteCloser, r io.ReadCloser, done chan<- bool) {
	defer forwardingDone(done)
	var (
		b       = make([]byte, 32*1024)
		n       int
		er, ew  error
		noerror = true
	)
	for noerror {
		n, er = r.Read(b)
		if n > 0 {
			_, ew = w.Write(b[:n])
			if ew != nil {
				w.Close()
				noerror = false
			}
		}
		if er != nil {
			r.Close()
			noerror = false
		}
	}
}
func forwardingDone(done chan<- bool) {
	done <- true
}
