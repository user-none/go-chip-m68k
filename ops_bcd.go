package m68k

func init() {
	registerABCD()
	registerSBCD()
	registerNBCD()
}

// --- ABCD ---

func registerABCD() {
	// Encoding: 1100 XXX1 0000 RYYY  R=0: Dy,Dx  R=1: -(Ay),-(Ax)
	for rx := uint16(0); rx < 8; rx++ {
		for ry := uint16(0); ry < 8; ry++ {
			opcodeTable[0xC100|rx<<9|ry] = opABCDreg
			opcodeTable[0xC108|rx<<9|ry] = opABCDmem
		}
	}
}

func opABCDreg(c *CPU) {
	rx := (c.ir >> 9) & 7
	ry := c.ir & 7

	s := c.reg.D[ry] & 0xFF
	d := c.reg.D[rx] & 0xFF
	result := bcdAdd(c, s, d)
	c.reg.D[rx] = (c.reg.D[rx] & 0xFFFFFF00) | (result & 0xFF)

	c.cycles += 6
}

func opABCDmem(c *CPU) {
	rx := (c.ir >> 9) & 7
	ry := c.ir & 7

	src := c.resolveEA(4, uint8(ry), Byte) // -(Ay)
	s := src.read(c, Byte)
	dst := c.resolveEA(4, uint8(rx), Byte) // -(Ax)
	d := dst.read(c, Byte)
	result := bcdAdd(c, s, d)
	dst.write(c, Byte, result)

	c.cycles += 18
}

func bcdAdd(c *CPU, s, d uint32) uint32 {
	x := uint32(0)
	if c.reg.SR&flagX != 0 {
		x = 1
	}

	binary := s + d + x

	lo := (s & 0x0F) + (d & 0x0F) + x
	hi := (s & 0xF0) + (d & 0xF0)

	if lo > 9 {
		lo += 6
	}
	result := hi + lo

	carry := false
	if result > 0x99 {
		result += 0x60
		carry = true
	}

	r8 := result & 0xFF
	c.reg.SR &^= flagC | flagX | flagN | flagV
	if carry {
		c.reg.SR |= flagC | flagX
	}
	if r8&0x80 != 0 {
		c.reg.SR |= flagN
	}
	// V: bit 7 went from 0 to 1 during BCD correction
	if binary&0x80 == 0 && r8&0x80 != 0 {
		c.reg.SR |= flagV
	}
	if r8 != 0 {
		c.reg.SR &^= flagZ
	}

	return r8
}

// --- SBCD ---

func registerSBCD() {
	for rx := uint16(0); rx < 8; rx++ {
		for ry := uint16(0); ry < 8; ry++ {
			opcodeTable[0x8100|rx<<9|ry] = opSBCDreg
			opcodeTable[0x8108|rx<<9|ry] = opSBCDmem
		}
	}
}

func opSBCDreg(c *CPU) {
	rx := (c.ir >> 9) & 7
	ry := c.ir & 7

	s := c.reg.D[ry] & 0xFF
	d := c.reg.D[rx] & 0xFF
	result := bcdSub(c, s, d)
	c.reg.D[rx] = (c.reg.D[rx] & 0xFFFFFF00) | (result & 0xFF)

	c.cycles += 6
}

func opSBCDmem(c *CPU) {
	rx := (c.ir >> 9) & 7
	ry := c.ir & 7

	src := c.resolveEA(4, uint8(ry), Byte)
	s := src.read(c, Byte)
	dst := c.resolveEA(4, uint8(rx), Byte)
	d := dst.read(c, Byte)
	result := bcdSub(c, s, d)
	dst.write(c, Byte, result)

	c.cycles += 18
}

func bcdSub(c *CPU, s, d uint32) uint32 {
	x := uint32(0)
	if c.reg.SR&flagX != 0 {
		x = 1
	}

	binary := d - s - x

	lo := (d & 0x0F) - (s & 0x0F) - x
	result := binary
	if lo&0x10 != 0 {
		result -= 6
	}

	borrow := d < s+x
	if borrow {
		result -= 0x60
	}

	r8 := result & 0xFF

	c.reg.SR &^= flagC | flagX | flagN | flagV
	if borrow {
		c.reg.SR |= flagC | flagX
	}
	if r8&0x80 != 0 {
		c.reg.SR |= flagN
	}
	// V: bit 7 went from 1 to 0 during BCD correction (sign change)
	if binary&0x80 != 0 && r8&0x80 == 0 {
		c.reg.SR |= flagV
	}
	if r8 != 0 {
		c.reg.SR &^= flagZ
	}

	return r8
}

// --- NBCD ---

func registerNBCD() {
	// Encoding: 0100 1000 00ss ssss
	for mode := uint16(0); mode < 8; mode++ {
		if mode == 1 {
			continue
		}
		for reg := uint16(0); reg < 8; reg++ {
			if mode == 7 && reg > 1 {
				continue
			}
			opcodeTable[0x4800|mode<<3|reg] = opNBCD
		}
	}
}

func opNBCD(c *CPU) {
	mode := uint8((c.ir >> 3) & 7)
	reg := uint8(c.ir & 7)

	dst := c.resolveEA(mode, reg, Byte)
	d := dst.read(c, Byte)
	result := bcdSub(c, d, 0)
	dst.write(c, Byte, result)

	if mode == 0 {
		c.cycles += 6
	} else {
		c.cycles += 8 + eaFetchCycles(mode, reg, Byte)
	}
}
