package mapper

type Mapper0003 struct {
	PrgBanks       uint8
	ChrBanks       uint8
	chrBanksSelect uint8
}

func (m *Mapper0003) CpuMapRead(addr uint16, mappedAddr *uint32) bool {
	if addr >= 0x8000 && addr <= 0xFFFF {
		if m.PrgBanks == 1 {
			*mappedAddr = uint32(addr & 0x3FFF)
		}
		if m.PrgBanks == 2 {
			*mappedAddr = uint32(addr & 0x7FFF)
		}
		return true
	}
	return false
}

func (m *Mapper0003) CpuMapWrite(addr uint16, mappedAddr *uint32, data uint8) bool {
	if addr >= 0x8000 && addr <= 0xFFFF {
		m.chrBanksSelect = data & 0x03
		*mappedAddr = uint32(addr)
	}
	return false
}

func (m *Mapper0003) PpuMapRead(addr uint16, mappedAddr *uint32) bool {
	if addr < 0x2000 {
		*mappedAddr = uint32(m.chrBanksSelect)*0x2000 + uint32(addr)
		return true
	}
	return false
}

func (m *Mapper0003) PpuMapWrite(addr uint16, mappedAddr *uint32, data uint8) bool {
	return false
}

func (m *Mapper0003) Reset() {
	m.chrBanksSelect = 0
}

func (m *Mapper0003) Mirror() MIRROR {
	return HARDWARE
}
func (m *Mapper0003) irqState() bool {
	return false
}
func (m *Mapper0003) irqClear() {
}
func (m *Mapper0003) scanline() {
}
