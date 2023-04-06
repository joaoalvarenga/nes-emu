package mapper

type Mapper0000 struct {
	PrgBanks uint8
	ChrBanks uint8
}

func (m Mapper0000) CpuMapRead(addr uint16, mappedAddr *uint32) bool {
	if addr >= 0x8000 && addr <= 0xFFFF {
		base := uint16(0x3FFF)
		if m.PrgBanks > 1 {
			base = 0x7FFF
		}
		*mappedAddr = uint32(addr & base)
		return true
	}
	return false
}

func (m Mapper0000) CpuMapWrite(addr uint16, mappedAddr *uint32, data uint8) bool {
	if addr >= 0x8000 && addr <= 0xFFFF {
		base := uint16(0x3FFF)
		if m.PrgBanks > 1 {
			base = 0x7FFF
		}
		*mappedAddr = uint32(addr & base)
		return true
	}
	return false
}

func (m Mapper0000) PpuMapRead(addr uint16, mappedAddr *uint32) bool {
	if addr >= 0x0000 && addr <= 0x1FFF {
		*mappedAddr = uint32(addr)
		return true
	}
	return false
}

func (m Mapper0000) PpuMapWrite(addr uint16, mappedAddr *uint32, data uint8) bool {
	if addr >= 0x0000 && addr <= 0x1FFF {
		if m.ChrBanks == 0 {
			*mappedAddr = uint32(addr)
			return true
		}
	}
	return false
}

func (m Mapper0000) Reset() {

}
