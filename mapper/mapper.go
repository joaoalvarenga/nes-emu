package mapper

type MIRROR uint8

const (
	HARDWARE     = MIRROR(0)
	HORIZONTAL   = MIRROR(1)
	VERTICAL     = MIRROR(2)
	ONESCREEN_LO = MIRROR(3)
	ONESCREEN_HI = MIRROR(4)
)

type Mapper interface {
	CpuMapRead(addr uint16, mappedAddr *uint32) bool
	CpuMapWrite(addr uint16, mappedAddr *uint32, data uint8) bool
	PpuMapRead(addr uint16, mappedAddr *uint32) bool
	PpuMapWrite(addr uint16, mappedAddr *uint32, data uint8) bool
	Reset()
	Mirror() MIRROR
	irqState() bool
	irqClear()
	scanline()
}
