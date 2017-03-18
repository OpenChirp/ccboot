package ccboot

// CC_SYNC contains the bootloader sync words
var CC_SYNC = []byte{0x55, 0x55}

const (
	CC_ACK  byte = 0xCC
	CC_NACK byte = 0x33
)

type CommandType byte

// CommandType constants
const (
	COMMAND_PING         = CommandType(0x20)
	COMMAND_DOWNLOAD     = CommandType(0x21)
	COMMAND_GET_STATUS   = CommandType(0x23)
	COMMAND_SEND_DATA    = CommandType(0x24)
	COMMAND_RESET        = CommandType(0x25)
	COMMAND_SECTOR_ERASE = CommandType(0x26)
	COMMAND_CRC32        = CommandType(0x27)
	COMMAND_GET_CHIP_ID  = CommandType(0x28)
	COMMAND_MEMORY_READ  = CommandType(0x2A)
	COMMAND_MEMORY_WRITE = CommandType(0x2B)
	COMMAND_BANK_ERASE   = CommandType(0x2C)
	COMMAND_SET_CCFG     = CommandType(0x2D)
)

var cmd2String = map[CommandType]string{
	COMMAND_PING:         "COMMAND_PING",
	COMMAND_DOWNLOAD:     "COMMAND_DOWNLOAD",
	COMMAND_GET_STATUS:   "COMMAND_GET_STATUS",
	COMMAND_SEND_DATA:    "COMMAND_SEND_DATA",
	COMMAND_RESET:        "COMMAND_RESET",
	COMMAND_SECTOR_ERASE: "COMMAND_SECTOR_ERASE",
	COMMAND_CRC32:        "COMMAND_CRC32",
	COMMAND_GET_CHIP_ID:  "COMMAND_GET_CHIP_ID",
	COMMAND_MEMORY_READ:  "COMMAND_MEMORY_READ",
	COMMAND_MEMORY_WRITE: "COMMAND_MEMORY_WRITE",
	COMMAND_BANK_ERASE:   "COMMAND_BANK_ERASE",
	COMMAND_SET_CCFG:     "COMMAND_SET_CCFG",
}

// Command represents the command type and paramerters
type Command struct {
	Type       CommandType
	Parameters []byte
}

func (c *Command) Marshal() []byte {
	size := 1 + len(c.Parameters)
	buf := make([]byte, size)
	buf[0] = byte(c.Type)
	copy(buf[1:], c.Parameters)
	return buf
}

func (c *Command) Unmarshal(data []byte) error {
	if len(data) < 1 {
		return ErrBadPacket
	}
	c.Type = CommandType(data[0])
	c.Parameters = data[1:]
	return nil
}

func (c CommandType) String() string {
	if str, ok := cmd2String[c]; ok {
		return str
	}
	return "NONE"
}

// Status represents the status received by the GetStatus command
type Status byte

// These constants are returned from COMMAND_GET_STATUS
const (
	COMMAND_RET_SUCCESS     = Status(0x40)
	COMMAND_RET_UNKNOW_CMD  = Status(0x41)
	COMMAND_RET_INVALID_CMD = Status(0x42)
	COMMAND_RET_INVALID_ADR = Status(0x43)
	COMMAND_RET_FLASH_FAIL  = Status(0x44)
)

var cmdret2String = map[Status]string{
	COMMAND_RET_SUCCESS:     "SUCCESS",
	COMMAND_RET_UNKNOW_CMD:  "UNKNOWN_CMD",
	COMMAND_RET_INVALID_CMD: "INVALID_CMD",
	COMMAND_RET_INVALID_ADR: "INVALID_ADR",
	COMMAND_RET_FLASH_FAIL:  "FLASH_FAIL",
}

func (s Status) String() string {
	if str, ok := cmdret2String[s]; ok {
		return str
	}
	return "NONE"
}

type ReadType byte

const (
	ReadType8Bit  = ReadType(0)
	ReadType32Bit = ReadType(1)
)

var readType2String = map[ReadType]string{
	ReadType8Bit:  "8BIT",
	ReadType32Bit: "32BIT",
}

func (rt ReadType) String() string {
	if str, ok := readType2String[rt]; ok {
		return str
	}
	return fmt.Sprintf("0x%X", byte(rt))
}

const (
	ReadMaxCount8Bit  = uint8(253)
	ReadMaxCount32Bit = uint8(63)
)

type CCFG_FieldID uint32

const (
	ID_SECTOR_PROT       = CCFG_FieldID(0)
	ID_IMAGE_VALID       = CCFG_FieldID(1)
	ID_TEST_TAP_LCK      = CCFG_FieldID(2)
	ID_PRCM_TAP_LCK      = CCFG_FieldID(3)
	ID_CPU_DAP_LCK       = CCFG_FieldID(4)
	ID_WUC_TAP_LCK       = CCFG_FieldID(5)
	ID_PBIST1_TAP_LCK    = CCFG_FieldID(6)
	ID_PBIST2_TAP_LCK    = CCFG_FieldID(7)
	ID_BANK_ERASE_DIS    = CCFG_FieldID(8)
	ID_CHIP_ERASE_DIS    = CCFG_FieldID(9)
	ID_TI_FA_ENABLE      = CCFG_FieldID(10)
	ID_BL_BACKDOOR_EN    = CCFG_FieldID(11)
	ID_BL_BACKDOOR_PIN   = CCFG_FieldID(12)
	ID_BL_BACKDOOR_LEVEL = CCFG_FieldID(13)
	ID_BL_ENABLE         = CCFG_FieldID(14)
)
