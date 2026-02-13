package m68k

import "testing"

// testBus is a flat 16MB byte-array bus for testing.
// Supports Read/Write at any address in the 24-bit space.
type testBus struct {
	mem [16 * 1024 * 1024]byte
}

func (b *testBus) Read(sz Size, addr uint32) uint32 {
	addr &= 0xFFFFFF
	switch sz {
	case Byte:
		return uint32(b.mem[addr])
	case Word:
		return uint32(b.mem[addr])<<8 | uint32(b.mem[addr+1])
	case Long:
		return uint32(b.mem[addr])<<24 | uint32(b.mem[addr+1])<<16 |
			uint32(b.mem[addr+2])<<8 | uint32(b.mem[addr+3])
	}
	return 0
}

func (b *testBus) Write(sz Size, addr uint32, val uint32) {
	addr &= 0xFFFFFF
	switch sz {
	case Byte:
		b.mem[addr] = byte(val)
	case Word:
		b.mem[addr] = byte(val >> 8)
		b.mem[addr+1] = byte(val)
	case Long:
		b.mem[addr] = byte(val >> 24)
		b.mem[addr+1] = byte(val >> 16)
		b.mem[addr+2] = byte(val >> 8)
		b.mem[addr+3] = byte(val)
	}
}

func (b *testBus) Reset() {}

// cpuState captures the full programmer-visible state for a test case.
// RAM entries are [address, byte_value] pairs.
// A[7] is unused; the active stack pointer is derived from USP/SSP/SR.
type cpuState struct {
	D      [8]uint32
	A      [7]uint32
	PC     uint32
	SR     uint16
	USP    uint32
	SSP    uint32
	RAM    [][2]uint32
	Halted bool
}

// prefetchOffset is the 68000 prefetch pipeline offset.
// The SingleStepTests JSON data models the 68000's 2-word prefetch queue,
// where the PC register is 4 bytes ahead of the instruction being executed.
// Our emulator does not model the prefetch pipeline, so we adjust PC by -4
// when loading initial state and comparing final state.
const prefetchOffset uint32 = 4

// runTest loads initial state, executes one Step, and compares against expected state.
// PC values from the test data are adjusted by -prefetchOffset to account for the
// 68000's prefetch pipeline (instruction is at PC-4 in the hardware model).
func runTest(t *testing.T, init, want cpuState) {
	t.Helper()

	bus := &testBus{}

	// Load initial RAM (byte-level entries)
	for _, entry := range init.RAM {
		bus.mem[entry[0]&0xFFFFFF] = byte(entry[1])
	}

	// Bridge [7]uint32 to [8]uint32 for SetState (A7 is set from USP/SSP)
	var a8 [8]uint32
	copy(a8[:7], init.A[:])
	cpu := &CPU{bus: bus}
	cpu.SetState(init.D, a8, init.PC-prefetchOffset, init.SR, init.USP, init.SSP)

	cpu.Step()

	if want.Halted {
		if !cpu.Halted() {
			t.Errorf("expected CPU to be halted, but it is not")
		}
		return // Register/memory state is undefined after halt
	}
	if cpu.Halted() {
		t.Errorf("CPU unexpectedly halted")
		return
	}

	reg := cpu.Registers()

	// Compare data registers
	for i := 0; i < 8; i++ {
		if reg.D[i] != want.D[i] {
			t.Errorf("D%d = 0x%08X, want 0x%08X", i, reg.D[i], want.D[i])
		}
	}

	// Compare address registers (A0-A6)
	for i := 0; i < 7; i++ {
		if reg.A[i] != want.A[i] {
			t.Errorf("A%d = 0x%08X, want 0x%08X", i, reg.A[i], want.A[i])
		}
	}

	// Compare stack pointers and A7.
	// In supervisor mode, A[7] is the live SSP and reg.USP is the shadow USP.
	// In user mode, A[7] is the live USP and reg.SSP is the shadow SSP.
	// The JSON always provides the "real" USP/SSP values regardless of mode.
	if want.SR&0x2000 != 0 {
		// Supervisor mode: A7 = SSP, USP is shadow
		if reg.A[7] != want.SSP {
			t.Errorf("A7/SSP = 0x%08X, want 0x%08X", reg.A[7], want.SSP)
		}
		if reg.USP != want.USP {
			t.Errorf("USP = 0x%08X, want 0x%08X", reg.USP, want.USP)
		}
	} else {
		// User mode: A7 = USP, SSP is shadow
		if reg.A[7] != want.USP {
			t.Errorf("A7/USP = 0x%08X, want 0x%08X", reg.A[7], want.USP)
		}
		if reg.SSP != want.SSP {
			t.Errorf("SSP = 0x%08X, want 0x%08X", reg.SSP, want.SSP)
		}
	}

	// Compare PC (adjusted for prefetch offset)
	wantPC := want.PC - prefetchOffset
	if reg.PC != wantPC {
		t.Errorf("PC = 0x%08X, want 0x%08X", reg.PC, wantPC)
	}

	// Compare SR
	if reg.SR != want.SR {
		t.Errorf("SR = 0x%04X, want 0x%04X (diff: %04X)", reg.SR, want.SR, reg.SR^want.SR)
	}

	// Compare RAM
	for _, entry := range want.RAM {
		addr := entry[0] & 0xFFFFFF
		wantVal := byte(entry[1])
		gotVal := bus.mem[addr]
		if gotVal != wantVal {
			t.Errorf("RAM[0x%06X] = 0x%02X, want 0x%02X", addr, gotVal, wantVal)
		}
	}
}

// writeWord stores a big-endian 16-bit word into the test bus memory.
func writeWord(bus *testBus, addr uint32, val uint16) {
	bus.mem[addr] = byte(val >> 8)
	bus.mem[addr+1] = byte(val)
}

func TestAddressError(t *testing.T) {
	t.Run("word read from odd address halts", func(t *testing.T) {
		bus := &testBus{}
		cpu := &CPU{bus: bus}

		// MOVE.W (A0), D0 — opcode 0x3010
		pc := uint32(0x1000)
		writeWord(bus, pc, 0x3010)

		var a [8]uint32
		a[0] = 0x2001 // A0 = odd address
		cpu.SetState([8]uint32{}, a, pc, 0x2700, 0, 0x10000)
		cpu.Step()

		if !cpu.Halted() {
			t.Errorf("expected CPU to be halted after word read from odd address")
		}
	})

	t.Run("long read from odd address halts", func(t *testing.T) {
		bus := &testBus{}
		cpu := &CPU{bus: bus}

		// MOVE.L (A0), D0 — opcode 0x2010
		pc := uint32(0x1000)
		writeWord(bus, pc, 0x2010)

		var a [8]uint32
		a[0] = 0x2001 // A0 = odd address
		cpu.SetState([8]uint32{}, a, pc, 0x2700, 0, 0x10000)
		cpu.Step()

		if !cpu.Halted() {
			t.Errorf("expected CPU to be halted after long read from odd address")
		}
	})

	t.Run("word write to odd address halts", func(t *testing.T) {
		bus := &testBus{}
		cpu := &CPU{bus: bus}

		// MOVE.W D0, (A0) — opcode 0x3080
		pc := uint32(0x1000)
		writeWord(bus, pc, 0x3080)

		var a [8]uint32
		a[0] = 0x2001 // A0 = odd address
		cpu.SetState([8]uint32{0x1234}, a, pc, 0x2700, 0, 0x10000)
		cpu.Step()

		if !cpu.Halted() {
			t.Errorf("expected CPU to be halted after word write to odd address")
		}
	})

	t.Run("long write to odd address halts", func(t *testing.T) {
		bus := &testBus{}
		cpu := &CPU{bus: bus}

		// MOVE.L D0, (A0) — opcode 0x2080
		pc := uint32(0x1000)
		writeWord(bus, pc, 0x2080)

		var a [8]uint32
		a[0] = 0x2001 // A0 = odd address
		cpu.SetState([8]uint32{0x12345678}, a, pc, 0x2700, 0, 0x10000)
		cpu.Step()

		if !cpu.Halted() {
			t.Errorf("expected CPU to be halted after long write to odd address")
		}
	})

	t.Run("byte read from odd address works", func(t *testing.T) {
		bus := &testBus{}
		cpu := &CPU{bus: bus}

		// MOVE.B (A0), D0 — opcode 0x1010
		pc := uint32(0x1000)
		writeWord(bus, pc, 0x1010)

		var a [8]uint32
		a[0] = 0x2001 // A0 = odd address
		bus.mem[0x2001] = 0xAB
		cpu.SetState([8]uint32{}, a, pc, 0x2700, 0, 0x10000)
		cpu.Step()

		if cpu.Halted() {
			t.Errorf("CPU should not halt on byte read from odd address")
		}
		reg := cpu.Registers()
		if reg.D[0]&0xFF != 0xAB {
			t.Errorf("D0 low byte = 0x%02X, want 0xAB", reg.D[0]&0xFF)
		}
	})

	t.Run("byte write to odd address works", func(t *testing.T) {
		bus := &testBus{}
		cpu := &CPU{bus: bus}

		// MOVE.B D0, (A0) — opcode 0x1080
		pc := uint32(0x1000)
		writeWord(bus, pc, 0x1080)

		var a [8]uint32
		a[0] = 0x2001 // A0 = odd address
		cpu.SetState([8]uint32{0xCD}, a, pc, 0x2700, 0, 0x10000)
		cpu.Step()

		if cpu.Halted() {
			t.Errorf("CPU should not halt on byte write to odd address")
		}
		if bus.mem[0x2001] != 0xCD {
			t.Errorf("RAM[0x2001] = 0x%02X, want 0xCD", bus.mem[0x2001])
		}
	})

	t.Run("odd PC halts", func(t *testing.T) {
		bus := &testBus{}
		cpu := &CPU{bus: bus}

		// Put a NOP at address 0x1000 in case fetch reaches there
		writeWord(bus, 0x1000, 0x4E71)

		// Set PC to an odd address
		cpu.SetState([8]uint32{}, [8]uint32{}, 0x1001, 0x2700, 0, 0x10000)
		cycles := cpu.Step()

		if !cpu.Halted() {
			t.Errorf("expected CPU to be halted with odd PC")
		}
		if cycles != 0 {
			t.Errorf("Step() returned %d cycles, want 0 for halted CPU", cycles)
		}
	})

	t.Run("odd SSP during exception halts", func(t *testing.T) {
		bus := &testBus{}
		cpu := &CPU{bus: bus}

		// Use an unimplemented opcode to trigger illegal instruction exception.
		// Opcode 0x4AFC is the explicit ILLEGAL instruction on 68000.
		// The illegal instruction vector (4) address = 4*4 = 16.
		// Put a handler address at vector 4 (address 0x10).
		bus.mem[0x10] = 0x00
		bus.mem[0x11] = 0x00
		bus.mem[0x12] = 0x20
		bus.mem[0x13] = 0x00 // handler at 0x2000

		pc := uint32(0x1000)
		writeWord(bus, pc, 0x4AFC)

		// SSP is odd — the exception push (pushLong/pushWord) will try
		// to write to an odd address, triggering the alignment check.
		cpu.SetState([8]uint32{}, [8]uint32{}, pc, 0x2700, 0, 0x10001)
		cpu.Step()

		if !cpu.Halted() {
			t.Errorf("expected CPU to be halted when exception pushes to odd SSP")
		}
	})
}

// fillNOPs writes NOP instructions (0x4E71, 4 cycles each) starting at addr.
func fillNOPs(bus *testBus, addr uint32, count int) {
	for i := 0; i < count; i++ {
		writeWord(bus, addr+uint32(i*2), 0x4E71)
	}
}

// newNOPCPU creates a CPU with NOPs at the given PC and returns it ready to run.
func newNOPCPU(nopCount int) (*CPU, *testBus) {
	bus := &testBus{}
	pc := uint32(0x1000)
	fillNOPs(bus, pc, nopCount)
	cpu := &CPU{bus: bus}
	cpu.SetState([8]uint32{}, [8]uint32{}, pc, 0x2700, 0, 0x10000)
	return cpu, bus
}

func TestStepCycles(t *testing.T) {
	t.Run("budget larger than cost", func(t *testing.T) {
		cpu, _ := newNOPCPU(1)

		cycles := cpu.StepCycles(100)
		if cycles != 4 {
			t.Errorf("StepCycles(100) = %d, want 4", cycles)
		}
		if cpu.Deficit() != 0 {
			t.Errorf("Deficit() = %d, want 0", cpu.Deficit())
		}
	})

	t.Run("budget equal to cost", func(t *testing.T) {
		cpu, _ := newNOPCPU(1)

		cycles := cpu.StepCycles(4)
		if cycles != 4 {
			t.Errorf("StepCycles(4) = %d, want 4", cycles)
		}
		if cpu.Deficit() != 0 {
			t.Errorf("Deficit() = %d, want 0", cpu.Deficit())
		}
	})

	t.Run("budget smaller than cost creates deficit", func(t *testing.T) {
		cpu, _ := newNOPCPU(1)

		cycles := cpu.StepCycles(1)
		if cycles != 1 {
			t.Errorf("StepCycles(1) = %d, want 1", cycles)
		}
		if cpu.Deficit() != 3 {
			t.Errorf("Deficit() = %d, want 3", cpu.Deficit())
		}
	})

	t.Run("deficit paid off in one call", func(t *testing.T) {
		cpu, _ := newNOPCPU(2)

		// First call: NOP costs 4, budget is 1 → deficit = 3
		cpu.StepCycles(1)

		// Second call: pay off deficit of 3 with budget of 100
		cycles := cpu.StepCycles(100)
		if cycles != 3 {
			t.Errorf("StepCycles(100) = %d, want 3", cycles)
		}
		if cpu.Deficit() != 0 {
			t.Errorf("Deficit() = %d, want 0", cpu.Deficit())
		}
	})

	t.Run("deficit paid off across multiple calls", func(t *testing.T) {
		cpu, _ := newNOPCPU(2)

		// NOP costs 4, budget is 1 → deficit = 3
		cpu.StepCycles(1)

		// Pay 1 of 3 → deficit = 2
		cycles := cpu.StepCycles(1)
		if cycles != 1 {
			t.Errorf("StepCycles(1) = %d, want 1", cycles)
		}
		if cpu.Deficit() != 2 {
			t.Errorf("Deficit() = %d, want 2", cpu.Deficit())
		}

		// Pay 1 of 2 → deficit = 1
		cycles = cpu.StepCycles(1)
		if cycles != 1 {
			t.Errorf("StepCycles(1) = %d, want 1", cycles)
		}
		if cpu.Deficit() != 1 {
			t.Errorf("Deficit() = %d, want 1", cpu.Deficit())
		}

		// Pay 1 of 1 → deficit = 0
		cycles = cpu.StepCycles(1)
		if cycles != 1 {
			t.Errorf("StepCycles(1) = %d, want 1", cycles)
		}
		if cpu.Deficit() != 0 {
			t.Errorf("Deficit() = %d, want 0", cpu.Deficit())
		}
	})

	t.Run("multiple instructions within budget", func(t *testing.T) {
		cpu, _ := newNOPCPU(10)

		// Run 3 NOPs using StepCycles in a budget loop
		budget := 12
		count := 0
		for budget > 0 {
			cycles := cpu.StepCycles(budget)
			budget -= cycles
			count++
		}
		if count != 3 {
			t.Errorf("executed %d steps, want 3", count)
		}
		if budget != 0 {
			t.Errorf("remaining budget = %d, want 0", budget)
		}
	})

	t.Run("scanline boundary simulation", func(t *testing.T) {
		cpu, _ := newNOPCPU(20)

		// Scanline 1: budget of 10 cycles. NOPs cost 4 each.
		// Should fit 2 NOPs (8 cycles), third NOP overflows (4 > 2 remaining).
		budget := 10
		total := 0
		for budget > 0 {
			cycles := cpu.StepCycles(budget)
			budget -= cycles
			total += cycles
		}
		if total != 10 {
			t.Errorf("scanline 1 total = %d, want 10", total)
		}
		deficit := cpu.Deficit()
		if deficit != 2 {
			t.Errorf("deficit after scanline 1 = %d, want 2", deficit)
		}

		// Scanline 2: budget of 10. First call pays off deficit of 2.
		budget = 10
		total = 0
		first := cpu.StepCycles(budget)
		budget -= first
		total += first
		if first != 2 {
			t.Errorf("first call of scanline 2 = %d, want 2 (deficit payoff)", first)
		}

		// Continue running the rest of the budget
		for budget > 0 {
			cycles := cpu.StepCycles(budget)
			budget -= cycles
			total += cycles
		}
		if total != 10 {
			t.Errorf("scanline 2 total = %d, want 10", total)
		}
	})

	t.Run("halted CPU returns zero", func(t *testing.T) {
		cpu, _ := newNOPCPU(1)

		// Set PC to odd address to trigger halt
		cpu.SetState([8]uint32{}, [8]uint32{}, 0x1001, 0x2700, 0, 0x10000)
		cpu.Step()

		cycles := cpu.StepCycles(100)
		if cycles != 0 {
			t.Errorf("StepCycles(100) on halted CPU = %d, want 0", cycles)
		}
	})

	t.Run("reset clears deficit", func(t *testing.T) {
		cpu, bus := newNOPCPU(1)

		// Create a deficit
		cpu.StepCycles(1)
		if cpu.Deficit() == 0 {
			t.Fatal("expected non-zero deficit before reset")
		}

		// Set up reset vectors so Reset() works
		bus.Write(Long, 0, 0x10000) // SSP
		bus.Write(Long, 4, 0x1000)  // PC
		fillNOPs(bus, 0x1000, 10)

		cpu.Reset()
		if cpu.Deficit() != 0 {
			t.Errorf("Deficit() after Reset = %d, want 0", cpu.Deficit())
		}
	})
}
