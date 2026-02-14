# m68k

A clean-room Motorola 68000 (MC68000) CPU emulator written in Go.

## Overview

This package implements a cycle-approximate, instruction-accurate emulation of
the MC68000 processor. The MC68000 is a 32-bit internal / 16-bit external CISC
processor featuring:

- Eight 32-bit data registers (D0-D7)
- Eight 32-bit address registers (A0-A7), where A7 is the stack pointer
- A 32-bit program counter (24-bit external address bus)
- A 16-bit status register (system byte + condition code register)
- Dual stack pointers (USP for user mode, SSP for supervisor mode)

The full MC68000 instruction set is implemented.

## Usage

The CPU communicates with the outside world through the `Bus` interface. Callers
provide a `Bus` implementation that maps reads and writes to memory, I/O, or
other hardware.

```go
package main

import "github.com/user-none/go-chip-m68k"

type MyBus struct {
    rom [512 * 1024]byte
    ram [64 * 1024]byte
}

func (b *MyBus) Read(sz m68k.Size, addr uint32) uint32  { /* ... */ }
func (b *MyBus) Write(sz m68k.Size, addr uint32, val uint32) { /* ... */ }
func (b *MyBus) Reset() {}

func main() {
    bus := &MyBus{}
    // Load ROM data into bus.rom ...

    cpu := m68k.New(bus)

    for !cpu.Halted() {
        cycles := cpu.Step()
        // Use cycles for timing synchronization
    }
}
```

## Bus Interface

```go
type Bus interface {
    Read(op Size, addr uint32) uint32
    Write(op Size, addr uint32, val uint32)
    Reset()
}
```

All addresses passed to `Bus` methods are masked to 24 bits by the CPU. The
`Size` parameter indicates the access width (`Byte`, `Word`, or `Long`). Word
and long accesses to odd addresses are detected by the CPU and cause an address
error before reaching the bus.

`Reset()` is called when the CPU executes a RESET instruction, allowing the bus
to reset connected peripherals.

## API

### CPU Lifecycle

| Function | Description |
|---|---|
| `New(bus Bus) *CPU` | Create a CPU and perform a hardware reset |
| `Reset()` | Hardware reset: load SSP from 0x0, PC from 0x4, enter supervisor mode |
| `Step() int` | Execute one instruction, return cycles consumed |
| `Halted() bool` | True if the CPU is halted (address error) |
| `Cycles() uint64` | Total cycle count since last reset |

### State Access

| Function | Description |
|---|---|
| `Registers() Registers` | Snapshot of all programmer-visible registers |
| `SetState(d, a, pc, sr, usp, ssp)` | Set all registers directly (for testing) |

### Interrupts

| Function | Description |
|---|---|
| `RequestInterrupt(level uint8, vector *uint8)` | Queue an interrupt at the given priority level (1-7) |

Pass `nil` for `vector` to use auto-vectoring. A higher priority level replaces
a pending lower-level interrupt. Level 7 is non-maskable.

### Types

```go
type Size int

const (
    Byte Size = 1  // 8-bit
    Word Size = 2  // 16-bit
    Long Size = 4  // 32-bit
)

type Registers struct {
    D   [8]uint32  // Data registers
    A   [8]uint32  // Address registers (A7 is active stack pointer)
    PC  uint32     // Program counter
    SR  uint16     // Status register
    USP uint32     // User stack pointer (shadowed)
    SSP uint32     // Supervisor stack pointer (shadowed)
    IR  uint16     // Instruction register
}
```

## Instruction Set

The complete MC68000 instruction set is organized into the following groups:

| Category | Instructions |
|---|---|
| Data Movement | MOVE, MOVEA, MOVEQ, MOVEP, MOVEM, LEA, PEA, EXG, SWAP |
| Arithmetic | ADD, ADDA, ADDI, ADDQ, ADDX, SUB, SUBA, SUBI, SUBQ, SUBX, CMP, CMPA, CMPI, CMPM, MULU, MULS, DIVU, DIVS, NEG, NEGX, CLR, EXT, CHK |
| Logical | AND, ANDI, OR, ORI, EOR, EORI, NOT, TST, TAS |
| Shift/Rotate | ASL, ASR, LSL, LSR, ROL, ROR, ROXL, ROXR |
| Bit Manipulation | BTST, BCHG, BCLR, BSET |
| Branch/Jump | Bcc, BRA, BSR, DBcc, JMP, JSR, RTS, RTE, RTR, Scc |
| BCD Arithmetic | ABCD, SBCD, NBCD |
| System Control | NOP, STOP, RESET, TRAP, TRAPV, LINK, UNLK, MOVE to/from SR, MOVE to/from CCR, MOVE USP, ANDI/ORI/EORI to CCR, ANDI/ORI/EORI to SR |

All 12 MC68000 addressing modes are supported:

| Mode | Syntax | Description |
|---|---|---|
| 0 | Dn | Data register direct |
| 1 | An | Address register direct |
| 2 | (An) | Address register indirect |
| 3 | (An)+ | Post-increment |
| 4 | -(An) | Pre-decrement |
| 5 | d16(An) | Displacement |
| 6 | d8(An,Xn) | Indexed |
| 7.0 | abs.W | Absolute short |
| 7.1 | abs.L | Absolute long |
| 7.2 | d16(PC) | PC-relative displacement |
| 7.3 | d8(PC,Xn) | PC-relative indexed |
| 7.4 | #imm | Immediate |

## Exceptions and Interrupts

The CPU supports the standard MC68000 exception model:

- **Reset** (vectors 0-1): SSP and PC initialization
- **Bus/Address Error** (vectors 2-3): Invalid memory access
- **Illegal Instruction** (vector 4): Unrecognized opcode
- **Divide by Zero** (vector 5): Division with zero divisor
- **CHK** (vector 6): Register out of bounds
- **TRAPV** (vector 7): Overflow trap
- **Privilege Violation** (vector 8): Supervisor instruction in user mode
- **Line-A / Line-F** (vectors 10-11): Unimplemented opcode lines
- **Spurious Interrupt** (vector 24): Interrupt with no vector
- **Auto-vectors** (vectors 25-31): Hardware interrupt levels 1-7
- **TRAP #0-#15** (vectors 32-47): Software traps

Interrupts are checked at the start of each `Step()` call. The interrupt mask
in the status register (bits 10-8) controls which levels are serviced. Level 7
is non-maskable.

## Design Notes

- **Cycle counts** are per-instruction accurate for most instructions, using
  addressing-mode-specific timing from the Motorola PRM. Known approximations:
  - Multiply and divide use flat worst-case values instead of calculating timing
    from the operand bit patterns: MULU (70 cycles, real range 38-70), MULS (70,
    range 38-70), DIVU (140, range 76-140), DIVS (158, range 120-158).
  - CHK exception processing uses a fixed 34-cycle cost (the standard exception
    overhead) rather than the instruction-specific timing which varies by
    addressing mode and trap condition.
  - BTST Dn,#imm uses the PRM value of 8 cycles; hardware-verified tests
    ([SingleStepTests/m68000](https://github.com/SingleStepTests/m68000)) show
    10 cycles for this specific addressing mode.
  - The EA addressing mode cost is included for all instructions.
- **Opcode dispatch** uses a 64K-entry lookup table indexed by the first
  instruction word for constant-time decode.
- **Address errors** on word/long access to odd addresses halt the CPU rather
  than pushing a full exception frame.
- **Trace exception** (T flag) is not implemented.
- **Data registers** are `uint32` internally for cleaner bit manipulation.
- **No external dependencies** beyond the Go standard library.

## Testing

Tests use a subset of the hardware-verified MC68000 reference data from
[SingleStepTests/m68000](https://github.com/SingleStepTests/m68000) to validate
instruction behavior including flag calculations, addressing modes, and edge
cases.

```
go test ./...
```
