package ppu

type Field struct {
	Index uint16
	Size  uint16
}

type Register struct {
	fields map[string]Field
	values map[string]uint16
	Reg    uint16
}

func (r *Register) SetField(key string, value uint16) {
	field, ok := r.fields[key]
	if !ok {
		return
	}

	mask := uint16(((^(0xFFFF << field.Size) & 0xFFFF) << field.Index) & 0xFFFF)
	negativeMask := ^mask & 0xFFFF
	r.Reg = (r.Reg & negativeMask) | (mask & (value << field.Index))
}

func (r *Register) GetField(key string) uint16 {
	field, ok := r.fields[key]
	if !ok {
		panic("Field " + key + " not found")
	}
	mask := uint16(((^(0xFFFF << field.Size) & 0xFF) << field.Index) & 0xFFFF)
	return (r.Reg & mask) >> field.Index
}

func (r *Register) allAttributes() map[string]uint16 {
	out := make(map[string]uint16)
	for k := range r.fields {
		out[k] = r.GetField(k)
	}
	return out
}

func CreateRegister(fields map[string]Field) Register {
	return Register{
		fields: fields,
		Reg:    uint16(0),
	}
}
