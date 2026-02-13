package m68k

func init() {
	registerBTST()
	registerBCHG()
	registerBCLR()
	registerBSET()
}

// Bit operations have two forms:
// Dynamic: 0000 DDD1 00tt teee (Dn specifies bit number)
// Static:  0000 1000 00tt teee + immediate word (bit number in extension)
// tt = 00:BTST, 01:BCHG, 10:BCLR, 11:BSET
// For Dn destination: operates on long (bit mod 32)
// For memory: operates on byte (bit mod 8)

// --- BTST ---

func registerBTST() {
	// Dynamic form: BTST Dn,<ea> (includes immediate as source)
	for dn := uint16(0); dn < 8; dn++ {
		for mode := uint16(0); mode < 8; mode++ {
			if mode == 1 {
				continue
			}
			for reg := uint16(0); reg < 8; reg++ {
				if mode == 7 && reg > 4 {
					continue
				}
				opcode := 0x0100 | dn<<9 | mode<<3 | reg
				opcodeTable[opcode] = opBTSTdyn
			}
		}
	}
	// Static form: BTST #imm,<ea>
	for mode := uint16(0); mode < 8; mode++ {
		if mode == 1 {
			continue
		}
		for reg := uint16(0); reg < 8; reg++ {
			if mode == 7 && reg > 3 {
				continue
			}
			opcode := 0x0800 | mode<<3 | reg
			opcodeTable[opcode] = opBTSTstatic
		}
	}
}

func opBTSTdyn(c *CPU) {
	dn := (c.ir >> 9) & 7
	mode := uint8((c.ir >> 3) & 7)
	reg := uint8(c.ir & 7)
	bitNum := c.reg.D[dn]

	if mode == 0 {
		bitNum &= 31
		val := c.reg.D[reg]
		if val&(1<<bitNum) == 0 {
			c.reg.SR |= flagZ
		} else {
			c.reg.SR &^= flagZ
		}
		c.cycles += 6
	} else {
		bitNum &= 7
		dst := c.resolveEA(mode, reg, Byte)
		val := dst.read(c, Byte)
		if val&(1<<bitNum) == 0 {
			c.reg.SR |= flagZ
		} else {
			c.reg.SR &^= flagZ
		}
		c.cycles += 4
	}
}

func opBTSTstatic(c *CPU) {
	bitNum := uint32(c.fetchPC() & 0xFF)
	mode := uint8((c.ir >> 3) & 7)
	reg := uint8(c.ir & 7)

	if mode == 0 {
		bitNum &= 31
		val := c.reg.D[reg]
		if val&(1<<bitNum) == 0 {
			c.reg.SR |= flagZ
		} else {
			c.reg.SR &^= flagZ
		}
		c.cycles += 10
	} else {
		bitNum &= 7
		dst := c.resolveEA(mode, reg, Byte)
		val := dst.read(c, Byte)
		if val&(1<<bitNum) == 0 {
			c.reg.SR |= flagZ
		} else {
			c.reg.SR &^= flagZ
		}
		c.cycles += 8
	}
}

// --- BCHG ---

func registerBCHG() {
	for dn := uint16(0); dn < 8; dn++ {
		for mode := uint16(0); mode < 8; mode++ {
			if mode == 1 {
				continue
			}
			for reg := uint16(0); reg < 8; reg++ {
				if mode == 7 && reg > 1 {
					continue
				}
				opcode := 0x0140 | dn<<9 | mode<<3 | reg
				opcodeTable[opcode] = opBCHGdyn
			}
		}
	}
	for mode := uint16(0); mode < 8; mode++ {
		if mode == 1 {
			continue
		}
		for reg := uint16(0); reg < 8; reg++ {
			if mode == 7 && reg > 1 {
				continue
			}
			opcode := 0x0840 | mode<<3 | reg
			opcodeTable[opcode] = opBCHGstatic
		}
	}
}

func opBCHGdyn(c *CPU) {
	dn := (c.ir >> 9) & 7
	mode := uint8((c.ir >> 3) & 7)
	reg := uint8(c.ir & 7)
	bitNum := c.reg.D[dn]

	if mode == 0 {
		bitNum &= 31
		mask := uint32(1) << bitNum
		if c.reg.D[reg]&mask == 0 {
			c.reg.SR |= flagZ
		} else {
			c.reg.SR &^= flagZ
		}
		c.reg.D[reg] ^= mask
		c.cycles += 8
	} else {
		bitNum &= 7
		dst := c.resolveEA(mode, reg, Byte)
		val := dst.read(c, Byte)
		mask := uint32(1) << bitNum
		if val&mask == 0 {
			c.reg.SR |= flagZ
		} else {
			c.reg.SR &^= flagZ
		}
		dst.write(c, Byte, val^mask)
		c.cycles += 8
	}
}

func opBCHGstatic(c *CPU) {
	bitNum := uint32(c.fetchPC() & 0xFF)
	mode := uint8((c.ir >> 3) & 7)
	reg := uint8(c.ir & 7)

	if mode == 0 {
		bitNum &= 31
		mask := uint32(1) << bitNum
		if c.reg.D[reg]&mask == 0 {
			c.reg.SR |= flagZ
		} else {
			c.reg.SR &^= flagZ
		}
		c.reg.D[reg] ^= mask
		c.cycles += 12
	} else {
		bitNum &= 7
		dst := c.resolveEA(mode, reg, Byte)
		val := dst.read(c, Byte)
		mask := uint32(1) << bitNum
		if val&mask == 0 {
			c.reg.SR |= flagZ
		} else {
			c.reg.SR &^= flagZ
		}
		dst.write(c, Byte, val^mask)
		c.cycles += 12
	}
}

// --- BCLR ---

func registerBCLR() {
	for dn := uint16(0); dn < 8; dn++ {
		for mode := uint16(0); mode < 8; mode++ {
			if mode == 1 {
				continue
			}
			for reg := uint16(0); reg < 8; reg++ {
				if mode == 7 && reg > 1 {
					continue
				}
				opcode := 0x0180 | dn<<9 | mode<<3 | reg
				opcodeTable[opcode] = opBCLRdyn
			}
		}
	}
	for mode := uint16(0); mode < 8; mode++ {
		if mode == 1 {
			continue
		}
		for reg := uint16(0); reg < 8; reg++ {
			if mode == 7 && reg > 1 {
				continue
			}
			opcode := 0x0880 | mode<<3 | reg
			opcodeTable[opcode] = opBCLRstatic
		}
	}
}

func opBCLRdyn(c *CPU) {
	dn := (c.ir >> 9) & 7
	mode := uint8((c.ir >> 3) & 7)
	reg := uint8(c.ir & 7)
	bitNum := c.reg.D[dn]

	if mode == 0 {
		bitNum &= 31
		mask := uint32(1) << bitNum
		if c.reg.D[reg]&mask == 0 {
			c.reg.SR |= flagZ
		} else {
			c.reg.SR &^= flagZ
		}
		c.reg.D[reg] &^= mask
		c.cycles += 10
	} else {
		bitNum &= 7
		dst := c.resolveEA(mode, reg, Byte)
		val := dst.read(c, Byte)
		mask := uint32(1) << bitNum
		if val&mask == 0 {
			c.reg.SR |= flagZ
		} else {
			c.reg.SR &^= flagZ
		}
		dst.write(c, Byte, val&^mask)
		c.cycles += 8
	}
}

func opBCLRstatic(c *CPU) {
	bitNum := uint32(c.fetchPC() & 0xFF)
	mode := uint8((c.ir >> 3) & 7)
	reg := uint8(c.ir & 7)

	if mode == 0 {
		bitNum &= 31
		mask := uint32(1) << bitNum
		if c.reg.D[reg]&mask == 0 {
			c.reg.SR |= flagZ
		} else {
			c.reg.SR &^= flagZ
		}
		c.reg.D[reg] &^= mask
		c.cycles += 14
	} else {
		bitNum &= 7
		dst := c.resolveEA(mode, reg, Byte)
		val := dst.read(c, Byte)
		mask := uint32(1) << bitNum
		if val&mask == 0 {
			c.reg.SR |= flagZ
		} else {
			c.reg.SR &^= flagZ
		}
		dst.write(c, Byte, val&^mask)
		c.cycles += 12
	}
}

// --- BSET ---

func registerBSET() {
	for dn := uint16(0); dn < 8; dn++ {
		for mode := uint16(0); mode < 8; mode++ {
			if mode == 1 {
				continue
			}
			for reg := uint16(0); reg < 8; reg++ {
				if mode == 7 && reg > 1 {
					continue
				}
				opcode := 0x01C0 | dn<<9 | mode<<3 | reg
				opcodeTable[opcode] = opBSETdyn
			}
		}
	}
	for mode := uint16(0); mode < 8; mode++ {
		if mode == 1 {
			continue
		}
		for reg := uint16(0); reg < 8; reg++ {
			if mode == 7 && reg > 1 {
				continue
			}
			opcode := 0x08C0 | mode<<3 | reg
			opcodeTable[opcode] = opBSETstatic
		}
	}
}

func opBSETdyn(c *CPU) {
	dn := (c.ir >> 9) & 7
	mode := uint8((c.ir >> 3) & 7)
	reg := uint8(c.ir & 7)
	bitNum := c.reg.D[dn]

	if mode == 0 {
		bitNum &= 31
		mask := uint32(1) << bitNum
		if c.reg.D[reg]&mask == 0 {
			c.reg.SR |= flagZ
		} else {
			c.reg.SR &^= flagZ
		}
		c.reg.D[reg] |= mask
		c.cycles += 8
	} else {
		bitNum &= 7
		dst := c.resolveEA(mode, reg, Byte)
		val := dst.read(c, Byte)
		mask := uint32(1) << bitNum
		if val&mask == 0 {
			c.reg.SR |= flagZ
		} else {
			c.reg.SR &^= flagZ
		}
		dst.write(c, Byte, val|mask)
		c.cycles += 8
	}
}

func opBSETstatic(c *CPU) {
	bitNum := uint32(c.fetchPC() & 0xFF)
	mode := uint8((c.ir >> 3) & 7)
	reg := uint8(c.ir & 7)

	if mode == 0 {
		bitNum &= 31
		mask := uint32(1) << bitNum
		if c.reg.D[reg]&mask == 0 {
			c.reg.SR |= flagZ
		} else {
			c.reg.SR &^= flagZ
		}
		c.reg.D[reg] |= mask
		c.cycles += 12
	} else {
		bitNum &= 7
		dst := c.resolveEA(mode, reg, Byte)
		val := dst.read(c, Byte)
		mask := uint32(1) << bitNum
		if val&mask == 0 {
			c.reg.SR |= flagZ
		} else {
			c.reg.SR &^= flagZ
		}
		dst.write(c, Byte, val|mask)
		c.cycles += 12
	}
}
