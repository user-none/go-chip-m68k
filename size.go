package m68k

// Size represents the operand width of a memory access or ALU operation.
type Size int

const (
	Byte Size = 1
	Word Size = 2
	Long Size = 4
)

// Mask returns a bitmask covering the valid bits for this size.
func (s Size) Mask() uint32 {
	switch s {
	case Byte:
		return 0xFF
	case Word:
		return 0xFFFF
	case Long:
		return 0xFFFFFFFF
	default:
		return 0
	}
}

// MSB returns the most-significant bit position for this size.
func (s Size) MSB() uint32 {
	switch s {
	case Byte:
		return 0x80
	case Word:
		return 0x8000
	case Long:
		return 0x80000000
	default:
		return 0
	}
}

// Bits returns the number of bits for this size.
func (s Size) Bits() uint32 {
	return uint32(s) * 8
}

// String returns a human-readable name for this size.
func (s Size) String() string {
	switch s {
	case Byte:
		return "byte"
	case Word:
		return "word"
	case Long:
		return "long"
	default:
		return "unknown"
	}
}
