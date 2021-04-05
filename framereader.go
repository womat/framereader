package framereader

import (
	"encoding/hex"
	"io"
	"os"
	"time"

	"github.com/womat/debug"
)

func init() {
	debug.SetDebug(os.Stderr, debug.Standard)
}

func (r *Reader) frameReader() {
	defer func() {
		debug.InfoLog.Print("stop frame reader services")
		close(r.dataChan)
	}()
	data := make(chan []byte)

	go func() { // this goroutine reads data from *Reader, until reader is closed reader.Closed
		defer func() {
			debug.InfoLog.Print("stop serialport reader")
			close(data)
		}()
		for !r.closed {
			buffer := make([]byte, frameSize)
			if n, _ := r.reader.Read(buffer); n > 0 {
				debug.TraceLog.Printf("read %v byte(s) from serial port: %v", n, hex.EncodeToString(buffer[:n]))
				data <- buffer[:n]
			}
		}
	}()

	for !r.closed { // this goroutine reads data from *Reader, until reader is closed reader.Closed
		var icd, icdMax time.Duration
		buffer := make([]byte, frameSize)

		n, err := func(frame []byte) (int, error) {
			count := 0
			for { // this goroutine reads a frame: appends data from serial port, until the delay between characters greater then the interFrameDelay
				t := time.Now()
				timeout := time.NewTimer(r.interFrameDelay)
				select {
				case chunk, ok := <-data:
					if count > 0 {
						icd = time.Since(t)
					}

					debug.TraceLog.Printf("read new chunk (icd): (%v) %v", icd, hex.EncodeToString(chunk))

					if icd > icdMax {
						// calc icdMax of the received Frame
						icdMax = icd
					}

					for i := 0; count < len(frame) && i < len(chunk); i++ {
						frame[count] = chunk[i]
						count++
					}

					if !ok { // the channel is closed, no more characters can received
						debug.TraceLog.Print("the channel is closed, no more characters can received, exit with EOF")
						return count, io.EOF
					}

				case <-timeout.C:
					icd = time.Since(t)
					return count, nil
				}
			}
		}(buffer)

		if err != nil {
			debug.InfoLog.Print("the channel is closed, no more characters can received, stop service")
			return
		}

		if n > 0 {
			// New Frame received
			debug.DebugLog.Printf("read new frame (ifd/icdMax): (%v/%v) %v", icd, icdMax, hex.EncodeToString(buffer[:n]))
			r.dataChan <- buffer[:n]
		}
	}
}
