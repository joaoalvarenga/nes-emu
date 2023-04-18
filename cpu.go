package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
)

type Instruction struct {
	Name     string `json:"name"`
	AddrMode string `json:"addr_mode"`
	Cycles   uint8  `json:"cycles"`
}

type CPU struct {
	accumulator uint8
	xRegister   uint8
	yRegister   uint8
	stkp        uint8
	pc          uint16
	status      uint8

	fetched        uint8
	addrAbs        uint16
	addrRel        uint16
	previousOpcode uint8
	previousPc     uint16
	opcode         uint8
	cycles         uint8

	bus    *Bus
	lookup []Instruction
}

var AddressModes = map[string]func(*CPU) uint8{
	"IMP": IMP,
	"IMM": IMM,
	"ZP0": ZP0,
	"ZPX": ZPX,
	"ZPY": ZPY,
	"REL": REL,
	"ABS": ABS,
	"ABX": ABX,
	"ABY": ABY,
	"IND": IND,
	"IZX": IZX,
	"IZY": IZY,
}

func IMP(c *CPU) uint8 {
	c.fetched = c.accumulator
	return 0
}

func IMM(c *CPU) uint8 {
	c.addrAbs = c.pc
	c.pc++
	return 0
}

func ZP0(c *CPU) uint8 {
	c.addrAbs = uint16(c.read(c.pc))
	c.pc++
	c.addrAbs &= 0x00FF
	return 0
}

func ZPX(c *CPU) uint8 {
	c.addrAbs = uint16(c.read(c.pc)) + uint16(c.xRegister)
	c.pc++
	c.addrAbs &= 0x00FF
	return 0
}

func ZPY(c *CPU) uint8 {
	c.addrAbs = uint16(c.read(c.pc)) + uint16(c.yRegister)
	c.pc++
	c.addrAbs &= 0x00FF
	return 0
}

func REL(c *CPU) uint8 {
	c.addrRel = uint16(c.read(c.pc))
	c.pc++
	if c.addrRel&0x80 != 0 {
		c.addrRel |= 0xFF00
	}
	return 0
}

func ABS(c *CPU) uint8 {
	lo := c.read(c.pc)
	c.pc++
	hi := c.read(c.pc)
	c.pc++
	c.addrAbs = (uint16(hi) << 8) | uint16(lo)
	return 0
}

func ABX(c *CPU) uint8 {
	lo := c.read(c.pc)
	c.pc++
	hi := c.read(c.pc)
	c.pc++

	c.addrAbs = (uint16(hi) << 8) | uint16(lo)
	c.addrAbs += uint16(c.xRegister)

	if (c.addrAbs & 0xFF00) != (uint16(hi) << 8) {
		return 1
	}
	return 0
}

func ABY(c *CPU) uint8 {
	lo := c.read(c.pc)
	c.pc++
	hi := c.read(c.pc)
	c.pc++

	c.addrAbs = (uint16(hi) << 8) | uint16(lo)
	c.addrAbs += uint16(c.yRegister)

	if (c.addrAbs & 0xFF00) != (uint16(hi) << 8) {
		return 1
	}
	return 0
}

func IND(c *CPU) uint8 {
	ptrLo := c.read(c.pc)
	c.pc++
	ptrHi := c.read(c.pc)
	c.pc++

	ptr := (uint16(ptrHi) << 8) | uint16(ptrLo)
	if ptrLo == 0x00FF {
		c.addrAbs = (uint16(c.read(ptr&0xFF00)) << 8) | uint16(c.read(ptr))
		return 0
	}

	c.addrAbs = (uint16(c.read(ptr+1)) << 8) | uint16(c.read(ptr))
	return 0
}

func IZX(c *CPU) uint8 {
	t := uint16(c.read(c.pc))
	c.pc++

	lo := uint16(c.read((t + uint16(c.xRegister)) & 0x00FF))
	hi := uint16(c.read((t + uint16(c.xRegister) + 1) & 0x00FF))
	c.addrAbs = (hi << 8) | lo
	return 0
}

func IZY(c *CPU) uint8 {
	t := uint16(c.read(c.pc))
	c.pc++

	lo := uint16(c.read(t & 0x00FF))
	hi := uint16(c.read((t + 1) & 0x00FF))

	c.addrAbs = (hi << 8) | lo
	c.addrAbs += uint16(c.yRegister)

	if (c.addrAbs & 0xFF00) != (hi << 8) {
		return 1
	}
	return 0
}

var Operations = map[string]func(*CPU) uint8{
	"ADC": ADC,
	"SBC": SBC,
	"AND": AND,
	"ASL": ASL,
	"BCC": BCC,
	"BCS": BCS,
	"BEQ": BEQ,
	"BIT": BIT,
	"BMI": BMI,
	"BNE": BNE,
	"BPL": BPL,
	"BRK": BRK,
	"BVC": BVC,
	"BVS": BVS,
	"CLC": CLC,
	"CLD": CLD,
	"CLI": CLI,
	"CLV": CLV,
	"CMP": CMP,
	"CPX": CPX,
	"CPY": CPY,
	"DEC": DEC,
	"DEX": DEX,
	"DEY": DEY,
	"EOR": EOR,
	"INC": INC,
	"INX": INX,
	"INY": INY,
	"JMP": JMP,
	"JSR": JSR,
	"LDA": LDA,
	"LDX": LDX,
	"LDY": LDY,
	"LSR": LSR,
	"NOP": NOP,
	"ORA": ORA,
	"PHA": PHA,
	"PHP": PHP,
	"PLA": PLA,
	"PLP": PLP,
	"ROL": ROL,
	"ROR": ROR,
	"RTI": RTI,
	"RTS": RTS,
	"SEC": SEC,
	"SED": SED,
	"SEI": SEI,
	"STA": STA,
	"STX": STX,
	"STY": STY,
	"TAX": TAX,
	"TAY": TAY,
	"TSX": TSX,
	"TXA": TXA,
	"TXS": TXS,
	"TYA": TYA,
	"XXX": XXX,
}

func ADC(c *CPU) uint8 {
	c.fetch()
	temp := uint16(c.accumulator) + uint16(c.fetched) + uint16(c.getFlag(C))
	c.setFlag(C, temp > 255)
	c.setFlag(Z, (temp&0x00FF) == 0)
	overflow := ((^(uint16(c.accumulator) ^ uint16(c.fetched))) & (uint16(c.accumulator) ^ temp)) & 0x0080
	c.setFlag(V, overflow != 0)
	c.setFlag(N, (temp&0x80) != 0)
	c.accumulator = uint8(temp & 0x00FF)
	return 1
}

func SBC(c *CPU) uint8 {
	c.fetch()
	value := uint16(c.fetched) ^ 0x00FF
	temp := uint16(c.accumulator) + value + uint16(c.getFlag(C))
	c.setFlag(C, (temp&0xFF00) != 0)
	c.setFlag(Z, (temp&0x00FF) == 0)
	c.setFlag(V, ((temp^uint16(c.accumulator))&(temp^value)&0x0080) != 0)
	c.setFlag(N, temp&0x0080 != 0)
	c.accumulator = uint8(temp & 0x00FF)
	return 1
}

func AND(c *CPU) uint8 {
	c.fetch()
	c.accumulator = c.accumulator & c.fetched
	c.setFlag(Z, c.accumulator == 0x00)
	c.setFlag(N, c.accumulator&0x80 != 0)
	return 1
}

func ASL(c *CPU) uint8 {
	c.fetch()
	temp := uint16(c.fetched) << 1
	c.setFlag(C, (temp&0xFF00) > 0)
	c.setFlag(Z, (temp&0x00FF) == 0x00)
	c.setFlag(N, temp&0x80 != 0)
	if c.lookup[c.opcode].AddrMode == "IMP" {
		c.accumulator = uint8(temp & 0x00FF)
		return 0
	}
	c.write(c.addrAbs, uint8(temp&0x00FF))
	return 0
}

func (c *CPU) branch() {
	c.cycles++
	c.addrAbs = c.pc + c.addrRel
	if (c.addrAbs & 0xFF00) != (c.pc & 0xFF00) {
		c.cycles++
	}
	c.pc = c.addrAbs
}

func BCC(c *CPU) uint8 {
	if c.getFlag(C) == 0 {
		c.branch()
	}
	return 0
}

func BCS(c *CPU) uint8 {
	if c.getFlag(C) == 1 {
		c.branch()
	}
	return 0
}

func BEQ(c *CPU) uint8 {
	if c.getFlag(Z) == 1 {
		c.branch()
	}
	return 0
}

func BIT(c *CPU) uint8 {
	c.fetch()
	temp := c.accumulator & c.fetched
	c.setFlag(Z, (temp&0x00FF) == 0x00)
	c.setFlag(N, (c.fetched&(1<<7)) != 0)
	c.setFlag(V, (c.fetched&(1<<6)) != 0)
	return 0
}

func BMI(c *CPU) uint8 {
	if c.getFlag(N) == 1 {
		c.branch()
	}
	return 0
}

func BNE(c *CPU) uint8 {
	if c.getFlag(Z) == 0 {
		c.branch()
	}
	return 0
}

func BPL(c *CPU) uint8 {
	if c.getFlag(N) == 0 {
		c.branch()
	}
	return 0
}

func BRK(c *CPU) uint8 {
	c.pc++
	c.setFlag(I, true)
	c.write(0x0100+uint16(c.stkp), uint8((c.pc>>8)&0x00FF))
	c.stkp--
	c.write(0x0100+uint16(c.stkp), uint8(c.pc&0x00FF))
	c.stkp--

	c.setFlag(B, true)
	c.write(0x0100+uint16(c.stkp), c.status)
	c.stkp--
	c.setFlag(B, false)

	c.pc = uint16(c.read(0xFFFE)) | (uint16(c.read(0xFFFF)) << 8)
	return 0
}

func BVC(c *CPU) uint8 {
	if c.getFlag(V) == 0 {
		c.branch()
	}
	return 0
}

func BVS(c *CPU) uint8 {
	if c.getFlag(V) == 1 {
		c.branch()
	}
	return 0
}

func CLC(c *CPU) uint8 {
	c.setFlag(C, false)
	return 0
}

func CLD(c *CPU) uint8 {
	c.setFlag(D, false)
	return 0
}

func CLI(c *CPU) uint8 {
	c.setFlag(I, false)
	return 0
}

func CLV(c *CPU) uint8 {
	c.setFlag(V, false)
	return 0
}

func CMP(c *CPU) uint8 {
	c.fetch()
	temp := uint16(c.accumulator) - uint16(c.fetched)
	c.setFlag(C, c.accumulator >= c.fetched)
	c.setFlag(Z, (temp&0x00FF) == 0x0000)
	c.setFlag(N, (temp&0x0080) != 0)
	return 1
}

func CPX(c *CPU) uint8 {
	c.fetch()
	temp := uint16(c.xRegister) - uint16(c.fetched)
	c.setFlag(C, c.xRegister >= c.fetched)
	c.setFlag(Z, (temp&0x00FF) == 0x0000)
	c.setFlag(N, (temp&0x0080) != 0)
	return 0
}

func CPY(c *CPU) uint8 {
	c.fetch()
	temp := uint16(c.yRegister) - uint16(c.fetched)
	c.setFlag(C, c.yRegister >= c.fetched)
	c.setFlag(Z, (temp&0x00FF) == 0x0000)
	c.setFlag(N, (temp&0x0080) != 0)
	return 0
}

func DEC(c *CPU) uint8 {
	c.fetch()
	temp := c.fetched - 1
	c.write(c.addrAbs, temp&0x00FF)
	c.setFlag(Z, (temp&0x00FF) == 0x0000)
	c.setFlag(N, (temp&0x0080) != 0)
	return 0
}

func DEX(c *CPU) uint8 {
	c.xRegister--
	c.setFlag(Z, c.xRegister == 0x00)
	c.setFlag(N, (c.xRegister&0x80) != 0)
	return 0
}

func DEY(c *CPU) uint8 {
	c.yRegister--
	c.setFlag(Z, c.yRegister == 0x00)
	c.setFlag(N, (c.yRegister&0x80) != 0)
	return 0
}

func EOR(c *CPU) uint8 {
	c.fetch()
	c.accumulator = c.accumulator ^ c.fetched
	c.setFlag(Z, c.accumulator == 0x00)
	c.setFlag(N, (c.accumulator&0x80) != 0)
	return 1
}

func INC(c *CPU) uint8 {
	c.fetch()
	temp := c.fetched + 1
	c.write(c.addrAbs, temp&0x00FF)
	c.setFlag(Z, (temp&0x00FF) == 0x0000)
	c.setFlag(N, (temp&0x0080) != 0)
	return 0
}

func INX(c *CPU) uint8 {
	c.xRegister++
	c.setFlag(Z, c.xRegister == 0x00)
	c.setFlag(N, (c.xRegister&0x80) != 0)
	return 0
}

func INY(c *CPU) uint8 {
	c.yRegister++
	c.setFlag(Z, c.yRegister == 0x00)
	c.setFlag(N, (c.yRegister&0x80) != 0)
	return 0
}

func JMP(c *CPU) uint8 {
	c.pc = c.addrAbs
	return 0
}

func JSR(c *CPU) uint8 {
	c.pc--
	c.write(0x0100+uint16(c.stkp), uint8((c.pc>>8)&0x00FF))
	c.stkp--
	c.write(0x0100+uint16(c.stkp), uint8(c.pc&0x00FF))
	c.stkp--

	c.pc = c.addrAbs
	return 0
}

func LDA(c *CPU) uint8 {
	c.fetch()
	c.accumulator = c.fetched
	c.setFlag(Z, c.accumulator == 0x00)
	c.setFlag(N, (c.accumulator&0x80) != 0)
	return 1
}

func LDX(c *CPU) uint8 {
	c.fetch()
	c.xRegister = c.fetched
	c.setFlag(Z, c.xRegister == 0x00)
	c.setFlag(N, (c.xRegister&0x80) != 0)
	return 1
}

func LDY(c *CPU) uint8 {
	c.fetch()
	c.yRegister = c.fetched
	c.setFlag(Z, c.yRegister == 0x00)
	c.setFlag(N, (c.yRegister&0x80) != 0)
	return 1
}

func LSR(c *CPU) uint8 {
	c.fetch()
	c.setFlag(C, c.fetched&0x0001 != 0)
	temp := uint16(c.fetched) >> 1
	c.setFlag(Z, (temp&0x00FF) == 0x0000)
	c.setFlag(N, (temp&0x0080) != 0)
	if c.lookup[c.opcode].AddrMode == "IMP" {
		c.accumulator = uint8(temp & 0x00FF)
		return 0
	}
	c.write(c.addrAbs, uint8(temp&0x00FF))
	return 0
}

func NOP(c *CPU) uint8 {
	switch c.opcode {
	case 0x1C, 0x3C, 0x5C, 0x7C, 0xDC, 0xFC:
		return 1
	}
	return 0
}

func ORA(c *CPU) uint8 {
	c.fetch()
	c.accumulator |= c.fetched
	c.setFlag(Z, c.accumulator == 0x00)
	c.setFlag(N, (c.accumulator&0x80) != 0)
	return 1
}

func PHA(c *CPU) uint8 {
	c.write(0x0100+uint16(c.stkp), c.accumulator)
	c.stkp--
	return 0
}

func PHP(c *CPU) uint8 {
	c.write(0x0100+uint16(c.stkp), c.status|uint8(B)|uint8(U))
	c.setFlag(B, false)
	c.setFlag(U, false)
	c.stkp--
	return 0
}

func PLA(c *CPU) uint8 {
	c.stkp++
	c.accumulator = c.read(0x0100 + uint16(c.stkp))
	c.setFlag(Z, c.accumulator == 0x00)
	c.setFlag(N, (c.accumulator&0x80) != 0)
	return 0
}

func PLP(c *CPU) uint8 {
	c.stkp++
	c.status = c.read(0x0100 + uint16(c.stkp))
	c.setFlag(U, true)
	return 0
}

func ROL(c *CPU) uint8 {
	c.fetch()
	temp := (uint16(c.fetched) << 1) | uint16(c.getFlag(C))
	c.setFlag(C, (temp&0xFF00) != 0)
	c.setFlag(Z, (temp&0x00FF) == 0x0000)
	c.setFlag(N, (temp&0x0080) != 0)
	if c.lookup[c.opcode].AddrMode == "IMP" {
		c.accumulator = uint8(temp & 0x00FF)
		return 0
	}
	c.write(c.addrAbs, uint8(temp&0x00FF))
	return 0
}

func ROR(c *CPU) uint8 {
	c.fetch()
	temp := (uint16(c.getFlag(C)) << 7) | (uint16(c.fetched) >> 1)
	c.setFlag(C, c.fetched&0x01 != 0)
	c.setFlag(Z, (temp&0x00FF) == 0x00)
	c.setFlag(N, temp&0x0080 != 0)
	if c.lookup[c.opcode].AddrMode == "IMP" {
		c.accumulator = uint8(temp & 0x00FF)
	} else {
		c.write(c.addrAbs, uint8(temp&0x00FF))
	}
	return 0
}

func RTI(c *CPU) uint8 {
	c.stkp++
	c.status = c.read(0x0100 + uint16(c.stkp))
	c.status &= ^uint8(B)
	c.status &= ^uint8(U)

	c.stkp++
	c.pc = uint16(c.read(0x0100 + uint16(c.stkp)))
	c.stkp++
	c.pc |= uint16(c.read(0x0100+uint16(c.stkp))) << 8
	return 0
}

func RTS(c *CPU) uint8 {
	c.stkp++
	c.pc = uint16(c.read(0x0100 + uint16(c.stkp)))
	c.stkp++
	c.pc |= uint16(c.read(0x0100+uint16(c.stkp))) << 8
	c.pc++
	return 0
}

func SEC(c *CPU) uint8 {
	c.setFlag(C, true)
	return 0
}

func SED(c *CPU) uint8 {
	c.setFlag(D, true)
	return 0
}

func SEI(c *CPU) uint8 {
	c.setFlag(I, true)
	return 0
}

func STA(c *CPU) uint8 {
	c.write(c.addrAbs, c.accumulator)
	return 0
}

func STX(c *CPU) uint8 {
	c.write(c.addrAbs, c.xRegister)
	return 0
}

func STY(c *CPU) uint8 {
	c.write(c.addrAbs, c.yRegister)
	return 0
}

func TAX(c *CPU) uint8 {
	c.xRegister = c.accumulator
	c.setFlag(Z, c.xRegister == 0x00)
	c.setFlag(N, (c.xRegister&0x80) != 0)
	return 0
}

func TAY(c *CPU) uint8 {
	c.yRegister = c.accumulator
	c.setFlag(Z, c.yRegister == 0x00)
	c.setFlag(N, (c.yRegister&0x80) != 0)
	return 0
}

func TSX(c *CPU) uint8 {
	c.xRegister = c.stkp
	c.setFlag(Z, c.xRegister == 0x00)
	c.setFlag(N, (c.xRegister&0x80) != 0)
	return 0
}

func TXA(c *CPU) uint8 {
	c.accumulator = c.xRegister
	c.setFlag(Z, c.accumulator == 0x00)
	c.setFlag(N, (c.accumulator&0x80) != 0)
	return 0
}

func TXS(c *CPU) uint8 {
	c.stkp = c.xRegister
	return 0
}

func TYA(c *CPU) uint8 {
	c.accumulator = c.yRegister
	c.setFlag(Z, c.accumulator == 0x00)
	c.setFlag(N, (c.accumulator&0x80) != 0)
	return 0
}

func XXX(c *CPU) uint8 {
	return 0
}

type CPUFlag uint8

const (
	C = CPUFlag(1 << 0)
	Z = CPUFlag(1 << 1)
	I = CPUFlag(1 << 2)
	D = CPUFlag(1 << 3)
	B = CPUFlag(1 << 4)
	U = CPUFlag(1 << 5)
	V = CPUFlag(1 << 6)
	N = CPUFlag(1 << 7)
)

func (c *CPU) isComplete() bool {
	return c.cycles == 0
}

func (c *CPU) getFlag(flag CPUFlag) uint8 {
	if c.status&uint8(flag) != 0 {
		return 1
	}
	return 0
}

func (c *CPU) setFlag(flag CPUFlag, v bool) {
	if v {
		c.status |= uint8(flag)
	} else {
		c.status &= ^uint8(flag)
	}
}

func (c *CPU) fetch() uint8 {
	if c.lookup[c.opcode].AddrMode != "IMP" {
		c.fetched = c.read(c.addrAbs)
	}
	return c.fetched
}

func (c *CPU) read(addr uint16) uint8 {
	return c.bus.cpuRead(addr, false)
}

func (c *CPU) write(addr uint16, data uint8) {
	c.bus.cpuWrite(addr, data)
}

func (c *CPU) connectBus(bus *Bus) {
	c.bus = bus
}

func (c *CPU) irq() {
	if c.getFlag(I) == 0 {
		c.write(0x0100+uint16(c.stkp), uint8((c.pc>>8)&0x00FF))
		c.stkp--
		c.write(0x0100+uint16(c.stkp), uint8(c.pc&0x00FF))
		c.stkp--

		c.setFlag(B, false)
		c.setFlag(U, true)
		c.setFlag(I, true)
		c.write(0x0100+uint16(c.stkp), c.status)
		c.stkp--

		c.addrAbs = 0xFFFE
		lo := uint16(c.read(c.addrAbs))
		hi := uint16(c.read(c.addrAbs + 1))
		c.pc = (hi << 8) | lo

		c.cycles = 7
	}
}

func (c *CPU) nmi() {
	c.write(0x0100+uint16(c.stkp), uint8((c.pc>>8)&0x00FF))
	c.stkp--
	c.write(0x0100+uint16(c.stkp), uint8(c.pc&0x00FF))
	c.stkp--

	c.setFlag(B, false)
	c.setFlag(U, true)
	c.setFlag(I, true)
	c.write(0x0100+uint16(c.stkp), c.status)
	c.stkp--

	c.addrAbs = 0xFFFA
	lo := uint16(c.read(c.addrAbs))
	hi := uint16(c.read(c.addrAbs + 1))
	c.pc = (hi << 8) | lo

	c.cycles = 8
}

func (c *CPU) clock() {
	if c.cycles == 0 {
		c.previousOpcode = c.opcode
		c.opcode = c.read(c.pc)
		c.setFlag(U, true)

		c.previousPc = c.pc
		c.pc++

		c.cycles = c.lookup[c.opcode].Cycles

		_, ok := Operations[c.lookup[c.opcode].Name]
		if !ok {
			panic("jesus")
		}

		additionalCycle1 := AddressModes[c.lookup[c.opcode].AddrMode](c)
		additionalCycle2 := Operations[c.lookup[c.opcode].Name](c)
		c.cycles += additionalCycle1 & additionalCycle2
		c.setFlag(U, true)
	}
	c.cycles--
}

func loadInstructions() []Instruction {

	jsonFile, err := os.Open("lookup.json")
	// if we os.Open returns an error then handle it
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println("Successfully load lookup.json")
	// defer the closing of our jsonFile so that we can parse it later on
	defer jsonFile.Close()

	byteValue, _ := io.ReadAll(jsonFile)

	var result []Instruction
	errJson := json.Unmarshal(byteValue, &result)
	if errJson != nil {
		panic(errJson)
	}
	return result
}

func NewCPU() *CPU {

	cpu := &CPU{
		accumulator: 0x00,
		xRegister:   0x00,
		yRegister:   0x00,
		stkp:        0x00,
		pc:          0x0000,
		status:      0x00,
		fetched:     0x00,
		addrAbs:     0x0000,
		addrRel:     0x0000,
		opcode:      0x00,
		cycles:      0x00,
	}
	cpu.lookup = loadInstructions()

	return cpu
}

func (c *CPU) reset() {
	c.addrAbs = 0xFFFC
	lo := uint16(c.read(c.addrAbs))
	hi := uint16(c.read(c.addrAbs + 1))

	c.pc = (hi << 8) | lo

	c.accumulator = 0
	c.xRegister = 0
	c.yRegister = 0
	c.stkp = 0xFD
	c.status = 0x00 | uint8(U)

	c.addrRel = 0x0000
	c.addrAbs = 0x0000
	c.fetched = 0x00

	c.cycles = 8
}

type DissambledInstruction struct {
	instruction  string
	nextAddr     uint16
	previousAddr uint16
}

func (c *CPU) disassemble(nStart uint16, nStop uint16) map[uint16]DissambledInstruction {
	addr := uint32(nStart)
	value := uint8(0)
	lo := uint8(0)
	hi := uint8(0)
	mapLines := make(map[uint16]DissambledInstruction)
	lineAddr := uint16(0)
	for addr <= uint32(nStop) {
		previousAddr := lineAddr
		lineAddr = uint16(addr)
		sInst := "$" + numToHex(int(addr), 4) + ": "
		opcode := c.bus.cpuRead(uint16(addr), true)
		addr++
		sInst += c.lookup[opcode].Name + " "

		if c.lookup[opcode].AddrMode == "IMP" {
			sInst += " {IMP}"
		} else if c.lookup[opcode].AddrMode == "IMM" {
			value = c.bus.cpuRead(uint16(addr), true)
			addr++
			sInst += "#$" + numToHex(int(value), 2) + " {IMM}"
		} else if c.lookup[opcode].AddrMode == "ZP0" {
			lo = c.bus.cpuRead(uint16(addr), true)
			addr++
			hi = 0x00
			sInst += "$" + numToHex(int(lo), 2) + " {ZP0}"
		} else if c.lookup[opcode].AddrMode == "ZPX" {
			lo = c.bus.cpuRead(uint16(addr), true)
			addr++
			hi = 0x00
			sInst += "$" + numToHex(int(lo), 2) + ", X {ZPX}"
		} else if c.lookup[opcode].AddrMode == "ZPY" {
			lo = c.bus.cpuRead(uint16(addr), true)
			addr++
			hi = 0x00
			sInst += "$" + numToHex(int(lo), 2) + ", Y {ZPY}"
		} else if c.lookup[opcode].AddrMode == "IZX" {
			lo = c.bus.cpuRead(uint16(addr), true)
			addr++
			hi = 0x00
			sInst += "($" + numToHex(int(lo), 2) + ", X) {IZX}"
		} else if c.lookup[opcode].AddrMode == "IZY" {
			lo = c.bus.cpuRead(uint16(addr), true)
			addr++
			hi = 0x00
			sInst += "($" + numToHex(int(lo), 2) + "), Y {IZY}"
		} else if c.lookup[opcode].AddrMode == "ABS" {
			lo = c.bus.cpuRead(uint16(addr), true)
			addr++
			hi = c.bus.cpuRead(uint16(addr), true)
			addr++
			sInst += "$" + numToHex(int((uint16(hi)<<8)|uint16(lo)), 4) + " {ABS}"
		} else if c.lookup[opcode].AddrMode == "ABX" {
			lo = c.bus.cpuRead(uint16(addr), true)
			addr++
			hi = c.bus.cpuRead(uint16(addr), true)
			addr++
			sInst += "$" + numToHex(int((uint16(hi)<<8)|uint16(lo)), 4) + ", X {ABX}"
		} else if c.lookup[opcode].AddrMode == "ABY" {
			lo = c.bus.cpuRead(uint16(addr), true)
			addr++
			hi = c.bus.cpuRead(uint16(addr), true)
			addr++
			d := int((uint16(hi) << 8) | uint16(lo))
			sInst += "$" + numToHex(d, 4) + ", Y {ABY}"
		} else if c.lookup[opcode].AddrMode == "IND" {
			lo = c.bus.cpuRead(uint16(addr), true)
			addr++
			hi = c.bus.cpuRead(uint16(addr), true)
			addr++
			sInst += "($" + numToHex(int((uint16(hi)<<8)|uint16(lo)), 4) + ") {IND}"
		} else if c.lookup[opcode].AddrMode == "REL" {
			addrRel := uint16(c.bus.cpuRead(uint16(addr), true))
			if addrRel&0x80 != 0 {
				addrRel |= 0xFF00
			}
			addr++
			sInst += "$" + numToHex(int(value), 2) + " [$" + numToHex(int(uint16(addr)+uint16(addrRel)), 4) + "] {REL}"
		}

		// Add the formed string to a std::map, using the instruction's
		// address as the key. This makes it convenient to look for later
		// as the instructions are variable in length, so a straight up
		// incremental index is not sufficient.
		mapLines[lineAddr] = DissambledInstruction{
			instruction:  sInst,
			previousAddr: previousAddr,
			nextAddr:     uint16(addr),
		}
	}
	return mapLines
}
