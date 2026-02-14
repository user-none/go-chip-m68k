package m68k

func init() {
	registerADD()
	registerADDA()
	registerADDI()
	registerADDQ()
	registerADDX()
	registerSUB()
	registerSUBA()
	registerSUBI()
	registerSUBQ()
	registerSUBX()
	registerCMP()
	registerCMPA()
	registerCMPI()
	registerCMPM()
	registerMULU()
	registerMULS()
	registerDIVU()
	registerDIVS()
	registerNEG()
	registerNEGX()
	registerCLR()
	registerEXT()
	registerCHK()
}

// sizeEncoding maps the standard 2-bit size field (bits 7-6) to Size.
func sizeEncoding(bits uint16) Size {
	switch bits {
	case 0:
		return Byte
	case 1:
		return Word
	case 2:
		return Long
	}
	return 0
}

// --- ADD ---

// registerADD registers ADD <ea>,Dn and ADD Dn,<ea>.
// Encoding: 1101 DDD O SS eee eee
//
//	O=0: <ea>+Dn->Dn  O=1: Dn+<ea>-><ea>
func registerADD() {
	for dn := uint16(0); dn < 8; dn++ {
		for szBits := uint16(0); szBits < 3; szBits++ {
			// Direction 0: <ea>,Dn (all source EAs)
			for mode := uint16(0); mode < 8; mode++ {
				for reg := uint16(0); reg < 8; reg++ {
					if mode == 7 && reg > 4 {
						continue
					}
					// An direct only valid for Word/Long
					if mode == 1 && szBits == 0 {
						continue
					}
					opcode := 0xD000 | dn<<9 | szBits<<6 | mode<<3 | reg
					opcodeTable[opcode] = opADDtoReg
				}
			}
			// Direction 1: Dn,<ea> (memory alterable only)
			for mode := uint16(2); mode < 8; mode++ {
				for reg := uint16(0); reg < 8; reg++ {
					if mode == 7 && reg > 1 {
						continue
					}
					opcode := 0xD000 | dn<<9 | (szBits+4)<<6 | mode<<3 | reg
					opcodeTable[opcode] = opADDtoEA
				}
			}
		}
	}
}

func opADDtoReg(c *CPU) {
	dn := (c.ir >> 9) & 7
	sz := sizeEncoding((c.ir >> 6) & 3)
	mode := uint8((c.ir >> 3) & 7)
	reg := uint8(c.ir & 7)

	src := c.resolveEA(mode, reg, sz)
	s := src.read(c, sz)
	d := c.reg.D[dn] & sz.Mask()
	result := s + d
	c.setFlagsAdd(s, d, result, sz)

	mask := sz.Mask()
	c.reg.D[dn] = (c.reg.D[dn] & ^mask) | (result & mask)

	fetch := eaFetchCycles(mode, reg, sz)
	if sz != Long {
		c.cycles += 4 + fetch
	} else if mode >= 2 && !(mode == 7 && reg == 4) {
		c.cycles += 6 + fetch
	} else {
		c.cycles += 8 + fetch
	}
}

func opADDtoEA(c *CPU) {
	dn := (c.ir >> 9) & 7
	sz := sizeEncoding(((c.ir >> 6) & 7) - 4)
	mode := uint8((c.ir >> 3) & 7)
	reg := uint8(c.ir & 7)

	dst := c.resolveEA(mode, reg, sz)
	d := dst.read(c, sz)
	s := c.reg.D[dn] & sz.Mask()
	result := s + d
	c.setFlagsAdd(s, d, result, sz)
	dst.write(c, sz, result)

	fetch := eaFetchCycles(mode, reg, sz)
	if sz == Long {
		c.cycles += 12 + fetch
	} else {
		c.cycles += 8 + fetch
	}
}

// --- ADDA ---

func registerADDA() {
	for an := uint16(0); an < 8; an++ {
		for _, szBit := range []uint16{3, 7} { // 3=Word, 7=Long
			for mode := uint16(0); mode < 8; mode++ {
				for reg := uint16(0); reg < 8; reg++ {
					if mode == 7 && reg > 4 {
						continue
					}
					opcode := 0xD000 | an<<9 | szBit<<6 | mode<<3 | reg
					opcodeTable[opcode] = opADDA
				}
			}
		}
	}
}

func opADDA(c *CPU) {
	an := (c.ir >> 9) & 7
	sz := Word
	if (c.ir>>6)&7 == 7 {
		sz = Long
	}
	mode := uint8((c.ir >> 3) & 7)
	reg := uint8(c.ir & 7)

	src := c.resolveEA(mode, reg, sz)
	val := src.read(c, sz)
	if sz == Word {
		val = uint32(int32(int16(val)))
	}
	c.reg.A[an] += val

	// ADDA does not affect condition codes
	fetch := eaFetchCycles(mode, reg, sz)
	if sz == Long && mode >= 2 && !(mode == 7 && reg == 4) {
		c.cycles += 6 + fetch
	} else {
		c.cycles += 8 + fetch
	}
}

// --- ADDI ---

func registerADDI() {
	for szBits := uint16(0); szBits < 3; szBits++ {
		for mode := uint16(0); mode < 8; mode++ {
			if mode == 1 {
				continue
			}
			for reg := uint16(0); reg < 8; reg++ {
				if mode == 7 && reg > 1 {
					continue
				}
				opcode := 0x0600 | szBits<<6 | mode<<3 | reg
				opcodeTable[opcode] = opADDI
			}
		}
	}
}

func opADDI(c *CPU) {
	sz := sizeEncoding((c.ir >> 6) & 3)
	mode := uint8((c.ir >> 3) & 7)
	reg := uint8(c.ir & 7)

	var imm uint32
	if sz == Long {
		imm = c.fetchPCLong()
	} else {
		imm = uint32(c.fetchPC()) & sz.Mask()
	}

	dst := c.resolveEA(mode, reg, sz)
	d := dst.read(c, sz)
	result := imm + d
	c.setFlagsAdd(imm, d, result, sz)
	dst.write(c, sz, result)

	if mode == 0 {
		if sz == Long {
			c.cycles += 16
		} else {
			c.cycles += 8
		}
	} else {
		fetch := eaFetchCycles(mode, reg, sz)
		if sz == Long {
			c.cycles += 20 + fetch
		} else {
			c.cycles += 12 + fetch
		}
	}
}

// --- ADDQ ---

func registerADDQ() {
	for data := uint16(0); data < 8; data++ {
		for szBits := uint16(0); szBits < 3; szBits++ {
			for mode := uint16(0); mode < 8; mode++ {
				for reg := uint16(0); reg < 8; reg++ {
					if mode == 7 && reg > 1 {
						continue
					}
					// Byte size not valid for An
					if mode == 1 && szBits == 0 {
						continue
					}
					opcode := 0x5000 | data<<9 | szBits<<6 | mode<<3 | reg
					opcodeTable[opcode] = opADDQ
				}
			}
		}
	}
}

func opADDQ(c *CPU) {
	data := uint32((c.ir >> 9) & 7)
	if data == 0 {
		data = 8
	}
	sz := sizeEncoding((c.ir >> 6) & 3)
	mode := uint8((c.ir >> 3) & 7)
	reg := uint8(c.ir & 7)

	if mode == 1 {
		// ADDQ to An: always 32-bit, no flags
		c.reg.A[reg] += data
		c.cycles += 8
		return
	}

	dst := c.resolveEA(mode, reg, sz)
	d := dst.read(c, sz)
	result := data + d
	c.setFlagsAdd(data, d, result, sz)
	dst.write(c, sz, result)

	if mode == 0 {
		if sz == Long {
			c.cycles += 8
		} else {
			c.cycles += 4
		}
	} else {
		fetch := eaFetchCycles(mode, reg, sz)
		if sz == Long {
			c.cycles += 12 + fetch
		} else {
			c.cycles += 8 + fetch
		}
	}
}

// --- ADDX ---

func registerADDX() {
	for rx := uint16(0); rx < 8; rx++ {
		for ry := uint16(0); ry < 8; ry++ {
			for szBits := uint16(0); szBits < 3; szBits++ {
				// Dn,Dn
				opcodeTable[0xD100|rx<<9|szBits<<6|ry] = opADDXreg
				// -(Ax),-(Ay)
				opcodeTable[0xD108|rx<<9|szBits<<6|ry] = opADDXmem
			}
		}
	}
}

func opADDXreg(c *CPU) {
	rx := (c.ir >> 9) & 7
	sz := sizeEncoding((c.ir >> 6) & 3)
	ry := c.ir & 7

	s := c.reg.D[ry] & sz.Mask()
	d := c.reg.D[rx] & sz.Mask()
	x := uint32(0)
	if c.reg.SR&flagX != 0 {
		x = 1
	}
	result := d + s + x

	oldZ := c.reg.SR & flagZ
	c.setFlagsAdd(s, d, result, sz)
	// ADDX: Z flag only cleared, never set (preserves Z across multi-precision)
	if result&sz.Mask() == 0 {
		c.reg.SR = (c.reg.SR &^ flagZ) | oldZ
	}

	mask := sz.Mask()
	c.reg.D[rx] = (c.reg.D[rx] & ^mask) | (result & mask)

	c.cycles += 4
	if sz == Long {
		c.cycles += 4
	}
}

func opADDXmem(c *CPU) {
	rx := (c.ir >> 9) & 7
	sz := sizeEncoding((c.ir >> 6) & 3)
	ry := c.ir & 7

	src := c.resolveEA(4, uint8(ry), sz) // -(Ay)
	s := src.read(c, sz)
	dst := c.resolveEA(4, uint8(rx), sz) // -(Ax)
	d := dst.read(c, sz)
	x := uint32(0)
	if c.reg.SR&flagX != 0 {
		x = 1
	}
	result := d + s + x

	oldZ := c.reg.SR & flagZ
	c.setFlagsAdd(s, d, result, sz)
	if result&sz.Mask() == 0 {
		c.reg.SR = (c.reg.SR &^ flagZ) | oldZ
	}

	dst.write(c, sz, result)
	if sz == Long {
		c.cycles += 30
	} else {
		c.cycles += 18
	}
}

// --- SUB ---

func registerSUB() {
	for dn := uint16(0); dn < 8; dn++ {
		for szBits := uint16(0); szBits < 3; szBits++ {
			// <ea>,Dn
			for mode := uint16(0); mode < 8; mode++ {
				for reg := uint16(0); reg < 8; reg++ {
					if mode == 7 && reg > 4 {
						continue
					}
					if mode == 1 && szBits == 0 {
						continue
					}
					opcode := 0x9000 | dn<<9 | szBits<<6 | mode<<3 | reg
					opcodeTable[opcode] = opSUBtoReg
				}
			}
			// Dn,<ea>
			for mode := uint16(2); mode < 8; mode++ {
				for reg := uint16(0); reg < 8; reg++ {
					if mode == 7 && reg > 1 {
						continue
					}
					opcode := 0x9000 | dn<<9 | (szBits+4)<<6 | mode<<3 | reg
					opcodeTable[opcode] = opSUBtoEA
				}
			}
		}
	}
}

func opSUBtoReg(c *CPU) {
	dn := (c.ir >> 9) & 7
	sz := sizeEncoding((c.ir >> 6) & 3)
	mode := uint8((c.ir >> 3) & 7)
	reg := uint8(c.ir & 7)

	src := c.resolveEA(mode, reg, sz)
	s := src.read(c, sz)
	d := c.reg.D[dn] & sz.Mask()
	result := d - s
	c.setFlagsSub(s, d, result, sz)

	mask := sz.Mask()
	c.reg.D[dn] = (c.reg.D[dn] & ^mask) | (result & mask)

	fetch := eaFetchCycles(mode, reg, sz)
	if sz != Long {
		c.cycles += 4 + fetch
	} else if mode >= 2 && !(mode == 7 && reg == 4) {
		c.cycles += 6 + fetch
	} else {
		c.cycles += 8 + fetch
	}
}

func opSUBtoEA(c *CPU) {
	dn := (c.ir >> 9) & 7
	sz := sizeEncoding(((c.ir >> 6) & 7) - 4)
	mode := uint8((c.ir >> 3) & 7)
	reg := uint8(c.ir & 7)

	dst := c.resolveEA(mode, reg, sz)
	d := dst.read(c, sz)
	s := c.reg.D[dn] & sz.Mask()
	result := d - s
	c.setFlagsSub(s, d, result, sz)
	dst.write(c, sz, result)

	fetch := eaFetchCycles(mode, reg, sz)
	if sz == Long {
		c.cycles += 12 + fetch
	} else {
		c.cycles += 8 + fetch
	}
}

// --- SUBA ---

func registerSUBA() {
	for an := uint16(0); an < 8; an++ {
		for _, szBit := range []uint16{3, 7} {
			for mode := uint16(0); mode < 8; mode++ {
				for reg := uint16(0); reg < 8; reg++ {
					if mode == 7 && reg > 4 {
						continue
					}
					opcode := 0x9000 | an<<9 | szBit<<6 | mode<<3 | reg
					opcodeTable[opcode] = opSUBA
				}
			}
		}
	}
}

func opSUBA(c *CPU) {
	an := (c.ir >> 9) & 7
	sz := Word
	if (c.ir>>6)&7 == 7 {
		sz = Long
	}
	mode := uint8((c.ir >> 3) & 7)
	reg := uint8(c.ir & 7)

	src := c.resolveEA(mode, reg, sz)
	val := src.read(c, sz)
	if sz == Word {
		val = uint32(int32(int16(val)))
	}
	c.reg.A[an] -= val

	fetch := eaFetchCycles(mode, reg, sz)
	if sz == Long && mode >= 2 && !(mode == 7 && reg == 4) {
		c.cycles += 6 + fetch
	} else {
		c.cycles += 8 + fetch
	}
}

// --- SUBI ---

func registerSUBI() {
	for szBits := uint16(0); szBits < 3; szBits++ {
		for mode := uint16(0); mode < 8; mode++ {
			if mode == 1 {
				continue
			}
			for reg := uint16(0); reg < 8; reg++ {
				if mode == 7 && reg > 1 {
					continue
				}
				opcode := 0x0400 | szBits<<6 | mode<<3 | reg
				opcodeTable[opcode] = opSUBI
			}
		}
	}
}

func opSUBI(c *CPU) {
	sz := sizeEncoding((c.ir >> 6) & 3)
	mode := uint8((c.ir >> 3) & 7)
	reg := uint8(c.ir & 7)

	var imm uint32
	if sz == Long {
		imm = c.fetchPCLong()
	} else {
		imm = uint32(c.fetchPC()) & sz.Mask()
	}

	dst := c.resolveEA(mode, reg, sz)
	d := dst.read(c, sz)
	result := d - imm
	c.setFlagsSub(imm, d, result, sz)
	dst.write(c, sz, result)

	if mode == 0 {
		if sz == Long {
			c.cycles += 16
		} else {
			c.cycles += 8
		}
	} else {
		fetch := eaFetchCycles(mode, reg, sz)
		if sz == Long {
			c.cycles += 20 + fetch
		} else {
			c.cycles += 12 + fetch
		}
	}
}

// --- SUBQ ---

func registerSUBQ() {
	for data := uint16(0); data < 8; data++ {
		for szBits := uint16(0); szBits < 3; szBits++ {
			for mode := uint16(0); mode < 8; mode++ {
				for reg := uint16(0); reg < 8; reg++ {
					if mode == 7 && reg > 1 {
						continue
					}
					if mode == 1 && szBits == 0 {
						continue
					}
					opcode := 0x5100 | data<<9 | szBits<<6 | mode<<3 | reg
					opcodeTable[opcode] = opSUBQ
				}
			}
		}
	}
}

func opSUBQ(c *CPU) {
	data := uint32((c.ir >> 9) & 7)
	if data == 0 {
		data = 8
	}
	sz := sizeEncoding((c.ir >> 6) & 3)
	mode := uint8((c.ir >> 3) & 7)
	reg := uint8(c.ir & 7)

	if mode == 1 {
		c.reg.A[reg] -= data
		c.cycles += 8
		return
	}

	dst := c.resolveEA(mode, reg, sz)
	d := dst.read(c, sz)
	result := d - data
	c.setFlagsSub(data, d, result, sz)
	dst.write(c, sz, result)

	if mode == 0 {
		if sz == Long {
			c.cycles += 8
		} else {
			c.cycles += 4
		}
	} else {
		fetch := eaFetchCycles(mode, reg, sz)
		if sz == Long {
			c.cycles += 12 + fetch
		} else {
			c.cycles += 8 + fetch
		}
	}
}

// --- SUBX ---

func registerSUBX() {
	for rx := uint16(0); rx < 8; rx++ {
		for ry := uint16(0); ry < 8; ry++ {
			for szBits := uint16(0); szBits < 3; szBits++ {
				opcodeTable[0x9100|rx<<9|szBits<<6|ry] = opSUBXreg
				opcodeTable[0x9108|rx<<9|szBits<<6|ry] = opSUBXmem
			}
		}
	}
}

func opSUBXreg(c *CPU) {
	rx := (c.ir >> 9) & 7
	sz := sizeEncoding((c.ir >> 6) & 3)
	ry := c.ir & 7

	s := c.reg.D[ry] & sz.Mask()
	d := c.reg.D[rx] & sz.Mask()
	x := uint32(0)
	if c.reg.SR&flagX != 0 {
		x = 1
	}
	result := d - s - x

	oldZ := c.reg.SR & flagZ
	c.setFlagsSub(s, d, result, sz)
	// SUBX: Z flag only cleared, never set (preserves Z across multi-precision)
	if result&sz.Mask() == 0 {
		c.reg.SR = (c.reg.SR &^ flagZ) | oldZ
	}

	mask := sz.Mask()
	c.reg.D[rx] = (c.reg.D[rx] & ^mask) | (result & mask)

	c.cycles += 4
	if sz == Long {
		c.cycles += 4
	}
}

func opSUBXmem(c *CPU) {
	rx := (c.ir >> 9) & 7
	sz := sizeEncoding((c.ir >> 6) & 3)
	ry := c.ir & 7

	src := c.resolveEA(4, uint8(ry), sz)
	s := src.read(c, sz)
	dst := c.resolveEA(4, uint8(rx), sz)
	d := dst.read(c, sz)
	x := uint32(0)
	if c.reg.SR&flagX != 0 {
		x = 1
	}
	result := d - s - x

	oldZ := c.reg.SR & flagZ
	c.setFlagsSub(s, d, result, sz)
	if result&sz.Mask() == 0 {
		c.reg.SR = (c.reg.SR &^ flagZ) | oldZ
	}

	dst.write(c, sz, result)
	if sz == Long {
		c.cycles += 30
	} else {
		c.cycles += 18
	}
}

// --- CMP ---

func registerCMP() {
	for dn := uint16(0); dn < 8; dn++ {
		for szBits := uint16(0); szBits < 3; szBits++ {
			for mode := uint16(0); mode < 8; mode++ {
				for reg := uint16(0); reg < 8; reg++ {
					if mode == 7 && reg > 4 {
						continue
					}
					if mode == 1 && szBits == 0 {
						continue
					}
					opcode := 0xB000 | dn<<9 | szBits<<6 | mode<<3 | reg
					opcodeTable[opcode] = opCMP
				}
			}
		}
	}
}

func opCMP(c *CPU) {
	dn := (c.ir >> 9) & 7
	sz := sizeEncoding((c.ir >> 6) & 3)
	mode := uint8((c.ir >> 3) & 7)
	reg := uint8(c.ir & 7)

	src := c.resolveEA(mode, reg, sz)
	s := src.read(c, sz)
	d := c.reg.D[dn] & sz.Mask()
	result := d - s
	c.setFlagsCmp(s, d, result, sz)

	fetch := eaFetchCycles(mode, reg, sz)
	if sz == Long {
		c.cycles += 6 + fetch
	} else {
		c.cycles += 4 + fetch
	}
}

// --- CMPA ---

func registerCMPA() {
	for an := uint16(0); an < 8; an++ {
		for _, szBit := range []uint16{3, 7} {
			for mode := uint16(0); mode < 8; mode++ {
				for reg := uint16(0); reg < 8; reg++ {
					if mode == 7 && reg > 4 {
						continue
					}
					opcode := 0xB000 | an<<9 | szBit<<6 | mode<<3 | reg
					opcodeTable[opcode] = opCMPA
				}
			}
		}
	}
}

func opCMPA(c *CPU) {
	an := (c.ir >> 9) & 7
	sz := Word
	if (c.ir>>6)&7 == 7 {
		sz = Long
	}
	mode := uint8((c.ir >> 3) & 7)
	reg := uint8(c.ir & 7)

	src := c.resolveEA(mode, reg, sz)
	val := src.read(c, sz)
	if sz == Word {
		val = uint32(int32(int16(val)))
	}
	d := c.reg.A[an]
	result := d - val
	c.setFlagsCmp(val, d, result, Long)

	c.cycles += 6 + eaFetchCycles(mode, reg, sz)
}

// --- CMPI ---

func registerCMPI() {
	for szBits := uint16(0); szBits < 3; szBits++ {
		for mode := uint16(0); mode < 8; mode++ {
			if mode == 1 {
				continue
			}
			for reg := uint16(0); reg < 8; reg++ {
				if mode == 7 && reg > 1 {
					continue
				}
				opcode := 0x0C00 | szBits<<6 | mode<<3 | reg
				opcodeTable[opcode] = opCMPI
			}
		}
	}
}

func opCMPI(c *CPU) {
	sz := sizeEncoding((c.ir >> 6) & 3)
	mode := uint8((c.ir >> 3) & 7)
	reg := uint8(c.ir & 7)

	var imm uint32
	if sz == Long {
		imm = c.fetchPCLong()
	} else {
		imm = uint32(c.fetchPC()) & sz.Mask()
	}

	dst := c.resolveEA(mode, reg, sz)
	d := dst.read(c, sz)
	result := d - imm
	c.setFlagsCmp(imm, d, result, sz)

	if mode == 0 {
		if sz == Long {
			c.cycles += 14
		} else {
			c.cycles += 8
		}
	} else {
		fetch := eaFetchCycles(mode, reg, sz)
		if sz == Long {
			c.cycles += 12 + fetch
		} else {
			c.cycles += 8 + fetch
		}
	}
}

// --- CMPM ---

func registerCMPM() {
	for ax := uint16(0); ax < 8; ax++ {
		for ay := uint16(0); ay < 8; ay++ {
			for szBits := uint16(0); szBits < 3; szBits++ {
				opcode := 0xB108 | ax<<9 | szBits<<6 | ay
				opcodeTable[opcode] = opCMPM
			}
		}
	}
}

func opCMPM(c *CPU) {
	sz := sizeEncoding((c.ir >> 6) & 3)
	ay := c.ir & 7
	ax := (c.ir >> 9) & 7

	src := c.resolveEA(3, uint8(ay), sz) // (Ay)+
	s := src.read(c, sz)
	dst := c.resolveEA(3, uint8(ax), sz) // (Ax)+
	d := dst.read(c, sz)
	result := d - s
	c.setFlagsCmp(s, d, result, sz)

	if sz == Long {
		c.cycles += 20
	} else {
		c.cycles += 12
	}
}

// --- MULU ---

func registerMULU() {
	for dn := uint16(0); dn < 8; dn++ {
		for mode := uint16(0); mode < 8; mode++ {
			if mode == 1 {
				continue
			}
			for reg := uint16(0); reg < 8; reg++ {
				if mode == 7 && reg > 4 {
					continue
				}
				opcode := 0xC0C0 | dn<<9 | mode<<3 | reg
				opcodeTable[opcode] = opMULU
			}
		}
	}
}

func opMULU(c *CPU) {
	dn := (c.ir >> 9) & 7
	mode := uint8((c.ir >> 3) & 7)
	reg := uint8(c.ir & 7)

	src := c.resolveEA(mode, reg, Word)
	s := src.read(c, Word)
	d := c.reg.D[dn] & 0xFFFF
	result := s * d
	c.reg.D[dn] = result

	c.setFlagsLogical(result, Long)
	c.cycles += 70 + eaFetchCycles(mode, reg, Word) // base varies 38-70, using worst-case
}

// --- MULS ---

func registerMULS() {
	for dn := uint16(0); dn < 8; dn++ {
		for mode := uint16(0); mode < 8; mode++ {
			if mode == 1 {
				continue
			}
			for reg := uint16(0); reg < 8; reg++ {
				if mode == 7 && reg > 4 {
					continue
				}
				opcode := 0xC1C0 | dn<<9 | mode<<3 | reg
				opcodeTable[opcode] = opMULS
			}
		}
	}
}

func opMULS(c *CPU) {
	dn := (c.ir >> 9) & 7
	mode := uint8((c.ir >> 3) & 7)
	reg := uint8(c.ir & 7)

	src := c.resolveEA(mode, reg, Word)
	s := int32(int16(src.read(c, Word)))
	d := int32(int16(c.reg.D[dn] & 0xFFFF))
	result := uint32(s * d)
	c.reg.D[dn] = result

	c.setFlagsLogical(result, Long)
	c.cycles += 70 + eaFetchCycles(mode, reg, Word) // base varies 38-70, using worst-case
}

// --- DIVU ---

func registerDIVU() {
	for dn := uint16(0); dn < 8; dn++ {
		for mode := uint16(0); mode < 8; mode++ {
			if mode == 1 {
				continue
			}
			for reg := uint16(0); reg < 8; reg++ {
				if mode == 7 && reg > 4 {
					continue
				}
				opcode := 0x80C0 | dn<<9 | mode<<3 | reg
				opcodeTable[opcode] = opDIVU
			}
		}
	}
}

func opDIVU(c *CPU) {
	dn := (c.ir >> 9) & 7
	mode := uint8((c.ir >> 3) & 7)
	reg := uint8(c.ir & 7)

	src := c.resolveEA(mode, reg, Word)
	divisor := src.read(c, Word)

	if divisor == 0 {
		c.exception(vecDivideByZero)
		return
	}

	dividend := c.reg.D[dn]
	quotient := dividend / divisor
	remainder := dividend % divisor

	if quotient > 0xFFFF {
		// Overflow
		c.reg.SR |= flagV
		c.reg.SR &^= flagC
	} else {
		c.reg.D[dn] = (remainder&0xFFFF)<<16 | (quotient & 0xFFFF)
		c.setFlagsLogical(quotient, Word)
	}

	c.cycles += 140 + eaFetchCycles(mode, reg, Word) // base varies 76-140, using worst-case
}

// --- DIVS ---

func registerDIVS() {
	for dn := uint16(0); dn < 8; dn++ {
		for mode := uint16(0); mode < 8; mode++ {
			if mode == 1 {
				continue
			}
			for reg := uint16(0); reg < 8; reg++ {
				if mode == 7 && reg > 4 {
					continue
				}
				opcode := 0x81C0 | dn<<9 | mode<<3 | reg
				opcodeTable[opcode] = opDIVS
			}
		}
	}
}

func opDIVS(c *CPU) {
	dn := (c.ir >> 9) & 7
	mode := uint8((c.ir >> 3) & 7)
	reg := uint8(c.ir & 7)

	src := c.resolveEA(mode, reg, Word)
	divisor := int32(int16(src.read(c, Word)))

	if divisor == 0 {
		c.exception(vecDivideByZero)
		return
	}

	dividend := int32(c.reg.D[dn])
	quotient := dividend / divisor
	remainder := dividend % divisor

	if quotient > 32767 || quotient < -32768 {
		c.reg.SR |= flagV | flagN
		c.reg.SR &^= flagC | flagZ
	} else {
		c.reg.D[dn] = uint32(remainder&0xFFFF)<<16 | uint32(quotient)&0xFFFF
		c.setFlagsLogical(uint32(quotient), Word)
	}

	c.cycles += 158 + eaFetchCycles(mode, reg, Word) // base varies 120-158, using worst-case
}

// --- NEG ---

func registerNEG() {
	for szBits := uint16(0); szBits < 3; szBits++ {
		for mode := uint16(0); mode < 8; mode++ {
			if mode == 1 {
				continue
			}
			for reg := uint16(0); reg < 8; reg++ {
				if mode == 7 && reg > 1 {
					continue
				}
				opcode := 0x4400 | szBits<<6 | mode<<3 | reg
				opcodeTable[opcode] = opNEG
			}
		}
	}
}

func opNEG(c *CPU) {
	sz := sizeEncoding((c.ir >> 6) & 3)
	mode := uint8((c.ir >> 3) & 7)
	reg := uint8(c.ir & 7)

	dst := c.resolveEA(mode, reg, sz)
	d := dst.read(c, sz)
	result := uint32(0) - d
	c.setFlagsSub(d, 0, result, sz)
	dst.write(c, sz, result)

	if mode == 0 {
		if sz == Long {
			c.cycles += 6
		} else {
			c.cycles += 4
		}
	} else {
		fetch := eaFetchCycles(mode, reg, sz)
		if sz == Long {
			c.cycles += 12 + fetch
		} else {
			c.cycles += 8 + fetch
		}
	}
}

// --- NEGX ---

func registerNEGX() {
	for szBits := uint16(0); szBits < 3; szBits++ {
		for mode := uint16(0); mode < 8; mode++ {
			if mode == 1 {
				continue
			}
			for reg := uint16(0); reg < 8; reg++ {
				if mode == 7 && reg > 1 {
					continue
				}
				opcode := 0x4000 | szBits<<6 | mode<<3 | reg
				opcodeTable[opcode] = opNEGX
			}
		}
	}
}

func opNEGX(c *CPU) {
	sz := sizeEncoding((c.ir >> 6) & 3)
	mode := uint8((c.ir >> 3) & 7)
	reg := uint8(c.ir & 7)

	dst := c.resolveEA(mode, reg, sz)
	d := dst.read(c, sz)
	x := uint32(0)
	if c.reg.SR&flagX != 0 {
		x = 1
	}
	result := uint32(0) - d - x
	oldZ := c.reg.SR & flagZ
	c.setFlagsSub(d, 0, result, sz)
	// NEGX: Z flag only cleared, never set (preserves Z across multi-precision)
	if result&sz.Mask() == 0 {
		c.reg.SR = (c.reg.SR &^ flagZ) | oldZ
	}
	dst.write(c, sz, result)

	if mode == 0 {
		if sz == Long {
			c.cycles += 6
		} else {
			c.cycles += 4
		}
	} else {
		fetch := eaFetchCycles(mode, reg, sz)
		if sz == Long {
			c.cycles += 12 + fetch
		} else {
			c.cycles += 8 + fetch
		}
	}
}

// --- CLR ---

func registerCLR() {
	for szBits := uint16(0); szBits < 3; szBits++ {
		for mode := uint16(0); mode < 8; mode++ {
			if mode == 1 {
				continue
			}
			for reg := uint16(0); reg < 8; reg++ {
				if mode == 7 && reg > 1 {
					continue
				}
				opcode := 0x4200 | szBits<<6 | mode<<3 | reg
				opcodeTable[opcode] = opCLR
			}
		}
	}
}

func opCLR(c *CPU) {
	sz := sizeEncoding((c.ir >> 6) & 3)
	mode := uint8((c.ir >> 3) & 7)
	reg := uint8(c.ir & 7)

	dst := c.resolveEA(mode, reg, sz)
	dst.write(c, sz, 0)

	// CLR always sets Z, clears NVC
	c.reg.SR &^= flagN | flagV | flagC
	c.reg.SR |= flagZ

	if mode == 0 {
		if sz == Long {
			c.cycles += 6
		} else {
			c.cycles += 4
		}
	} else {
		fetch := eaFetchCycles(mode, reg, sz)
		if sz == Long {
			c.cycles += 12 + fetch
		} else {
			c.cycles += 8 + fetch
		}
	}
}

// --- EXT ---

func registerEXT() {
	for dn := uint16(0); dn < 8; dn++ {
		// EXT.W (byte->word): opmode 010
		opcodeTable[0x4880|dn] = opEXTW
		// EXT.L (word->long): opmode 011
		opcodeTable[0x48C0|dn] = opEXTL
	}
}

func opEXTW(c *CPU) {
	dn := c.ir & 7
	val := uint32(int16(int8(c.reg.D[dn])))
	c.reg.D[dn] = (c.reg.D[dn] & 0xFFFF0000) | (val & 0xFFFF)
	c.setFlagsLogical(val, Word)
	c.cycles += 4
}

func opEXTL(c *CPU) {
	dn := c.ir & 7
	val := uint32(int32(int16(c.reg.D[dn])))
	c.reg.D[dn] = val
	c.setFlagsLogical(val, Long)
	c.cycles += 4
}

// --- CHK ---

// registerCHK registers CHK <ea>,Dn (word only on 68000).
// Encoding: 0100 DDD 110 MMM RRR
func registerCHK() {
	for dn := uint16(0); dn < 8; dn++ {
		for mode := uint16(0); mode < 8; mode++ {
			if mode == 1 {
				continue
			}
			for reg := uint16(0); reg < 8; reg++ {
				if mode == 7 && reg > 4 {
					continue
				}
				opcode := 0x4180 | dn<<9 | mode<<3 | reg
				opcodeTable[opcode] = opCHK
			}
		}
	}
}

func opCHK(c *CPU) {
	dn := (c.ir >> 9) & 7
	mode := uint8((c.ir >> 3) & 7)
	reg := uint8(c.ir & 7)

	src := c.resolveEA(mode, reg, Word)
	bound := int16(src.read(c, Word))
	val := int16(c.reg.D[dn] & 0xFFFF)

	if val < 0 {
		c.reg.SR &^= flagN | flagZ | flagV | flagC
		c.reg.SR |= flagN
		c.exception(vecCHK)
		return
	}
	if val > bound {
		c.reg.SR &^= flagN | flagZ | flagV | flagC
		c.exception(vecCHK)
		return
	}

	c.cycles += 10 + eaFetchCycles(mode, reg, Word)
}
