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
	"log"
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

type Device struct {
	port io.ReadWriteCloser
}

// NewDevice sets up a new CC bootloader device.
//
// We assume that port.Read has some timeout set
func NewDevice(port io.ReadWriteCloser) *Device {
	return &Device{port}
}

//////////////////////////////////////////////////////////////////////
//                   Low Level Serial Interface                     //
//////////////////////////////////////////////////////////////////////

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

//////////////////////////////////////////////////////////////////////
//                   Packet Abstration Layer                        //
//////////////////////////////////////////////////////////////////////

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

//////////////////////////////////////////////////////////////////////
//                      High Level Commands                         //
//////////////////////////////////////////////////////////////////////

func (d *Device) Ping() error {
	return d.SendPacket(encodeCmdPacket(COMMAND_PING, nil))
}

// Download indicates to the bootloader where to store data in flash
// and how many bytes will be sent by the following SendData command.
//
// This command must be followed by a GetStatus command to ensure that
// the program address and program size are valid for the device.
func (d *Device) Download(address, size uint32) error {
	data := []byte{
		byte((address >> (3 * 8)) & 0xFF),
		byte((address >> (2 * 8)) & 0xFF),
		byte((address >> (1 * 8)) & 0xFF),
		byte((address >> (0 * 8)) & 0xFF),
		byte((size >> (3 * 8)) & 0xFF),
		byte((size >> (2 * 8)) & 0xFF),
		byte((size >> (1 * 8)) & 0xFF),
		byte((size >> (0 * 8)) & 0xFF),
	}
	err := d.SendPacket(encodeCmdPacket(COMMAND_DOWNLOAD, data))
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
// 252 is max data size
func (d *Device) SendData(data []byte) error {
	if len(data) > SendDataMaxSize {
		return ErrBadArguments
	}
	return d.SendPacket(encodeCmdPacket(COMMAND_SEND_DATA, data))
}

func (d *Device) SectorErase(address uint32) error {
	data := []byte{
		byte((address >> (3 * 8)) & 0xFF),
		byte((address >> (2 * 8)) & 0xFF),
		byte((address >> (1 * 8)) & 0xFF),
		byte((address >> (0 * 8)) & 0xFF),
	}
	return d.SendPacket(encodeCmdPacket(COMMAND_SECTOR_ERASE, data))
}

func (d *Device) GetStatus() (Status, error) {
	err := d.SendPacket(encodeCmdPacket(COMMAND_GET_STATUS, nil))
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
	return d.SendPacket(encodeCmdPacket(COMMAND_RESET, nil))
}

func (d *Device) GetChipID() (uint32, error) {
	var id uint32
	err := d.SendPacket(encodeCmdPacket(COMMAND_GET_CHIP_ID, nil))
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
	id |= uint32(data[0]) << (3 * 8)
	id |= uint32(data[1]) << (2 * 8)
	id |= uint32(data[2]) << (1 * 8)
	id |= uint32(data[3]) << (0 * 8)
	return id, nil
}

func (d *Device) CRC32(address, size, rcount uint32) (uint32, error) {
	var crc uint32
	data := []byte{
		byte((address >> (3 * 8)) & 0xFF),
		byte((address >> (2 * 8)) & 0xFF),
		byte((address >> (1 * 8)) & 0xFF),
		byte((address >> (0 * 8)) & 0xFF),
		byte((size >> (3 * 8)) & 0xFF),
		byte((size >> (2 * 8)) & 0xFF),
		byte((size >> (1 * 8)) & 0xFF),
		byte((size >> (0 * 8)) & 0xFF),
		byte((rcount >> (3 * 8)) & 0xFF),
		byte((rcount >> (2 * 8)) & 0xFF),
		byte((rcount >> (1 * 8)) & 0xFF),
		byte((rcount >> (0 * 8)) & 0xFF),
	}
	err := d.SendPacket(encodeCmdPacket(COMMAND_CRC32, data))
	if err != nil {
		return 0, err
	}
	data, err = d.RecvPacket()
	if err != nil {
		return 0, err
	}
	crc |= uint32(data[0]) << (3 * 8)
	crc |= uint32(data[1]) << (2 * 8)
	crc |= uint32(data[2]) << (1 * 8)
	crc |= uint32(data[3]) << (0 * 8)
	return crc, nil
}

func (d *Device) BankErase() error {
	return d.SendPacket(encodeCmdPacket(COMMAND_BANK_ERASE, nil))
}

func (d *Device) MemoryRead(address uint32, typ ReadWriteType, count uint8) ([]byte, error) {
	if typ == ReadWriteType8Bit && count > ReadMaxCount8Bit {
		return nil, ErrBadArguments
	}
	if typ == ReadWriteType32Bit && count > ReadMaxCount32Bit {
		return nil, ErrBadArguments
	}
	data := []byte{
		byte((address >> (3 * 8)) & 0xFF),
		byte((address >> (2 * 8)) & 0xFF),
		byte((address >> (1 * 8)) & 0xFF),
		byte((address >> (0 * 8)) & 0xFF),
		byte(typ),
		byte(count),
	}
	err := d.SendPacket(encodeCmdPacket(COMMAND_MEMORY_READ, data))
	if err != nil {
		return nil, err
	}
	data, err = d.RecvPacket()
	if err != nil {
		return nil, err
	}
	return data, nil
}

func (d *Device) MemoryWrite(address uint32, typ ReadWriteType, data []byte) error {
	if typ == ReadWriteType8Bit && uint8(len(data)) > WriteMaxCount8Bit {
		return ErrBadArguments
	}
	if typ == ReadWriteType32Bit && uint8(len(data)) > WriteMaxCount32Bit {
		return ErrBadArguments
	}

	if (typ == ReadWriteType32Bit) && (len(data)%4 != 0) {
		log.Printf("# Error Detected: Tying to write a non-multiple of 4 bytes in 32bit mode\n")
	}

	buf := []byte{
		byte((address >> (3 * 8)) & 0xFF),
		byte((address >> (2 * 8)) & 0xFF),
		byte((address >> (1 * 8)) & 0xFF),
		byte((address >> (0 * 8)) & 0xFF),
		byte(typ),
	}
	buf = append(buf, data...)
	err := d.SendPacket(encodeCmdPacket(COMMAND_MEMORY_WRITE, buf))
	if err != nil {
		return err
	}
	return nil
}

func (d *Device) SetCCFG(id CCFG_FieldID, value uint32) error {
	data := []byte{
		byte((id >> (3 * 8)) & 0xFF),
		byte((id >> (2 * 8)) & 0xFF),
		byte((id >> (1 * 8)) & 0xFF),
		byte((id >> (0 * 8)) & 0xFF),
		byte((value >> (3 * 8)) & 0xFF),
		byte((value >> (2 * 8)) & 0xFF),
		byte((value >> (1 * 8)) & 0xFF),
		byte((value >> (0 * 8)) & 0xFF),
	}
	err := d.SendPacket(encodeCmdPacket(COMMAND_CRC32, data))
	if err != nil {
		return err
	}
	return nil
}

//////////////////////////////////////////////////////////////////////
//             Marshaling and Unmarshaling Helpers                  //
//////////////////////////////////////////////////////////////////////

// checksum calculates the checksum of the data as specified by the
// CC1650 bootloader spec
func checksum(data []byte) byte {
	var sum byte = 0x00
	for _, b := range data {
		sum += b
	}
	return sum
}

// encodeSize turns size into a byte modulo 256. This is actually not needed,
// since packets areen't allowed to be larger than 0xFF anyways.
func encodeSize(size int) byte {
	return byte(size & 0xFF)
}

func decodeUint32(value []byte) uint32 {
	u := uint32(0)
	msbShift := uint(3)
	if len(value) < 4 {
		msbShift = uint(len(value)) - 1
	}
	for i := 0; i < len(value) && i < 4; i++ {
		u |= uint32(uint8(value[i])) << ((msbShift - uint(i)) * 8)
	}

	return u
}

func encodePacket(data []byte) []byte {
	size := 2 + len(data)
	buf := make([]byte, size)

	buf[0] = encodeSize(size)
	buf[1] = checksum(data)
	copy(buf[2:], data)
	return buf
}

// decodePacket returns the packet data or an error if the packet
// was malformed
func decodePacket(pkt []byte) ([]byte, error) {
	// sanity check min size of packet
	if len(pkt) < 3 {
		return nil, ErrBadPacket
	}
	// check the size written in packet
	if encodeSize(len(pkt)) != pkt[0] {
		return nil, ErrBadPacket
	}
	// check the packet checksuum
	if checksum(pkt[2:]) != pkt[1] {
		return nil, ErrBadPacket
	}
	// return data of packet
	return pkt[2:], nil
}

// encodeCmdPacket encodes a command and parameters into a packet
func encodeCmdPacket(cmd CommandType, parameters []byte) []byte {
	command := Command{cmd, parameters}
	return encodePacket(command.Marshal())
}

// decodeCmdPacket decodes a command and parameters from a packet
func decodeCmdPacket(pkt []byte) (Command, error) {
	command := Command{}
	data, err := decodePacket(pkt)
	if err != nil {
		return command, err
	}
	err = command.Unmarshal(data)
	if err != nil {
		return command, err
	}
	return command, nil
}
