// Copyright 2017 OpenChirp. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.
//
// March 13, 2017
// Craig Hesling <craig@hesling.com>

package ccboot

import (
	"testing"

	"github.com/jacobsa/go-serial/serial"
)

const (
	port = "/dev/ttyUSB0"
	// timeout in ms
	readTimeout = 1000
)

func TestCCBoot(t *testing.T) {
	// Set up options.
	options := serial.OpenOptions{
		PortName:              port,
		BaudRate:              115200,
		DataBits:              8,
		StopBits:              1,
		MinimumReadSize:       1,
		InterCharacterTimeout: readTimeout,
	}

	// Open the port
	t.Log("# Opening Serial")
	port, err := serial.Open(options)
	if err != nil {
		t.Errorf("serial.Open: %v", err)
	}
	// Make sure to close it later.
	defer port.Close()

	d := NewDevice(port)

	// Sync
	t.Log("# Syncing")
	if err = d.Sync(); err != nil {
		t.Errorf("Error connecting to device: %s\n", err.Error())
		return
	}
	t.Log("Syncronization Success")

	// Try Pinging a Few Times
	for i := 0; i < 3; i++ {
		t.Log("# Pinging")
		err = d.Ping()
		if err != nil {
			t.Errorf("Error pinging device: %s\n", err.Error())
		}
		t.Log("Ping success")
	}

	// Get Status
	t.Log("# Getting Status")
	status, err := d.GetStatus()
	if err != nil {
		t.Errorf("Error reading chip id: %s\n", err.Error())
	}
	t.Logf("Status is 0x%.2X = %s\n", byte(status), status.GetString())

	// Get Chip ID
	t.Log("# Getting Chip ID")
	id, err := d.GetChipID()
	if err != nil {
		t.Errorf("Error reading chip id: %s\n", err.Error())
	}
	t.Logf("Chip ID is 0x%X = %d\n", id, id)

	// Get Status a Few Times
	for i := 0; i < 3; i++ {
		t.Log("# Getting Status")
		status, err = d.GetStatus()
		if err != nil {
			t.Errorf("Error reading chip id: %s\n", err.Error())
		}
		t.Logf("Status is 0x%.2X = %s\n", byte(status), status.GetString())
	}

	// Bank Erase
	t.Log("# Bank Erasing Device")
	err = d.BankErase()
	if err != nil {
		t.Errorf("Error bank erasing chip: %s\n", err.Error())
	}
	t.Log("Device erased")

	// Get Status
	t.Log("# Getting Status")
	status, err = d.GetStatus()
	if err != nil {
		t.Errorf("Error reading chip id: %s\n", err.Error())
	}
	t.Logf("Status is 0x%.2X = %s\n", byte(status), status.GetString())

	// Reset Device
	t.Log("# Resetting Device")
	err = d.Reset()
	if err != nil {
		t.Errorf("Error resetting chip: %s\n", err.Error())
	}
	t.Log("Device reset")

	t.Log("# Exiting\n")
	return
}
