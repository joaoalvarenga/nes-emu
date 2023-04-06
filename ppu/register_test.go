package ppu

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func Test8BitsRegister(t *testing.T) {
	r := Register{
		fields: map[string]Field{
			"sprite_overflow": {0, 1},
			"unused":          {1, 5},
			"sprite_zero_hit": {6, 1},
			"vertical_bank":   {7, 1},
		},
	}

	assert.Equal(t, uint16(0), r.Reg)
	r.SetField("vertical_bank", 1)
	assert.Equal(t, map[string]uint16{
		"sprite_overflow": 0,
		"unused":          0,
		"sprite_zero_hit": 0,
		"vertical_bank":   1,
	}, r.allAttributes())
	assert.Equal(t, uint16(0b10000000), r.Reg)

	r.SetField("unused", 31)
	assert.Equal(t, map[string]uint16{
		"sprite_overflow": 0,
		"unused":          31,
		"sprite_zero_hit": 0,
		"vertical_bank":   1,
	}, r.allAttributes())
	assert.Equal(t, uint16(0b10111110), r.Reg)

	r.SetField("sprite_overflow", 1)
	assert.Equal(t, map[string]uint16{
		"sprite_overflow": 1,
		"unused":          31,
		"sprite_zero_hit": 0,
		"vertical_bank":   1,
	}, r.allAttributes())
	assert.Equal(t, uint16(0b10111111), r.Reg)

	r.SetField("unused", 2)
	assert.Equal(t, map[string]uint16{
		"sprite_overflow": 1,
		"unused":          2,
		"sprite_zero_hit": 0,
		"vertical_bank":   1,
	}, r.allAttributes())
	assert.Equal(t, uint16(0b10000101), r.Reg)
}

func Test16BitsRegister(t *testing.T) {
	r := Register{
		fields: map[string]Field{
			"coarse_x":    {0, 5},
			"coarse_y":    {5, 5},
			"nametable_x": {10, 1},
			"nametable_y": {11, 1},
			"fine_y":      {12, 3},
			"unused":      {15, 1},
		},
	}

	assert.Equal(t, uint16(0), r.Reg)
	r.SetField("coarse_x", 31)
	assert.Equal(t, map[string]uint16{
		"coarse_x":    31,
		"coarse_y":    0,
		"nametable_x": 0,
		"nametable_y": 0,
		"fine_y":      0,
		"unused":      0,
	}, r.allAttributes())
	assert.Equal(t, uint16(0b0000000000011111), r.Reg)

	r.SetField("coarse_y", 31)
	assert.Equal(t, map[string]uint16{
		"coarse_x":    31,
		"coarse_y":    31,
		"nametable_x": 0,
		"nametable_y": 0,
		"fine_y":      0,
		"unused":      0,
	}, r.allAttributes())
	assert.Equal(t, uint16(0b0000001111111111), r.Reg)

	r.SetField("fine_y", 5)
	assert.Equal(t, map[string]uint16{
		"coarse_x":    31,
		"coarse_y":    31,
		"nametable_x": 0,
		"nametable_y": 0,
		"fine_y":      5,
		"unused":      0,
	}, r.allAttributes())
	assert.Equal(t, uint16(0b0101001111111111), r.Reg)

	r.SetField("coarse_y", 9)
	assert.Equal(t, map[string]uint16{
		"coarse_x":    31,
		"coarse_y":    9,
		"nametable_x": 0,
		"nametable_y": 0,
		"fine_y":      5,
		"unused":      0,
	}, r.allAttributes())
	assert.Equal(t, uint16(0b0101000100111111), r.Reg)

	r.SetField("unused", 10)
	assert.Equal(t, map[string]uint16{
		"coarse_x":    31,
		"coarse_y":    9,
		"nametable_x": 0,
		"nametable_y": 0,
		"fine_y":      5,
		"unused":      0,
	}, r.allAttributes())
	assert.Equal(t, uint16(0b0101000100111111), r.Reg)

	r.SetField("unused", 11)
	assert.Equal(t, map[string]uint16{
		"coarse_x":    31,
		"coarse_y":    9,
		"nametable_x": 0,
		"nametable_y": 0,
		"fine_y":      5,
		"unused":      1,
	}, r.allAttributes())
	assert.Equal(t, uint16(0b1101000100111111), r.Reg)

	r.SetField("coarse_y", 32)
	assert.Equal(t, map[string]uint16{
		"coarse_x":    31,
		"coarse_y":    0,
		"nametable_x": 0,
		"nametable_y": 0,
		"fine_y":      5,
		"unused":      1,
	}, r.allAttributes())
	assert.Equal(t, uint16(0b1101000000011111), r.Reg)

}
