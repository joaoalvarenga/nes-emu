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
	r.SetReg((r.Reg & negativeMask) | (mask & (value << field.Index)))

}

func (r *Register) SetReg(value uint16) {
	r.Reg = value
	for key := range r.fields {
		internalField := r.fields[key]
		mask := uint16(((^(0xFFFF << internalField.Size) & 0xFF) << internalField.Index) & 0xFFFF)
		r.values[key] = (r.Reg & mask) >> internalField.Index
	}
}

func (r *Register) GetField(key string) uint16 {
	field, ok := r.values[key]
	if !ok {
		panic("Field " + key + " not found")
	}
	return field
}

func (r *Register) allAttributes() map[string]uint16 {
	out := make(map[string]uint16)
	for k := range r.fields {
		out[k] = r.GetField(k)
	}
	return out
}

func CreateRegister(fields map[string]Field) Register {
	reg := Register{
		fields: fields,
		Reg:    uint16(0),
		values: make(map[string]uint16),
	}
	for key := range reg.fields {
		reg.values[key] = 0
	}
	return reg
}
