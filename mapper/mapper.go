package mapper

type Mapper interface {
	CpuMapRead(addr uint16, mappedAddr *uint32) bool
	CpuMapWrite(addr uint16, mappedAddr *uint32, data uint8) bool
	PpuMapRead(addr uint16, mappedAddr *uint32) bool
	PpuMapWrite(addr uint16, mappedAddr *uint32, data uint8) bool
	Reset()
}
