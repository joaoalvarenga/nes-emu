package main

import (
	"fmt"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/examples/resources/fonts"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text"
	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
	"image/color"
	"log"
	"strconv"
	"time"
)

const (
	screenWidth  = 1280
	screenHeight = 900
)

type Game struct {
	nes             *Bus
	keys            []ebiten.Key
	mapAsm          map[uint16]DissambledInstruction
	defaultFont     font.Face
	emulationRun    bool
	stepping        bool
	selectedPalette uint8
	gameScreen      *ebiten.Image
}

var controllerKeys = map[ebiten.Key]uint8{
	ebiten.KeyX:     0x80,
	ebiten.KeyZ:     0x40,
	ebiten.KeyA:     0x20,
	ebiten.KeyS:     0x10,
	ebiten.KeyUp:    0x08,
	ebiten.KeyDown:  0x04,
	ebiten.KeyLeft:  0x02,
	ebiten.KeyRight: 0x01,
}

func (g *Game) Update() error {
	//start := time.Now()
	pressedKeys := inpututil.AppendPressedKeys(nil)
	g.nes.controller[0] = 0x00
	for _, p := range pressedKeys {
		value, ok := controllerKeys[p]
		if !ok {
			continue
		}
		g.nes.controller[0] |= value
	}
	if g.stepping {
		for true {
			g.nes.clock()
			if g.nes.cpu.isComplete() {
				break
			}
		}
		for true {
			g.nes.clock()
			if !g.nes.cpu.isComplete() {
				break
			}
		}
	}
	if g.emulationRun {
		//start := time.Now()
		cpuDuration := time.Duration(0)
		ppuDuration := time.Duration(0)
		for true {
			c, p := g.nes.clock()
			cpuDuration += c
			ppuDuration += p
			if g.nes.ppu.frameComplete {
				break
			}
		}
		g.nes.ppu.frameComplete = false
		//elapsed := time.Now().Sub(start)
		//fmt.Printf("CPU time = %s\n", cpuDuration)
		//fmt.Printf("PPU time = %s\n", ppuDuration)
	}
	//if inpututil.IsKeyJustPressed(ebiten.KeyP) {
	//	g.selectedPalette++
	//	g.selectedPalette &= 0x07
	//}
	if inpututil.IsKeyJustPressed(ebiten.KeySpace) {
		g.emulationRun = !g.emulationRun
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyR) {
		g.nes.reset()
	}
	//if inpututil.IsKeyJustPressed(ebiten.KeyV) {
	//	g.stepping = true
	//}
	//if inpututil.IsKeyJustReleased(ebiten.KeyV) {
	//	g.stepping = false
	//}
	//if inpututil.IsKeyJustPressed(ebiten.KeyC) {
	//	for true {
	//		g.nes.clock()
	//		if g.nes.cpu.isComplete() {
	//			break
	//		}
	//	}
	//
	//	for true {
	//		g.nes.clock()
	//		if !g.nes.cpu.isComplete() {
	//			break
	//		}
	//	}
	//}
	//if inpututil.IsKeyJustPressed(ebiten.KeyF) {
	//	for true {
	//		g.nes.clock()
	//		if g.nes.ppu.frameComplete {
	//			break
	//		}
	//	}
	//
	//	//for true {
	//	//	g.nes.clock()
	//	//	if g.nes.cpu.isComplete() {
	//	//		break
	//	//	}
	//	//}
	//
	//	g.nes.ppu.frameComplete = false
	//}
	//elapsed := time.Now().Sub(start)
	//fmt.Printf("Update cycle = %s\n", elapsed)

	return nil
}

func numToHex(n int, d int) string {
	format := "%0" + strconv.Itoa(d) + "x"
	return fmt.Sprintf(format, n)
}

func (g *Game) getDefaultFont() font.Face {
	if g.defaultFont != nil {
		return g.defaultFont
	}
	tt, err := opentype.Parse(fonts.MPlus1pRegular_ttf)
	if err != nil {
		log.Fatal(err)
	}
	const dpi = 72 * 2
	mplusNormalFont, err := opentype.NewFace(tt, &opentype.FaceOptions{
		Size:    8,
		DPI:     dpi,
		Hinting: font.HintingNone,
	})

	if err != nil {
		log.Fatal(err)
	}
	g.defaultFont = mplusNormalFont
	return g.defaultFont
}

func (g *Game) DrawCode(screen *ebiten.Image, x int, y int, nLines int, offset int) {
	itA, ok := g.mapAsm[g.nes.cpu.pc]
	if !ok {
		return
	}
	lineSize := 24
	lineY := y + (nLines>>1)*lineSize

	text.Draw(screen, itA.instruction, g.getDefaultFont(), x, lineY, color.RGBA{R: 0x00, G: 0xFF, B: 0xFF, A: 0xFF})
	for lineY < (y + (nLines * lineSize)) {
		lineY += lineSize
		itA, ok = g.mapAsm[itA.nextAddr]
		if !ok {
			break
		}
		text.Draw(screen, itA.instruction, g.getDefaultFont(), x, lineY, color.RGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 0xFF})
	}

	itA = g.mapAsm[g.nes.cpu.pc]
	lineY = y + (nLines>>1)*lineSize
	for lineY > y {
		lineY -= lineSize
		itA, ok = g.mapAsm[itA.previousAddr]
		if !ok {
			break
		}
		text.Draw(screen, itA.instruction, g.getDefaultFont(), x, lineY, color.RGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 0xFF})
	}
}

func (g *Game) DrawRam(screen *ebiten.Image, x int, y int, nAddr uint16, nRows int, nColumns int) {
	for row := 0; row < nRows; row++ {
		sOffset := fmt.Sprintf("%s:", numToHex(int(nAddr), 4))
		for col := 0; col < nColumns; col++ {
			sOffset = fmt.Sprintf("%s %s", sOffset, numToHex(int(g.nes.cpuRead(nAddr, true)), 2))
			nAddr += 1
		}
		ebitenutil.DebugPrintAt(screen, sOffset, x, y)
		y += 16
	}
}

var (
	WHITE = color.RGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 0xFF}
	GREEN = color.RGBA{G: 0xFF, A: 0xFF}
	RED   = color.RGBA{R: 0xFF, A: 0xFF}
)

func (g *Game) DrawString(screen *ebiten.Image, x int, y int, str string, clr color.RGBA) {
	text.Draw(screen, str, g.getDefaultFont(), x, y, clr)
}

func (g *Game) DrawCpu(screen *ebiten.Image, x int, y int) {
	g.DrawString(screen, x, y, "STATUS: ", WHITE)
	titleOffset := 70
	statusOffset := 10
	statusColor := RED
	if g.nes.cpu.getFlag(N) == 1 {
		statusColor = GREEN
	}
	g.DrawString(screen, x+titleOffset, y, "N", statusColor)
	statusColor = RED

	if g.nes.cpu.getFlag(V) == 1 {
		statusColor = GREEN
	}
	g.DrawString(screen, x+titleOffset+(statusOffset*1), y, "V", statusColor)
	statusColor = RED

	if g.nes.cpu.getFlag(U) == 1 {
		statusColor = GREEN
	}
	g.DrawString(screen, x+titleOffset+(statusOffset*2), y, "U", statusColor)
	statusColor = RED

	if g.nes.cpu.getFlag(B) == 1 {
		statusColor = GREEN
	}
	g.DrawString(screen, x+titleOffset+(statusOffset*3), y, "B", statusColor)
	statusColor = RED

	if g.nes.cpu.getFlag(D) == 1 {
		statusColor = GREEN
	}
	g.DrawString(screen, x+titleOffset+(statusOffset*4), y, "D", statusColor)
	statusColor = RED

	if g.nes.cpu.getFlag(I) == 1 {
		statusColor = GREEN
	}
	g.DrawString(screen, x+titleOffset+(statusOffset*5), y, "I", statusColor)
	statusColor = RED

	if g.nes.cpu.getFlag(Z) == 1 {
		statusColor = GREEN
	}
	g.DrawString(screen, x+titleOffset+(statusOffset*6), y, "Z", statusColor)
	statusColor = RED

	if g.nes.cpu.getFlag(C) == 1 {
		statusColor = GREEN
	}
	g.DrawString(screen, x+titleOffset+(statusOffset*7), y, "C", statusColor)
	statusColor = RED

	lineSize := 24
	g.DrawString(screen, x, y+lineSize, fmt.Sprintf("PC: $%s", numToHex(int(g.nes.cpu.pc), 4)), WHITE)
	g.DrawString(screen, x, y+(lineSize*2), fmt.Sprintf("A: $%s", numToHex(int(g.nes.cpu.accumulator), 2)), WHITE)
	g.DrawString(screen, x, y+(lineSize*3), fmt.Sprintf("X: $%s", numToHex(int(g.nes.cpu.xRegister), 2)), WHITE)
	g.DrawString(screen, x, y+(lineSize*4), fmt.Sprintf("Y: $%s", numToHex(int(g.nes.cpu.yRegister), 2)), WHITE)
	g.DrawString(screen, x, y+(lineSize*5), fmt.Sprintf("Stack P: $%s", numToHex(int(g.nes.cpu.stkp), 4)), WHITE)
	g.DrawString(screen, x, y+(lineSize*6), fmt.Sprintf("Cycles: %d Scanlines: %d", g.nes.ppu.cycle, g.nes.ppu.scanline), WHITE)
}

func (g *Game) DrawPalette(screen *ebiten.Image, x, y float64, i uint8, palette uint8) {
	op := &ebiten.DrawImageOptions{}

	op.GeoM.Scale(2, 2)
	op.GeoM.Translate(x, y)
	img := g.nes.ppu.getPatternTable(i, palette)
	screen.DrawImage(&img, op)
}

func (g *Game) DrawOAM(screen *ebiten.Image, x, y int) {
	for i := 0; i < 26; i++ {
		oam := &g.nes.ppu.oam[i]
		s := fmt.Sprintf("%s: (%d, %d) ID: %s AT: %s",
			numToHex(i, 2), oam.x, oam.y, numToHex(int(oam.id), 2), numToHex(int(oam.attribute), 2))
		g.DrawString(screen, x, y+(i*24), s, WHITE)
	}
}

func (g *Game) Draw(screen *ebiten.Image) {
	// start := time.Now()
	//g.DrawString(screen, screenWidth-250, 200, fmt.Sprintf("C: %08b", g.nes.controller), WHITE)
	//g.DrawRam(screen, 0, 0, 0x0000, 32, 16)
	//g.DrawCode(screen, screenWidth-250, 200, 26, -10)
	//g.DrawOAM(screen, screenWidth-250, 250)

	//g.DrawCpu(screen, screenWidth-250, 20)

	//nSwatchSize := 6
	//for p := uint8(0); p < 8; p++ { // For each palette
	//	for s := uint8(0); s < 4; s++ { // For each index
	//		vector.DrawFilledRect(screen,
	//			float32(5+int(p)*(nSwatchSize*5)+int(s)*nSwatchSize),
	//			screenHeight-270, float32(nSwatchSize),
	//			float32(nSwatchSize),
	//			g.nes.ppu.getColourFromPaletteRam(p, s),
	//			false)
	//	}
	//}

	// Draw selection reticule around selected palette
	//vector.StrokeRect(screen,
	//	float32(5+int(g.selectedPalette)*(nSwatchSize*5)-1),
	//	screenHeight-270,
	//	float32(nSwatchSize*4)+1,
	//	float32(nSwatchSize),
	//	1,
	//	WHITE,
	//	false)
	//DrawRect(516 + nSelectedPalette * (nSwatchSize * 5) - 1, 339, (nSwatchSize * 4), nSwatchSize, olc::WHITE);

	//g.DrawPalette(screen, 5, screenHeight-260, 0, g.selectedPalette)
	//g.DrawPalette(screen, 260+5, screenHeight-260, 1, g.selectedPalette)

	for y := 0; y < 240; y++ {
		for x := 0; x < 256; x++ {
			screen.Set(x, y, g.nes.ppu.sprScreen[y][x])
			//g.gameScreen.Set(x, y, g.nes.ppu.sprScreen[y][x])
		}
	}
	//op := ebiten.DrawImageOptions{}
	//op.GeoM.Translate(200, 0)
	//op.GeoM.Scale(2, 2)

	//screen.DrawImage(g.gameScreen, &op)
	//elapsed := time.Now().Sub(start)
	//fmt.Printf("Draw cycle = %s\n", elapsed)
}

func (g *Game) Layout(outsideWidth int, outsideHeight int) (int, int) {
	return screenWidth, screenHeight
}

func main() {

	cart := NewCartridge("/home/joao/PycharmProjects/nes-emu/smb.nes")
	cpu := NewCPU()
	ppu := NewPPU()
	bus := NewBus(cpu, ppu)
	cpu.connectBus(bus)
	bus.insertCartridge(cart)

	bus.reset()

	ebiten.SetWindowSize(screenWidth, screenHeight)
	ebiten.SetWindowTitle("Filter")
	ebiten.SetVsyncEnabled(false)
	ebiten.SetTPS(ebiten.SyncWithFPS)
	if err := ebiten.RunGame(&Game{
		nes:        bus,
		mapAsm:     cpu.disassemble(0x0000, 0xFFFF),
		gameScreen: ebiten.NewImage(256, 240),
	}); err != nil {
		log.Fatal(err)
	}

}
