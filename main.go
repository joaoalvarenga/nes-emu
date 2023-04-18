package main

import (
	"fmt"
	"github.com/alexflint/go-arg"
	"github.com/go-gl/gl/all-core/gl"
	"github.com/go-gl/glfw/v3.3/glfw"
	"github.com/gordonklaus/portaudio"
	"github.com/nullboundary/glfont"
	"image"
	"log"
	"runtime"
	"strconv"
	"sync"
	"time"
)

const padding = 0

func numToHex(n int, d int) string {
	format := "%0" + strconv.Itoa(d) + "x"
	return fmt.Sprintf(format, n)
}

func createTexture() uint32 {
	var texture uint32
	gl.GenTextures(1, &texture)
	gl.BindTexture(gl.TEXTURE_2D, texture)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.NEAREST)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.NEAREST)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_S, gl.CLAMP_TO_EDGE)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_T, gl.CLAMP_TO_EDGE)
	gl.BindTexture(gl.TEXTURE_2D, 0)
	return texture
}

func setTexture(im *image.RGBA) {
	size := im.Rect.Size()
	gl.TexImage2D(
		gl.TEXTURE_2D, 0, gl.RGBA, int32(size.X), int32(size.Y),
		0, gl.RGBA, gl.UNSIGNED_BYTE, gl.Ptr(im.Pix))
}

func drawBuffer(window *glfw.Window) {
	w, h := window.GetFramebufferSize()
	s1 := float32(w) / 256
	s2 := float32(h) / 240
	f := float32(1 - padding)
	var x, y float32
	if s1 >= s2 {
		x = f * s2 / s1
		y = f
	} else {
		x = f
		y = f * s1 / s2
	}
	gl.Begin(gl.QUADS)
	gl.TexCoord2f(0, 1)
	gl.Vertex2f(-x, -y)
	gl.TexCoord2f(1, 1)
	gl.Vertex2f(x, -y)
	gl.TexCoord2f(1, 0)
	gl.Vertex2f(x, y)
	gl.TexCoord2f(0, 0)
	gl.Vertex2f(-x, y)
	gl.End()
}

func init() {
	// we need a parallel OS thread to avoid audio stuttering
	runtime.GOMAXPROCS(2)

	// we need to keep OpenGL calls on a single thread
	runtime.LockOSThread()
}

var controllerKeys = map[glfw.Key]uint8{
	glfw.KeyX:     0x80,
	glfw.KeyZ:     0x40,
	glfw.KeyA:     0x20,
	glfw.KeyS:     0x10,
	glfw.KeyUp:    0x08,
	glfw.KeyDown:  0x04,
	glfw.KeyLeft:  0x02,
	glfw.KeyRight: 0x01,
}

type Game struct {
	window        *glfw.Window
	screenTexture uint32
	nes           *Bus
	defaultFont   *glfont.Font
	start         time.Time
	lock          sync.Mutex
}

func (g *Game) keyboardCallback(window *glfw.Window, key glfw.Key, scancode int,
	action glfw.Action, mods glfw.ModifierKey) {
	switch action {
	case glfw.Release:
		value, ok := controllerKeys[key]
		if ok {
			g.nes.controller[0] &= ^value
		}
	case glfw.Press:
		value, ok := controllerKeys[key]
		if ok {
			g.nes.controller[0] |= value
		}
		if key == glfw.KeyR {
			g.nes.reset()
		}
	}
}

func NewGame(nes *Bus, lock sync.Mutex) *Game {
	// initialize glfw
	game := &Game{nes: nes, lock: lock}

	// create window
	glfw.WindowHint(glfw.ContextVersionMajor, 2)
	glfw.WindowHint(glfw.ContextVersionMinor, 1)
	window, err := glfw.CreateWindow(1920, 1080, "NES Emulator", nil, nil)
	if err != nil {
		log.Fatalln(err)
	}
	game.window = window
	game.window.MakeContextCurrent()
	glfw.SwapInterval(1)
	game.window.SetKeyCallback(game.keyboardCallback)

	// initialize gl
	if err := gl.Init(); err != nil {
		log.Fatalln(err)
	}
	gl.Enable(gl.TEXTURE_2D)
	gl.Disable(gl.DEPTH_TEST)
	gl.Disable(gl.LIGHTING)

	game.screenTexture = createTexture()
	game.defaultFont, err = glfont.LoadFont("Minecraft.ttf", int32(52), 1920, 1080)
	if err != nil {
		log.Panicf("LoadFont: %v", err)
	}
	return game
}

func (g *Game) Draw() {
	frameDuration := time.Now().Sub(g.start)
	gl.BindTexture(gl.TEXTURE_2D, g.screenTexture)
	g.lock.Lock()
	setTexture(g.nes.ppu.screenImage)
	g.lock.Unlock()
	drawBuffer(g.window)
	gl.BindTexture(gl.TEXTURE_2D, 0)                                                   //r,g,b,a font color
	g.defaultFont.Printf(0, 100, 1.0, "FPS: %f", 1.0/float64(frameDuration.Seconds())) //x,y,scale,string,printf args
	// Do OpenGL stuff.
	g.window.SwapBuffers()
	glfw.PollEvents()
	g.start = time.Now()
}

var args struct {
	Rom string
}

func main() {
	arg.MustParse(&args)
	var mu sync.Mutex
	cart := NewCartridge(args.Rom)
	cpu := NewCPU()
	ppu := NewPPU(mu)
	apu := NewAPU()
	nes := NewBus(cpu, ppu, apu)
	cpu.connectBus(nes)
	nes.insertCartridge(cart)
	nes.reset()

	err := glfw.Init()
	if err != nil {
		log.Fatalln(err)
	}
	defer glfw.Terminate()
	game := NewGame(nes, mu)
	game.start = time.Now()
	game.defaultFont.SetColor(1.0, 1.0, 1.0, 1.0)

	portaudio.Initialize()
	defer portaudio.Terminate()
	//host, err := portaudio.DefaultHostApi()
	if err != nil {
		panic(err)
	}
	//parameters := portaudio.HighLatencyParameters(nil, host.DefaultOutputDevice)
	//start := time.Now()
	callback := func(out []float32) {
		//dur := time.Now().Sub(start)
		//fmt.Printf("Audio FPS %s\n", dur)
		//
		//start = time.Now()

		output := float32(0)

		for i := range out {
			if i%1 == 0 {
				select {
				case sample := <-nes.AudioSample:
					output = sample
				default:
					output = 0
				}
			}
			out[i] = output
		}
	}
	nes.SetSampleFrequency(uint32(44100))
	stream, err := portaudio.OpenDefaultStream(0, 1, 44100, 0, callback)

	if err != nil {
		panic(err)
	}
	if err := stream.Start(); err != nil {
		panic(err)
	}

	for !game.window.ShouldClose() {
		gl.Clear(gl.COLOR_BUFFER_BIT)

		for true {
			nes.clock()
			if nes.ppu.frameComplete {
				break
			}
		}
		nes.ppu.frameComplete = false
		game.Draw()
	}
	stream.Close()
}
