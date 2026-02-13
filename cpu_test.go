package m68k

import "testing"

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
		bus.Write(0, Long, 0, 0x10000) // SSP
		bus.Write(0, Long, 4, 0x1000)  // PC
		fillNOPs(bus, 0x1000, 10)

		cpu.Reset()
		if cpu.Deficit() != 0 {
			t.Errorf("Deficit() after Reset = %d, want 0", cpu.Deficit())
		}
	})
}

func TestBusCycleStamp(t *testing.T) {
	t.Run("reset passes cycle 0", func(t *testing.T) {
		bus := &spyBus{}
		// Set up reset vectors: SSP at addr 0, PC at addr 4
		bus.testBus.Write(0, Long, 0, 0x10000)
		bus.testBus.Write(0, Long, 4, 0x1000)

		cpu := &CPU{bus: bus}
		cpu.Reset()

		// Reset reads SSP (Long) and PC (Long) — two bus accesses
		if len(bus.cycles) < 2 {
			t.Fatalf("expected at least 2 bus accesses during Reset, got %d", len(bus.cycles))
		}
		for i, c := range bus.cycles {
			if c != 0 {
				t.Errorf("Reset bus access %d: cycle = %d, want 0", i, c)
			}
		}
	})

	t.Run("instruction accesses use pre-instruction cycle count", func(t *testing.T) {
		bus := &spyBus{}

		// MOVE.W D0, (A1) — opcode 0x3280: writes D0 to address in A1
		writeWord(&bus.testBus, 0x1000, 0x3280)

		cpu := &CPU{bus: bus}
		// Set A1 to a valid even address for the write destination
		var a [8]uint32
		a[1] = 0x2000
		cpu.SetState([8]uint32{0x1234}, a, 0x1000, 0x2700, 0, 0x10000)
		bus.cycles = nil // clear any prior accesses

		before := cpu.Cycles()
		cpu.Step()

		// All bus accesses within this instruction should have cycle == before
		if len(bus.cycles) == 0 {
			t.Fatal("expected at least one bus access")
		}
		for i, c := range bus.cycles {
			if c != before {
				t.Errorf("bus access %d: cycle = %d, want %d", i, c, before)
			}
		}
	})

	t.Run("cycle stamp advances across instructions", func(t *testing.T) {
		bus := &spyBus{}

		// Two NOPs
		writeWord(&bus.testBus, 0x1000, 0x4E71)
		writeWord(&bus.testBus, 0x1002, 0x4E71)

		cpu := &CPU{bus: bus}
		cpu.SetState([8]uint32{}, [8]uint32{}, 0x1000, 0x2700, 0, 0x10000)
		bus.cycles = nil

		// First NOP
		cpu.Step()
		first := bus.cycles[0]
		bus.cycles = nil

		// Second NOP
		cpu.Step()
		second := bus.cycles[0]

		if second <= first {
			t.Errorf("second instruction cycle = %d, want > %d", second, first)
		}
	})
}
