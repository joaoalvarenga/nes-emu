package main

import (
	"unsafe"
)

type Bus struct {
	systemClockCounter       uint8
	cpuRam                   []uint8
	apu                      *APU
	cpu                      *CPU
	ppu                      *PPU
	cartridge                *Cartridge
	controllerState          [2]uint8
	controller               [2]uint8
	dmaPage                  uint8
	dmaAddr                  uint8
	dmaData                  uint8
	dmaTransfer              bool
	dmaDummy                 bool
	AudioSample              chan float32
	AudioTimePerSystemSample float32
	AudioTimePerNESClock     float32
	audioTime                float32
}

func (b *Bus) cpuWrite(addr uint16, data uint8) {

	if b.cartridge.cpuWrite(addr, data) {

	} else if addr >= 0x0000 && addr <= 0x1FFF {
		b.cpuRam[addr&0x07FF] = data
	} else if addr >= 0x2000 && addr <= 0x3FFF {
		b.ppu.cpuWrite(addr&0x0007, data)
	} else if (addr >= 0x4000 && addr <= 0x4013) || addr == 0x4015 || addr == 0x4017 {
		b.apu.cpuWrite(addr, data)
	} else if addr == 0x4014 {
		b.dmaPage = data
		b.dmaAddr = 0x00
		b.dmaTransfer = true
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
	b.dmaDummy = true
	b.dmaTransfer = false
	b.dmaData = 0
	b.dmaPage = 0
	b.dmaAddr = 0
}

func (b *Bus) SetSampleFrequency(sampleRate uint32) {
	b.AudioTimePerSystemSample = 1.0 / float32(sampleRate)
	b.AudioTimePerNESClock = 1.0 / 5369318.0 // PPU Clock Frequency
}

func (b *Bus) clock() bool {
	//cpuDuration := time.Duration(0)
	//ppuDuration := time.Duration(0)

	//start := time.Now()
	b.ppu.clock()

	b.apu.clock()
	//ppuDuration = time.Now().Sub(start)

	if b.systemClockCounter%3 == 0 {
		if b.dmaTransfer {
			if b.dmaDummy {
				if b.systemClockCounter%2 == 1 {
					b.dmaDummy = false
				}
			} else {
				if b.systemClockCounter%2 == 0 {
					b.dmaData = b.cpuRead((uint16(b.dmaPage)<<8)|uint16(b.dmaAddr), false)
				} else {
					pointer := unsafe.Add(b.ppu.oamPtr, uintptr(b.dmaAddr)*unsafe.Sizeof(b.ppu.oam[0].y))
					value := (*uint8)(pointer)
					*value = b.dmaData
					b.dmaAddr++

					if b.dmaAddr == 0x00 {
						b.dmaTransfer = false
						b.dmaDummy = true
					}
				}
			}
		} else {
			//start := time.Now()
			b.cpu.clock()

			//cpuDuration = time.Now().Sub(start)
			//fmt.Printf("CPU time = %s\n", elapsed)
		}
	}

	// Synchronising with audio
	audioSampleReady := false
	b.audioTime += b.AudioTimePerNESClock
	if b.audioTime >= b.AudioTimePerSystemSample {
		b.audioTime -= b.AudioTimePerSystemSample
		select {
		case b.AudioSample <- b.apu.getOutputSample():
		default:
		}
		audioSampleReady = true
	}

	if b.ppu.nmi {
		b.ppu.nmi = false
		b.cpu.nmi()
	}

	b.systemClockCounter++
	//return cpuDuration, ppuDuration
	return audioSampleReady
}

func NewBus(cpu *CPU, ppu *PPU, apu *APU) *Bus {
	bus := &Bus{
		systemClockCounter: 0,
		cpuRam:             make([]uint8, 2048),
		cpu:                cpu,
		ppu:                ppu,
		apu:                apu,
		dmaDummy:           true,
		dmaTransfer:        false,
		dmaAddr:            0,
		dmaPage:            0,
		dmaData:            0,
		AudioSample:        make(chan float32, 44100),
	}
	return bus
}
