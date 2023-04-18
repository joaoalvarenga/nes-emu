package mapper

type Mapper0004 struct {
	targetRegister uint8
	prgBankMode    bool
	chrInversion   bool
	mirrorMode     MIRROR
	register       uint32
	chrBank        [8]uint32
	prgBank        [4]uint32
	IRQActive      bool
	IRQEnable      bool
	IRQUpdate      bool
	IRQCounter     uint16
	IRQReload      uint16
	ramStatic      []uint8
	prgBanks       uint8
	chrBanks       uint8
}
