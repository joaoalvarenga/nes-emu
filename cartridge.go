package main

import (
	"encoding/binary"
	"nes-emu/mapper"
	"os"
)

type MIRROR uint8

const (
	HORIZONTAL   = MIRROR(0)
	VERTICAL     = MIRROR(1)
	ONESCREEN_LO = MIRROR(2)
	ONESCREEN_HI = MIRROR(3)
)

type Cartridge struct {
	prgBanks  uint8
	chrBanks  uint8
	prgMemory []uint8
	chrMemory []uint8
	mapper    mapper.Mapper
	mirror    MIRROR
}

type Header struct {
	Name         [4]byte
	PrgRomChunks uint8
	ChrRomChunks uint8
	Mapper1      uint8
	Mapper2      uint8
	PrgRamSize   uint8
	TvSystem1    uint8
	TvSystem2    uint8
	Unused       [5]byte
}

func (c *Cartridge) cpuRead(addr uint16, data *uint8) bool {
	mappedAddr := uint32(0)
	if c.mapper.CpuMapRead(addr, &mappedAddr) {
		*data = c.prgMemory[mappedAddr]
		return true
	}
	return false
}

func (c *Cartridge) cpuWrite(addr uint16, data uint8) bool {
	mappedAddr := uint32(0)
	if c.mapper.CpuMapWrite(addr, &mappedAddr, data) {
		c.prgMemory[mappedAddr] = data
		return true
	}
	return false
}

func (c *Cartridge) ppuRead(addr uint16, data *uint8) bool {
	mappedAddr := uint32(0)
	if c.mapper.PpuMapRead(addr, &mappedAddr) {
		*data = c.chrMemory[mappedAddr]
		return true
	}
	return false
}

func (c *Cartridge) ppuWrite(addr uint16, data uint8) bool {
	mappedAddr := uint32(0)
	if c.mapper.PpuMapWrite(addr, &mappedAddr, data) {
		c.chrMemory[mappedAddr] = data
		return true
	}
	return false
}

func (c *Cartridge) reset() {
	if c.mapper != nil {
		c.mapper.Reset()
	}
}

func NewCartridge(filename string) *Cartridge {

	file, _ := os.Open(filename)

	defer file.Close()

	header := Header{}
	binary.Read(file, binary.LittleEndian, &header)
	if header.Mapper1&0x04 != 0 {
		file.Seek(512, 1)
	}

	cart := &Cartridge{}

	mapperId := ((header.Mapper2 >> 4) << 4) | (header.Mapper1 >> 4)
	cart.mirror = HORIZONTAL
	if header.Mapper1&0x01 != 0 {
		cart.mirror = VERTICAL
	}

	fileType := 1
	if fileType == 1 {
		cart.prgBanks = header.PrgRomChunks
		cart.prgMemory = make([]uint8, uint32(cart.prgBanks)*16384)
		binary.Read(file, binary.LittleEndian, &cart.prgMemory)

		cart.chrBanks = header.ChrRomChunks
		if cart.chrBanks == 0 {
			cart.chrMemory = make([]uint8, 8192)
		} else {
			cart.chrMemory = make([]uint8, uint32(cart.chrBanks)*8192)
		}
		binary.Read(file, binary.LittleEndian, &cart.chrMemory)
	}

	switch mapperId {
	case 0:
		cart.mapper = mapper.Mapper0000{
			PrgBanks: cart.prgBanks,
			ChrBanks: cart.chrBanks,
		}
	}

	return cart
}
