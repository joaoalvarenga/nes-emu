package main

import (
	"encoding/json"
	"fmt"
	"github.com/hajimehoshi/ebiten/v2"
	"image/color"
	"io"
	"nes-emu/ppu"
	"os"
)

func CreateStatusRegister() ppu.Register {
	return ppu.CreateRegister(map[string]ppu.Field{
		"unused":          {0, 5},
		"sprite_overflow": {5, 1},
		"sprite_zero_hit": {6, 1},
		"vertical_blank":  {7, 1},
	})
}

func CreateMaskRegister() ppu.Register {
	return ppu.CreateRegister(map[string]ppu.Field{
		"grayscale":              {0, 1},
		"render_background_left": {1, 1},
		"render_sprites_left":    {2, 1},
		"render_background":      {3, 1},
		"render_sprites":         {4, 1},
		"enhance_red":            {5, 1},
		"enhance_green":          {6, 1},
		"enhance_blue":           {7, 1},
	})
}

func CreateControlRegister() ppu.Register {
	return ppu.CreateRegister(map[string]ppu.Field{
		"nametable_x":        {0, 1},
		"nametable_y":        {1, 1},
		"increment_mode":     {2, 1},
		"pattern_sprite":     {3, 1},
		"pattern_background": {4, 1},
		"sprite_size":        {5, 1},
		"slave_mode":         {6, 1},
		"enable_nmi":         {7, 1},
	})
}

func CreateLoopyRegister() ppu.Register {
	return ppu.CreateRegister(map[string]ppu.Field{
		"coarse_x":    {0, 5},
		"coarse_y":    {5, 5},
		"nametable_x": {10, 1},
		"nametable_y": {11, 1},
		"fine_y":      {12, 3},
		"unused":      {15, 1},
	})
}

type PPU struct {
	palScreen       []color.Color
	sprScreen       *ebiten.Image
	sprNameTable    [2]*ebiten.Image
	sprPatternTable [2]*ebiten.Image

	tableName    [2][1024]uint8
	tablePattern [2][4096]uint8
	tablePalette [32]uint8

	status  ppu.Register
	mask    ppu.Register
	control ppu.Register

	vramAddr ppu.Register
	tramAddr ppu.Register
	fineX    uint8

	addressLatch  uint8
	ppuDataBuffer uint8

	scanline      int16
	cycle         int16
	frameComplete bool

	bgNextTileId       uint8
	bgNextTileAttrib   uint8
	bgNextTileLsb      uint8
	bgNextTileMsb      uint8
	bgShifterPatternLo uint16
	bgShifterPatternHi uint16
	bgShifterAttribLo  uint16
	bgShifterAttribHi  uint16

	cartridge *Cartridge
	nmi       bool
}

func (p *PPU) connectCartridge(cartridge *Cartridge) {
	p.cartridge = cartridge
}

func loadPalette() []color.Color {
	jsonFile, err := os.Open("palette.json")
	// if we os.Open returns an error then handle it
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println("Successfully Opened palette.json")
	// defer the closing of our jsonFile so that we can parse it later on
	defer jsonFile.Close()

	byteValue, _ := io.ReadAll(jsonFile)

	var result [][3]uint8
	errJson := json.Unmarshal(byteValue, &result)
	if errJson != nil {
		panic(errJson)
	}

	palette := make([]color.Color, 64)
	for i := 0; i < len(result); i++ {
		palette[i] = color.RGBA{R: result[i][0], G: result[i][1], B: result[i][2], A: 0xFF}
	}
	return palette

}

func (p *PPU) cpuRead(addr uint16, readOnly bool) uint8 {
	data := uint8(0)
	if readOnly {
		switch addr {
		case 0x0000:
			data = uint8(p.control.Reg)
		case 0x0001:
			data = uint8(p.mask.Reg)
		case 0x0002:
			data = uint8(p.status.Reg)
		}
		return data
	}

	switch addr {
	case 0x0002:
		data = (uint8(p.status.Reg) & 0xE0) | (uint8(p.ppuDataBuffer) & 0x1F)
		p.status.SetField("vertical_blank", 0)
		p.addressLatch = 0
	case 0x0007:
		data = p.ppuDataBuffer
		p.ppuDataBuffer = p.ppuRead(p.vramAddr.Reg, false)

		if p.vramAddr.Reg >= 0x3F00 {
			data = p.ppuDataBuffer
		}

		if p.control.GetField("increment_mode") != 0 {
			p.vramAddr.Reg += 32
		} else {
			p.vramAddr.Reg += 1
		}
	}
	return data
}

func (p *PPU) cpuWrite(addr uint16, data uint8) {
	switch addr {
	case 0x0000:
		p.control.Reg = uint16(data)
		p.tramAddr.SetField("nametable_x", p.control.GetField("nametable_x"))
		p.tramAddr.SetField("nametable_y", p.control.GetField("nametable_y"))
	case 0x0001:
		p.mask.Reg = uint16(data)
	case 0x0005:
		if p.addressLatch == 0 {
			p.fineX = data & 0x07
			p.tramAddr.SetField("coarse_x", uint16(data)>>3)
			p.addressLatch = 1
		} else {
			p.tramAddr.SetField("fine_y", uint16(data)&0x07)
			p.tramAddr.SetField("coarse_y", uint16(data)>>3)
			p.addressLatch = 0
		}
	case 0x0006:
		if p.addressLatch == 0 {
			p.tramAddr.Reg = ((uint16(data) & 0x3F) << 8) | (p.tramAddr.Reg & 0x00FF)
			p.addressLatch = 1
		} else {
			p.tramAddr.Reg = (p.tramAddr.Reg & 0xFF00) | uint16(data)
			p.vramAddr.Reg = p.tramAddr.Reg
			p.addressLatch = 0
		}
	case 0x0007:
		p.ppuWrite(p.vramAddr.Reg, data)
		increment := uint16(1)
		if p.control.GetField("increment_mode") != 0 {
			increment = 32
		}
		p.vramAddr.Reg += increment
	}
}

func (p *PPU) ppuRead(addr uint16, readOnly bool) uint8 {
	data := uint8(0)
	addr &= 0x3FFF
	if p.cartridge.ppuRead(addr, &data) {
		return data
	}

	if addr >= 0x0000 && addr <= 0x1FFF {
		return p.tablePattern[(addr&0x1000)>>12][addr&0x0FFF]
	}

	if addr >= 0x2000 && addr <= 0x3EFF {
		addr &= 0x0FFF
		if p.cartridge.mirror == VERTICAL {
			// Vertical
			if addr >= 0x0000 && addr <= 0x03FF {
				data = p.tableName[0][addr&0x03FF]
			}
			if addr >= 0x0400 && addr <= 0x07FF {
				data = p.tableName[1][addr&0x03FF]
			}
			if addr >= 0x0800 && addr <= 0x0BFF {
				data = p.tableName[0][addr&0x03FF]
			}
			if addr >= 0x0C00 && addr <= 0x0FFF {
				data = p.tableName[1][addr&0x03FF]
			}
			return data
		}

		if p.cartridge.mirror == HORIZONTAL {
			// Horizontal
			if addr >= 0x0000 && addr <= 0x03FF {
				data = p.tableName[0][addr&0x03FF]
			}
			if addr >= 0x0400 && addr <= 0x07FF {
				data = p.tableName[0][addr&0x03FF]
			}
			if addr >= 0x0800 && addr <= 0x0BFF {
				data = p.tableName[1][addr&0x03FF]
			}
			if addr >= 0x0C00 && addr <= 0x0FFF {
				data = p.tableName[1][addr&0x03FF]
			}
			return data
		}
	}

	if addr >= 0x3F00 && addr <= 0x3FFF {
		addr &= 0x001F
		if addr == 0x0010 {
			addr = 0x0000
		}
		if addr == 0x0014 {
			addr = 0x0004
		}
		if addr == 0x0018 {
			addr = 0x0008
		}
		if addr == 0x001C {
			addr = 0x000C
		}
		mask := uint8(0x3F)
		if p.mask.GetField("grayscale") != 0 {
			mask = 0x30
		}
		data = p.tablePalette[addr] & mask
		return data
	}
	return data
}

func (p *PPU) ppuWrite(addr uint16, data uint8) {
	addr &= 0x3FFF
	if p.cartridge.ppuWrite(addr, data) {
		return
	}
	if addr >= 0x0000 && addr <= 0x1FFF {
		p.tablePattern[(addr&0x1000)>>12][addr&0x0FFF] = data
		return
	}
	if addr >= 0x2000 && addr <= 0x3EFF {
		addr &= 0x0FFF
		if p.cartridge.mirror == VERTICAL {
			// Vertical
			if addr >= 0x0000 && addr <= 0x03FF {
				p.tableName[0][addr&0x03FF] = data
			}
			if addr >= 0x0400 && addr <= 0x07FF {
				p.tableName[1][addr&0x03FF] = data
			}
			if addr >= 0x0800 && addr <= 0x0BFF {
				p.tableName[0][addr&0x03FF] = data
			}
			if addr >= 0x0C00 && addr <= 0x0FFF {
				p.tableName[1][addr&0x03FF] = data
			}
			return
		}
		if p.cartridge.mirror == HORIZONTAL {
			// Horizontal
			if addr >= 0x0000 && addr <= 0x03FF {
				p.tableName[0][addr&0x03FF] = data
			}
			if addr >= 0x0400 && addr <= 0x07FF {
				p.tableName[0][addr&0x03FF] = data
			}
			if addr >= 0x0800 && addr <= 0x0BFF {
				p.tableName[1][addr&0x03FF] = data
			}
			if addr >= 0x0C00 && addr <= 0x0FFF {
				p.tableName[1][addr&0x03FF] = data
			}
			return
		}
	}
	if addr >= 0x3F00 && addr <= 0x3FFF {
		addr &= 0x001F
		if addr == 0x0010 {
			addr = 0x0000
		}
		if addr == 0x0014 {
			addr = 0x0004
		}
		if addr == 0x0018 {
			addr = 0x0008
		}
		if addr == 0x001C {
			addr = 0x000C
		}
		p.tablePalette[addr] = data
	}
}

func (p *PPU) getColourFromPaletteRam(palette uint8, pixel uint8) color.Color {
	paletteCode := uint16(palette) << 2
	return p.palScreen[p.ppuRead(0x3F00+uint16(paletteCode)+uint16(pixel), false)&0x3F]
}

func (p *PPU) getPatternTable(i uint8, palette uint8) ebiten.Image {
	for tileY := uint16(0); tileY < 16; tileY++ {
		for tileX := uint16(0); tileX < 16; tileX++ {
			offset := tileY*256 + tileX*16
			for row := uint16(0); row < 8; row++ {
				tileLsb := p.ppuRead(uint16(i)*0x1000+offset+row, false)
				tileMsb := p.ppuRead(uint16(i)*0x1000+offset+row+0x0008, false)

				for col := uint16(0); col < 8; col++ {
					pixel := (tileLsb & 0x01) + (tileMsb & 0x01)
					tileLsb >>= 1
					tileMsb >>= 1
					p.sprPatternTable[i].Set(
						int(tileX*8+(7-col)),
						int(tileY*8+row),
						p.getColourFromPaletteRam(palette, pixel),
					)
				}
			}
		}
	}
	return *(p.sprPatternTable[i])
}

func NewPPU() *PPU {
	mPPU := &PPU{
		palScreen: loadPalette(),
		sprScreen: ebiten.NewImage(256, 240),
		sprNameTable: [2]*ebiten.Image{
			ebiten.NewImage(256, 240),
			ebiten.NewImage(256, 240),
		},
		sprPatternTable: [2]*ebiten.Image{
			ebiten.NewImage(128, 128),
			ebiten.NewImage(128, 128),
		},
		control:            CreateControlRegister(),
		mask:               CreateMaskRegister(),
		status:             CreateStatusRegister(),
		vramAddr:           CreateLoopyRegister(),
		tramAddr:           CreateLoopyRegister(),
		frameComplete:      false,
		cycle:              0,
		scanline:           0,
		fineX:              0,
		bgNextTileId:       0,
		bgNextTileAttrib:   0,
		bgNextTileLsb:      0,
		bgNextTileMsb:      0,
		bgShifterPatternLo: 0,
		bgShifterPatternHi: 0,
		bgShifterAttribLo:  0,
		bgShifterAttribHi:  0,
		addressLatch:       0,
		ppuDataBuffer:      0,
		nmi:                false,
	}
	return mPPU
}

func (p *PPU) clock() {
	IncrementScrollX := func() {
		if p.mask.GetField("render_background") != 0 || p.mask.GetField("render_sprites") != 0 {
			if p.vramAddr.GetField("coarse_x") == 31 {
				p.vramAddr.SetField("coarse_x", 0)
				invertedNameTableX := ^(p.vramAddr.GetField("nametable_x"))
				p.vramAddr.SetField("nametable_x", invertedNameTableX)
				return
			}
			p.vramAddr.SetField("coarse_x", p.vramAddr.GetField("coarse_x")+1)
		}
	}

	IncrementScrollY := func() {
		if p.mask.GetField("render_background") != 0 || p.mask.GetField("render_sprites") != 0 {
			if p.vramAddr.GetField("fine_y") < 7 {
				p.vramAddr.SetField("fine_y", p.vramAddr.GetField("fine_y")+1)
			} else {
				p.vramAddr.SetField("fine_y", 0)

				// Check if we need to swap vertical nametable targets
				if p.vramAddr.GetField("coarse_y") == 29 {
					// We do, so reset coarse y offset
					p.vramAddr.SetField("coarse_y", 0)

					invertedNameTableY := ^(p.vramAddr.GetField("nametable_y"))
					p.vramAddr.SetField("nametable_y", invertedNameTableY)
				} else if p.vramAddr.GetField("coarse_y") == 31 {
					// In case the pointer is in the attribute memory, we
					// just wrap around the current nametable
					p.vramAddr.SetField("coarse_y", 0)
				} else {
					p.vramAddr.SetField("coarse_y", p.vramAddr.GetField("coarse_y")+1)
				}
			}
		}
	}

	TransferAddressX := func() {
		if p.mask.GetField("render_background") != 0 || p.mask.GetField("render_sprites") != 0 {
			p.vramAddr.SetField("nametable_x", p.tramAddr.GetField("nametable_x"))
			p.vramAddr.SetField("coarse_x", p.tramAddr.GetField("coarse_x"))
		}
	}

	TransferAddressY := func() {
		if p.mask.GetField("render_background") != 0 || p.mask.GetField("render_sprites") != 0 {
			p.vramAddr.SetField("fine_y", p.tramAddr.GetField("fine_y"))
			p.vramAddr.SetField("nametable_y", p.tramAddr.GetField("nametable_y"))
			p.vramAddr.SetField("coarse_y", p.tramAddr.GetField("coarse_y"))
		}
	}

	LoadBackgroundShifters := func() {
		p.bgShifterPatternLo = (p.bgShifterPatternLo & 0xFF00) | uint16(p.bgNextTileLsb)
		p.bgShifterPatternHi = (p.bgShifterPatternHi & 0xFF00) | uint16(p.bgNextTileMsb)

		acc := uint16(0x00)
		if p.bgNextTileAttrib&0b01 != 0 {
			acc = 0xFF
		}
		p.bgShifterAttribLo = (p.bgShifterAttribLo & 0xFF00) | acc
		acc = uint16(0x00)
		if p.bgNextTileAttrib&0b10 != 0 {
			acc = 0xFF
		}
		p.bgShifterAttribHi = (p.bgShifterAttribHi & 0xFF00) | acc
	}

	UpdateShifters := func() {
		if p.mask.GetField("render_background") != 0 {
			p.bgShifterPatternLo <<= 1
			p.bgShifterPatternHi <<= 1
			p.bgShifterAttribLo <<= 1
			p.bgShifterAttribHi <<= 1
		}
	}

	if p.scanline >= -1 && p.scanline < 240 {
		if p.scanline == 0 && p.cycle == 0 {
			p.cycle = 1
		}

		if p.scanline == -1 && p.cycle == 1 {
			p.status.SetField("vertical_blank", 0)
		}

		if (p.cycle >= 2 && p.cycle < 258) || (p.cycle >= 321 && p.cycle < 338) {
			UpdateShifters()
			switch (p.cycle - 1) % 8 {
			case 0:
				LoadBackgroundShifters()
				p.bgNextTileId = p.ppuRead(0x2000|(p.vramAddr.Reg&0x0FFF), false)
			case 2:
				p.bgNextTileAttrib = p.ppuRead(0x23C0|(p.vramAddr.GetField("nametable_y")<<11)|(p.vramAddr.GetField("nametable_x")<<10)|((p.vramAddr.GetField("coarse_y")>>2)<<3)|(p.vramAddr.GetField("coarse_x")>>2), false)
				if p.vramAddr.GetField("coarse_y")&0x02 != 0 {
					p.bgNextTileAttrib >>= 4
				}
				if p.vramAddr.GetField("coarse_x")&0x02 != 0 {
					p.bgNextTileAttrib >>= 2
				}
				p.bgNextTileAttrib &= 0x03
			case 4:
				p.bgNextTileLsb = p.ppuRead((p.control.GetField("pattern_background")<<12)+(uint16(p.bgNextTileId)<<4)+(p.vramAddr.GetField("fine_y")), false)
			case 6:
				p.bgNextTileMsb = p.ppuRead((p.control.GetField("pattern_background")<<12)+(uint16(p.bgNextTileId)<<4)+(p.vramAddr.GetField("fine_y")+8), false)
			case 7:
				IncrementScrollX()
			}
		}
		if p.cycle == 256 {
			IncrementScrollY()
		}
		if p.cycle == 257 {
			LoadBackgroundShifters()
			TransferAddressX()
		}
		if p.cycle == 338 || p.cycle == 340 {
			p.bgNextTileId = p.ppuRead(0x2000|(p.vramAddr.Reg&0x0FFF), false)
		}
		if p.scanline == -1 && p.cycle >= 280 && p.cycle < 305 {
			TransferAddressY()
		}
	}

	if p.scanline >= 241 && p.scanline < 261 {
		if p.scanline == 241 && p.cycle == 1 {
			p.status.SetField("vertical_blank", 1)
			if p.control.GetField("enable_nmi") != 0 {
				p.nmi = true
			}
		}
	}

	bgPixel := uint8(0)
	bgPalette := uint8(0)
	if p.mask.GetField("render_background") != 0 {
		bitMux := uint16(0x8000 >> p.fineX)
		p0Pixel := uint8(0)
		if p.bgShifterPatternLo&bitMux > 0 {
			p0Pixel = 1
		}
		p1Pixel := uint8(0)
		if p.bgShifterPatternHi&bitMux > 0 {
			p1Pixel = 1
		}

		bgPixel = (p1Pixel << 1) | p0Pixel

		bgPal0 := uint8(0)
		if p.bgShifterAttribLo&bitMux > 0 {
			bgPal0 = 1
		}
		bgPal1 := uint8(0)
		if p.bgShifterAttribHi&bitMux > 0 {
			bgPal1 = 1
		}

		bgPalette = (bgPal1 << 1) | bgPal0
	}

	pixelColor := p.getColourFromPaletteRam(bgPalette, bgPixel)
	p.sprScreen.Set(int(p.cycle)-1, int(p.scanline), pixelColor)

	p.cycle++
	if p.cycle >= 341 {
		p.cycle = 0
		p.scanline++
		if p.scanline >= 261 {
			p.scanline = -1
			p.frameComplete = true
		}
	}
}

func (p *PPU) reset() {
	p.fineX = 0
	p.addressLatch = 0
	p.ppuDataBuffer = 0
	p.scanline = 0
	p.cycle = 0
	p.bgNextTileId = 0
	p.bgNextTileAttrib = 0
	p.bgNextTileLsb = 0
	p.bgNextTileMsb = 0
	p.bgShifterPatternLo = 0x0000
	p.bgShifterPatternHi = 0x0000
	p.bgShifterAttribLo = 0x0000
	p.bgShifterAttribHi = 0x0000
	p.status.Reg = 0x00
	p.mask.Reg = 0x00
	p.control.Reg = 0x00
	p.vramAddr.Reg = 0x0000
	p.tramAddr.Reg = 0x0000
}
