package serialrtu

import (
	"fmt"
	"testing"
)

const (
	PortName              = "com4"
	BaudRate              = 9600
	DataBits              = 8
	StopBits              = 1
	Parity                = "N"
	Timeout               = 1000
	InterCharacterTimeout = 100
	MinimumReadSize       = 0
)

func TestOpen(t *testing.T) {
	c, err := Open(Config{
		PortName:              PortName,
		BaudRate:              BaudRate,
		DataBits:              DataBits,
		StopBits:              StopBits,
		Parity:                Parity,
		Timeout:               Timeout,
		InterCharacterTimeout: InterCharacterTimeout,
		MinimumReadSize:       MinimumReadSize,
		RS485:                 RS485Config{},
	})
	defer func() {
		if c == nil {
			c.Close()
		}
	}()

	if err != nil {
		t.Error(err)
		return
	}
}

func TestClose(t *testing.T) {
	c, err := Open(Config{
		PortName:              PortName,
		BaudRate:              BaudRate,
		DataBits:              DataBits,
		StopBits:              StopBits,
		Parity:                Parity,
		Timeout:               Timeout,
		InterCharacterTimeout: InterCharacterTimeout,
		MinimumReadSize:       MinimumReadSize,
		RS485:                 RS485Config{},
	})

	if err != nil {
		t.Error(err)
		return
	}

	if err = c.Close(); err == nil {
		t.Error(err)
	}
}

func TestWrite(t *testing.T) {
	c, err := Open(Config{
		PortName:              PortName,
		BaudRate:              BaudRate,
		DataBits:              DataBits,
		StopBits:              StopBits,
		Parity:                Parity,
		Timeout:               Timeout,
		InterCharacterTimeout: InterCharacterTimeout,
		MinimumReadSize:       MinimumReadSize,
		RS485:                 RS485Config{},
	})

	if err != nil {
		t.Error(err)
		return
	}

	defer c.Close()

	n, err := c.Write([]byte("Hello"))

	fmt.Println(n)
	if err == nil {
		t.Error(err)
	}
}
