package mapper

type Mapper0002 struct {
	PrgBankSelectLo uint8
	PrgBankSelectHi uint8
	PrgBanks        uint8
	ChrBanks        uint8
}

func (m *Mapper0002) CpuMapRead(addr uint16, mappedAddr *uint32) bool {
	if addr >= 0x8000 && addr <= 0xBFFF {
		*mappedAddr = uint32(m.PrgBankSelectLo)*0x4000 + uint32(addr&0x3FFF)
		return true
	}

	if addr >= 0xC000 && addr <= 0xFFFF {
		*mappedAddr = uint32(m.PrgBankSelectHi)*0x4000 + uint32(addr&0x3FFF)
		return true
	}

	return false
}

func (m *Mapper0002) CpuMapWrite(addr uint16, mappedAddr *uint32, data uint8) bool {
	if addr >= 0x8000 && addr <= 0xFFFF {
		m.PrgBankSelectLo = data & 0x0F
	}

	return false
}

func (m *Mapper0002) PpuMapRead(addr uint16, mappedAddr *uint32) bool {
	if addr < 0x2000 {
		*mappedAddr = uint32(addr)
		return true
	}
	return false
}

func (m *Mapper0002) PpuMapWrite(addr uint16, mappedAddr *uint32, data uint8) bool {
	if addr < 0x2000 {
		if m.ChrBanks == 0 {
			*mappedAddr = uint32(addr)
			return true
		}
	}
	return false
}

func (m *Mapper0002) Reset() {
	m.PrgBankSelectLo = 0
	m.PrgBankSelectHi = m.PrgBanks - 1
}

func (m *Mapper0002) Mirror() MIRROR {
	return HARDWARE
}
func (m *Mapper0002) irqState() bool {
	return false
}
func (m *Mapper0002) irqClear() {
}
func (m *Mapper0002) scanline() {
}
