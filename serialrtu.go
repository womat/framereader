package serialrtu

import (
	"encoding/hex"
	"io"
	"log"
	"time"

	"github.com/jacobsa/go-serial/serial"
)

type serialPort struct {
	f               io.ReadWriteCloser
	rChan           chan []byte
	interframedelay time.Duration
	timeout         time.Duration
	closed          bool
}

// OpenOptions is the struct containing all of the options necessary for
// opening a serial port.
type Config struct {
	// The name of the port, e.g. "/dev/tty.usbserial-A8008HlV".
	PortName string
	// The baud rate for the port (default 19200)
	BaudRate uint
	// The number of data bits per frame. Legal values are 5, 6, 7, and 8 (default 8)
	DataBits uint
	// The number of stop bits per frame. Legal values are 1 and 2 (default 1)
	StopBits uint
	// The type of parity bits to use for the connection. Currently parity errors
	// are simply ignored; that is, bytes are delivered to the user no matter
	// whether they were received with a parity error or not.
	// Parity: N - None, E - Even, O - Odd (default E)
	Parity string
	// Read (Write) timeout.
	Timeout time.Duration
	// An inter-character timeout value, in milliseconds, and a minimum number of
	// bytes to block for on each read. A call to Read() that otherwise may block
	// waiting for more data will return immediately if the specified amount of
	// time elapses between successive bytes received from the device or if the
	// minimum number of bytes has been exceeded.
	//
	// Note that the inter-character timeout value may be rounded to the nearest
	// 100 ms on some systems, and that behavior is undefined if calls to Read
	// supply a buffer whose length is less than the minimum read size.
	//
	// Behaviors for various settings for these values are described below. For
	// more information, see the discussion of VMIN and VTIME here:
	//
	//     http://www.unixwiz.net/techtips/termios-vmin-vtime.html
	//
	// InterCharacterTimeout = 0 and MinimumReadSize = 0 (the default):
	//     This arrangement is not legal; you must explicitly set at least one of
	//     these fields to a positive number. (If MinimumReadSize is zero then
	//     InterCharacterTimeout must be at least 100.)
	//
	// InterCharacterTimeout > 0 and MinimumReadSize = 0
	//     If data is already available on the read queue, it is transferred to
	//     the caller's buffer and the Read() call returns immediately.
	//     Otherwise, the call blocks until some data arrives or the
	//     InterCharacterTimeout milliseconds elapse from the start of the call.
	//     Note that in this configuration, InterCharacterTimeout must be at
	//     least 100 ms.
	//
	// InterCharacterTimeout > 0 and MinimumReadSize > 0
	//     Calls to Read() return when at least MinimumReadSize bytes are
	//     available or when InterCharacterTimeout milliseconds elapse between
	//     received bytes. The inter-character timer is not started until the
	//     first byte arrives.
	//
	// InterCharacterTimeout = 0 and MinimumReadSize > 0
	//     Calls to Read() return only when at least MinimumReadSize bytes are
	//     available. The inter-character timer is not used.
	//
	// For windows usage, these options (termios) do not conform well to the
	//     windows serial port / comms abstractions.  Please see the code in
	//		 open_windows setCommTimeouts function for full documentation.
	//   	 Summary:
	//			Setting MinimumReadSize > 0 will cause the serialPort to block until
	//			until data is available on the port.
	//			Setting IntercharacterTimeout > 0 and MinimumReadSize == 0 will cause
	//			the port to either wait until IntercharacterTimeout wait time is
	//			exceeded OR there is character data to return from the port.
	//
	// MODBUS specification requires an inter-frame delay of 3.5 character times to determine end of a frame (transmission).
	InterframeDelay time.Duration
}

// Open creates an io.ReadWriteCloser based on the supplied options struct.
func Open(c Config) (io.ReadWriteCloser, error) {
	var err error

	o := serial.OpenOptions{
		PortName:              c.PortName,
		BaudRate:              c.BaudRate,
		DataBits:              c.DataBits,
		StopBits:              c.StopBits,
		MinimumReadSize:       0,
		InterCharacterTimeout: 100,
		ParityMode:            serial.PARITY_NONE,
	}

	switch c.Parity {
	case "O":
		o.ParityMode = serial.PARITY_ODD
	case "E":
		o.ParityMode = serial.PARITY_EVEN
	}

	port := new(serialPort)
	if port.f, err = serial.Open(o); err != nil {
		return port, err
	}
	port.interframedelay = c.InterframeDelay
	port.timeout = c.Timeout
	port.rChan = make(chan []byte, 10)

	go port.Serv()
	return port, err
}

func (p *serialPort) Close() error {
	p.closed = true
	return p.f.Close()
}

func (p *serialPort) Write(buf []byte) (int, error) {
	return p.f.Write(buf)
}

func (p *serialPort) Read(buf []byte) (n int, err error) {
	timeout := time.NewTimer(p.timeout)

	select {
	case b, ok := <-p.rChan:
		n = copy(buf, b)
		if !ok {
			err = io.EOF
		}
	case <-timeout.C:
		err = io.EOF
	}

	return
}

func (p *serialPort) Serv() {
	var err error
	var n int
	// max. inter-character delay
	var icdmax time.Duration

	frame := make([]byte, 0, 255)
	chunk := make([]byte, 255)

	for {
		if p.closed {
			break
		}

		t := time.Now()
		if n, err = p.f.Read(chunk); err != nil && err != io.EOF {
			log.Println("serialrtu ERROR: reading from serial port: ", err)
		}

		// measure the inter-character delay
		icd := time.Since(t)

		if icd > p.interframedelay && len(frame) > 0 {
			// New Frame received
			log.Printf("serialrtu read new frame (ifd/icdmax): (%v/%v) %v\n", icd, icdmax, hex.EncodeToString(frame))
			p.rChan <- frame

			// empty frame frame, be ready for new Frame
			frame = frame[0:0]
			icdmax = 0
		}

		if n > 0 {
			// add chunk to frame
			if len(frame) > 0 && icd > icdmax {
				// calc ict of the received Frame
				icdmax = icd
			}
			frame = append(frame, chunk[:n]...)
		}
	}
	close(p.rChan)
}
