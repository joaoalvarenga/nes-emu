package main

import (
	"encoding/binary"
	"nes-emu/mapper"
	"os"
)

type Cartridge struct {
	prgBanks  uint8
	chrBanks  uint8
	prgMemory []uint8
	chrMemory []uint8
	mapper    mapper.Mapper
	mirror    mapper.MIRROR
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
		if mappedAddr == 0xFFFFFFFF {
			return true
		}
		*data = c.prgMemory[mappedAddr]
		return true
	}
	return false
}

func (c *Cartridge) cpuWrite(addr uint16, data uint8) bool {
	mappedAddr := uint32(0)
	if c.mapper.CpuMapWrite(addr, &mappedAddr, data) {
		if mappedAddr == 0xFFFFFFFF {
			return true
		}
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

func (c *Cartridge) Mirror() mapper.MIRROR {
	m := c.mapper.Mirror()
	if m == mapper.HARDWARE {
		return c.mirror
	}
	return m
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
	cart.mirror = mapper.HORIZONTAL
	if header.Mapper1&0x01 != 0 {
		cart.mirror = mapper.VERTICAL
	}

	fileType := 1
	if header.Mapper2&0x0C == 0x80 {
		fileType = 2
	}
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
	if fileType == 2 {
		cart.prgBanks = ((header.PrgRamSize & 0x07) << 8) | header.PrgRomChunks
		cart.prgMemory = make([]uint8, uint32(cart.prgBanks)*16384)
		binary.Read(file, binary.LittleEndian, &cart.prgMemory)

		cart.chrBanks = ((header.PrgRamSize & 0x38) << 8) | header.ChrRomChunks
		cart.chrMemory = make([]uint8, uint32(cart.chrBanks)*8192)
		binary.Read(file, binary.LittleEndian, &cart.chrMemory)
	}

	switch mapperId {
	case 0:
		cart.mapper = &mapper.Mapper0000{
			PrgBanks: cart.prgBanks,
			ChrBanks: cart.chrBanks,
		}
	case 2:
		cart.mapper = &mapper.Mapper0002{
			PrgBanks: cart.prgBanks,
			ChrBanks: cart.chrBanks,
		}
	case 3:
		cart.mapper = &mapper.Mapper0003{
			PrgBanks: cart.prgBanks,
			ChrBanks: cart.chrBanks,
		}
	}

	return cart
}
