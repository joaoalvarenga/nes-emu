package main

type Bus struct {
	systemClockCounter uint8
	cpuRam             []uint8
	cpu                *CPU
	ppu                *PPU
	cartridge          *Cartridge
	controllerState    [2]uint8
	controller         [2]uint8
}

func (b *Bus) cpuWrite(addr uint16, data uint8) {

	if b.cartridge.cpuWrite(addr, data) {

	} else if addr >= 0x0000 && addr <= 0x1FFF {
		b.cpuRam[addr&0x07FF] = data
	} else if addr >= 0x2000 && addr <= 0x3FFF {
		b.ppu.cpuWrite(addr&0x0007, data)
	} else if addr >= 0x4016 && addr <= 0x4017 {
		b.controllerState[addr&0x0001] = b.controller[addr&0x0001]
	}
}

func (b *Bus) cpuRead(addr uint16, readOnly bool) uint8 {
	data := uint8(0)
	if b.cartridge.cpuRead(addr, &data) {

	} else if addr >= 0x0000 && addr <= 0x1FFF {
		data = b.cpuRam[addr&0x07FF]
	} else if addr >= 0x2000 && addr <= 0x3FFF {
		data = b.ppu.cpuRead(addr&0x0007, readOnly)
	} else if addr >= 0x4016 && addr <= 0x4017 {
		if b.controllerState[addr&0x0001]&0x80 > 0 {
			data = 1
		}
		b.controllerState[addr&0x0001] <<= 1
	}
	return data
}

func (b *Bus) insertCartridge(cartridge *Cartridge) {
	b.cartridge = cartridge
	b.ppu.connectCartridge(cartridge)
}

func (b *Bus) reset() {
	b.cartridge.reset()
	b.cpu.reset()
	b.ppu.reset()
	b.systemClockCounter = 0
}

func (b *Bus) clock() {
	b.ppu.clock()
	if b.systemClockCounter%3 == 0 {
		b.cpu.clock()
	}

	if b.ppu.nmi {
		b.ppu.nmi = false
		b.cpu.nmi()
	}

	b.systemClockCounter++
}

func NewBus(cpu *CPU, ppu *PPU) *Bus {
	bus := &Bus{
		systemClockCounter: 0,
		cpuRam:             make([]uint8, 2048),
		cpu:                cpu,
		ppu:                ppu,
	}
	return bus
}
