// Copyright 2017 OpenChirp. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.
//
// March 13, 2017
// Craig Hesling <craig@hesling.com>

// Package ccboot provides the low level interface to the CC2650
// bootloader. This may be similar enough to other CC chips.
//
// Used the bootloader interface described in section 8.2 of
// the following datasheet:
// http://www.ti.com/lit/ug/swcu117g/swcu117g.pdf
package ccboot

import (
	"io"
	"time"

	"errors"
)

const (
	// numAttempts is the number of connect attempts
	numAttempts int = 3
)

var ErrSerial = errors.New("Error interacting with reader or writer")

var ErrDevice = errors.New("Unexpected error from device")

var ErrDeviceTimeout = errors.New("Timed out waiting for device")

var ErrBadPacket = errors.New("The received packet was malformed")

var ErrBadArguments = errors.New("The arguments supplied are invalid")

var ErrNotImplemented = errors.New("This method is not implemented yet")

// CC_SYNC contains the bootloader sync words
var CC_SYNC = []byte{0x55, 0x55}

const (
	CC_ACK  byte = 0xCC
	CC_NACK byte = 0x33
)

const (
	CC_COMMAND_PING         byte = 0x20
	CC_COMMAND_DOWNLOAD     byte = 0x21
	CC_COMMAND_GET_STATUS   byte = 0x23
	CC_COMMAND_SEND_DATA    byte = 0x24
	CC_COMMAND_RESET        byte = 0x25
	CC_COMMAND_SECTOR_ERASE byte = 0x26
	CC_COMMAND_CRC32        byte = 0x27
	CC_COMMAND_GET_CHIP_ID  byte = 0x28
	CC_COMMAND_MEMORY_READ  byte = 0x2A
	CC_COMMAND_MEMORY_WRITE byte = 0x2B
	CC_COMMAND_BANK_ERASE   byte = 0x2C
	CC_COMMAND_SET_CCFG     byte = 0x2D
)

type Status byte

func (s Status) GetString() string {
	switch s {
	case CC_COMMAND_RET_SUCCESS:
		return "SUCCESS"
	case CC_COMMAND_RET_UNKNOW_CMD:
		return "UNKNOWN_CMD"
	case CC_COMMAND_RET_INVALID_CMD:
		return "INVALID_CMD"
	case CC_COMMAND_RET_INVALID_ADR:
		return "INVALID_ADR"
	case CC_COMMAND_RET_FLASH_FAIL:
		return "FLASH_FAIL"
	default:
		return "NONE"
	}
}

// These constants are returned from COMMAND_GET_STATUS
const (
	CC_COMMAND_RET_SUCCESS     Status = 0x40
	CC_COMMAND_RET_UNKNOW_CMD  Status = 0x41
	CC_COMMAND_RET_INVALID_CMD Status = 0x42
	CC_COMMAND_RET_INVALID_ADR Status = 0x43
	CC_COMMAND_RET_FLASH_FAIL  Status = 0x44
)

// checksum calculates the checksum of the data as specified by the
// CC1650 bootloader spec
func checksum(data []byte) byte {
	var sum byte = 0x00
	for _, b := range data {
		sum += b
	}
	return sum
}

func encodeSize(size int) byte {
	return byte(size & 0xFF)
}

// encodePacket encodes a command and parameters into a packet
func encodePacket(cmd byte, parameters []byte) []byte {
	size := 3 + len(parameters)
	buf := make([]byte, size)

	buf[0] = encodeSize(size)
	buf[2] = cmd
	copy(buf[3:], parameters)
	buf[1] = checksum(buf[2:])
	return buf
}

// decodePacket returns the packet data or an error if the packet
// was malformed
func decodePacket(pkt []byte) ([]byte, error) {
	if len(pkt) < 3 {
		return nil, ErrBadPacket
	}
	if encodeSize(len(pkt)) != pkt[0] {
		return nil, ErrBadPacket
	}
	if checksum(pkt[2:]) != pkt[1] {
		return nil, ErrBadPacket
	}
	return pkt[2:], nil
}

type Device struct {
	port io.ReadWriteCloser
}

// NewDevice sets up a new CC bootloader device.
//
// We assume that port.Read has some timeout set
func NewDevice(port io.ReadWriteCloser) *Device {
	return &Device{port}
}

// Sync sends the sync command and waits for the device to respond
func (d *Device) Sync() error {
	for attempt := 0; attempt < numAttempts; attempt++ {
		buf := make([]byte, 100)
		n, err := d.port.Write(CC_SYNC)
		if err != nil {
			return err
		}
		if n != len(CC_SYNC) {
			return ErrSerial
		}
		time.Sleep(time.Millisecond * 10)
		n, err = d.port.Read(buf)
		if err != nil {
			return err
		}
		if n != 2 {
			continue
		}
		// For sync, it is actually said to return 0x00 and 0xCC
		if buf[0] == 0x00 && buf[1] == CC_ACK {
			// Success
			return nil
		}
	}

	// Could not connect and maxed out number of attempts
	return ErrDevice
}

func (d *Device) recvNonZero() (byte, error) {
	buf := make([]byte, 1)
	attempts := 0
	for {
		if attempts > numAttempts {
			return 0, ErrDeviceTimeout
		}

		n, err := d.port.Read(buf)
		if err != nil {
			return 0, err
		}

		if n == 0 {
			// timed out waiting for byte
			attempts++
			continue
		} else if n == 1 {
			// fmt.Printf("recv: 0x%.2X\n", buf[0])
			if buf[0] == 0x00 {
				// throw away zeros
				continue
			}
			// got an non-zero byte
			return buf[0], nil
		} else {
			// not sure what else n could be, must be serial interface
			return 0, ErrSerial
		}
	}
}

func (d *Device) recvByte() (byte, error) {
	buf := make([]byte, 1)
	attempts := 0
	for {
		if attempts > numAttempts {
			return 0, ErrDeviceTimeout
		}

		n, err := d.port.Read(buf)
		if err != nil {
			return 0, err
		}

		if n == 0 {
			// timed out waiting for byte
			attempts++
			continue
		} else if n == 1 {
			// fmt.Printf("recv: 0x%.2X\n", buf[0])
			// got an non-zero byte
			return buf[0], nil
		} else {
			// not sure what else n could be, must be serial interface
			return 0, ErrSerial
		}
	}
}

func (d *Device) recvAck() (byte, error) {
	b, err := d.recvNonZero()
	return b, err
}

func (d *Device) sendAck(ack byte) error {
	buf := make([]byte, 1)
	buf[0] = ack
	n, err := d.port.Write(buf)
	if err != nil {
		return err
	}
	if n != 1 {
		return ErrSerial
	}
	return nil
}

func (d *Device) SendPacket(pkt []byte) error {
	for attempt := 0; attempt < numAttempts; attempt++ {
		// fmt.Printf("Sending Packet: 0x%.2X\n", pkt)
		n, err := d.port.Write(pkt)
		if err != nil {
			return err
		}
		if n != len(pkt) {
			return ErrSerial
		}
		ack, err := d.recvAck()
		if err == ErrDeviceTimeout {
			// try again
			continue
		} else if err != nil {
			// bad serial error
			return err
		}
		if ack == CC_ACK {
			// success
			return nil
		}

		// don't care if it is a NACK or bad characters
		// try again
	}

	// we spent all of our attempts
	return ErrDevice
}

func (d *Device) RecvPacket() ([]byte, error) {
	for attempt := 0; attempt < numAttempts; attempt++ {
		// get packet start size byte
		size, err := d.recvNonZero()
		if err != nil {
			return nil, err
		}
		pkt := make([]byte, int(size))
		pkt[0] = size
		// get remaining packet bytes
		for count := 1; count < int(size); count++ {
			b, err := d.recvByte()
			if err != nil {
				return nil, err
			}
			pkt[count] = b
		}
		// decode and verify packet
		data, err := decodePacket(pkt)
		if err != nil {
			err = d.sendAck(CC_NACK)
			if err != nil {
				return nil, err
			}
			// sent NACK and try again
			continue
		}

		err = d.sendAck(CC_ACK)
		if err != nil {
			return nil, err
		}
		return data, nil
	}

	return nil, ErrDevice
}

func (d *Device) Ping() error {
	return d.SendPacket(encodePacket(CC_COMMAND_PING, nil))
}

// Download indicates to the bootloader where to store data in flash
// and how many bytes will be sent by the following SendData command.
//
// This command must be followed by a GetStatus command to ensure that
// the program address and program size are valid for the device.
func (d *Device) Download(address, size uint32) error {
	data := []byte{
		byte((address >> 3) & 0xFF),
		byte((address >> 2) & 0xFF),
		byte((address >> 1) & 0xFF),
		byte((address >> 0) & 0xFF),
		byte((size >> 3) & 0xFF),
		byte((size >> 2) & 0xFF),
		byte((size >> 1) & 0xFF),
		byte((size >> 0) & 0xFF),
	}
	err := d.SendPacket(encodePacket(CC_COMMAND_DOWNLOAD, data))
	if err != nil {
		return err
	}
	return nil
}

// SendData must only follow a Download command or another SendData
// command, if more data is needed.
// Consecutive SendData commands automatically increment the address
// and continue programming from the previous location.
//
// The command terminates programming when the number of bytes
// indicated by the Download command is received.
// Each time this function is called, send a GetStatus command to
// ensure that the data was successfully programmed into the flash.
func (d *Device) SendData(data []byte) error {
	if len(data) > 255-3 {
		return ErrBadArguments
	}
	return d.SendPacket(encodePacket(CC_COMMAND_SEND_DATA, data))
}

func (d *Device) SectorErase(address uint32) error {
	data := []byte{
		byte((address >> 3) & 0xFF),
		byte((address >> 2) & 0xFF),
		byte((address >> 1) & 0xFF),
		byte((address >> 0) & 0xFF),
	}
	return d.SendPacket(encodePacket(CC_COMMAND_SECTOR_ERASE, data))
}

func (d *Device) GetStatus() (Status, error) {
	err := d.SendPacket(encodePacket(CC_COMMAND_GET_STATUS, nil))
	if err != nil {
		return 0, err
	}
	data, err := d.RecvPacket()
	if err != nil {
		return 0, err
	}
	if len(data) != 1 {
		return Status(0), ErrDevice
	}
	return Status(data[0]), nil
}

func (d *Device) Reset() error {
	return d.SendPacket(encodePacket(CC_COMMAND_RESET, nil))
}

func (d *Device) GetChipID() (uint32, error) {
	var id uint32
	err := d.SendPacket(encodePacket(CC_COMMAND_GET_CHIP_ID, nil))
	if err != nil {
		return 0, err
	}
	data, err := d.RecvPacket()
	if err != nil {
		return 0, err
	}
	if len(data) != 4 {
		return 0, ErrDevice
	}
	id |= uint32(data[0]) << 3
	id |= uint32(data[1]) << 2
	id |= uint32(data[2]) << 1
	id |= uint32(data[3]) << 0
	return id, nil
}

func (d *Device) CRC32(address, size, rcount uint32) (uint32, error) {
	var crc uint32
	data := []byte{
		byte((address >> 3) & 0xFF),
		byte((address >> 2) & 0xFF),
		byte((address >> 1) & 0xFF),
		byte((address >> 0) & 0xFF),
		byte((size >> 3) & 0xFF),
		byte((size >> 2) & 0xFF),
		byte((size >> 1) & 0xFF),
		byte((size >> 0) & 0xFF),
		byte((rcount >> 3) & 0xFF),
		byte((rcount >> 2) & 0xFF),
		byte((rcount >> 1) & 0xFF),
		byte((rcount >> 0) & 0xFF),
	}
	err := d.SendPacket(encodePacket(CC_COMMAND_CRC32, data))
	if err != nil {
		return 0, err
	}
	data, err = d.RecvPacket()
	if err != nil {
		return 0, err
	}
	crc |= uint32(data[0]) << 3
	crc |= uint32(data[1]) << 2
	crc |= uint32(data[2]) << 1
	crc |= uint32(data[3]) << 0
	return crc, nil
}

func (d *Device) BankErase() error {
	return d.SendPacket(encodePacket(CC_COMMAND_BANK_ERASE, nil))
}

func (d *Device) MemoryRead() error {
	return ErrNotImplemented
}
