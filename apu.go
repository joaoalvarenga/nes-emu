package main

type APU struct {
	pulse1Enable      bool
	pulse1Sample      float64
	pulse1Seq         Sequencer
	pulse1osc         OSCPulse
	clockCounter      uint32
	frameClockCounter uint32
	globalTime        float64
}

type Sequencer struct {
	sequence uint32
	timer    uint16
	reload   uint16
	output   uint8
}

type OSCPulse struct {
	frequency float64
	dutycycle float64
	amplitude float64
	pi        float64
	harmonics float64
}

func (o *OSCPulse) sample(t float64) float64 {
	a := float64(0)
	b := float64(0)
	p := o.dutycycle * 2.0 * o.pi

	approxsin := func(t float64) float64 {
		j := t * 0.15915
		j = j - float64(int(j))
		return 20.785 * j * (j - 0.5) * (j - 1.0)
	}
	for n := float64(1); n < o.harmonics; n++ {
		c := n * o.frequency * 2.0 * o.pi * t
		a += -approxsin(c) / n
		b += -approxsin(c-p*n) / n
	}
	return (2.0 * o.amplitude / o.pi) * (a - b)
}

func (s *Sequencer) clock(enable bool, funcManip func(*uint32)) uint8 {
	if enable {
		s.timer--
		if s.timer == 0xFFFF {
			s.timer = s.reload + 1
			funcManip(&(s.sequence))
			s.output = uint8(s.sequence & uint32(0x01))
		}
	}
	return s.output
}

func (a *APU) cpuRead(addr uint16) uint8 {
	return 0x00
}

func (a *APU) cpuWrite(addr uint16, data uint8) {
	switch addr {
	case 0x4000:
		switch (data & 0xC0) >> 6 {
		case 0x00:
			a.pulse1Seq.sequence = 0b00000001
			a.pulse1osc.dutycycle = 0.125
		case 0x01:
			a.pulse1Seq.sequence = 0b00000011
			a.pulse1osc.dutycycle = 0.250
		case 0x02:
			a.pulse1Seq.sequence = 0b00001111
			a.pulse1osc.dutycycle = 0.500
		case 0x03:
			a.pulse1Seq.sequence = 0b11111100
			a.pulse1osc.dutycycle = 0.750
		}
	case 0x4002:
		a.pulse1Seq.reload = (a.pulse1Seq.reload & 0xFF00) | uint16(data)
	case 0x4003:
		a.pulse1Seq.reload = ((uint16(data) & 0x07) << 8) | (a.pulse1Seq.reload & 0x00FF)
		a.pulse1Seq.timer = a.pulse1Seq.reload
	case 0x4015:
		a.pulse1Enable = data&0x01 != 0
	}
}

func (a *APU) clock() {
	quarterFrameClock := false
	halfFrameClock := false
	a.globalTime += 0.3333333333 / 1789773
	if a.clockCounter%6 == 0 {
		a.frameClockCounter++
		if a.frameClockCounter == 3729 {
			quarterFrameClock = true
		}
		if a.frameClockCounter == 7457 {
			quarterFrameClock = false
			halfFrameClock = false
		}
		if a.frameClockCounter == 11186 {
			quarterFrameClock = true
		}
		if a.frameClockCounter == 14916 {
			quarterFrameClock = true
			halfFrameClock = true
			a.frameClockCounter = 0
		}

		if quarterFrameClock {

		}

		if halfFrameClock {

		}

		//a.pulse1Seq.clock(a.pulse1Enable, func(u *uint32) {
		//	*u = ((*u & 0x0001) << 7) | ((*u & 0x00FE) >> 1)
		//})
		//
		//a.pulse1Sample = float64(a.pulse1Seq.output)
		a.pulse1osc.frequency = 1789773.0 / (16.0 * float64(a.pulse1Seq.reload+1))
		a.pulse1Sample = a.pulse1osc.sample(a.globalTime)
	}
	a.clockCounter++
}

func (a *APU) reset() {

}

func (a *APU) getOutputSample() float32 {
	return float32(a.pulse1Sample)
}

func NewAPU() *APU {
	return &APU{
		pulse1Enable: false,
		pulse1Sample: 0,
		pulse1Seq:    Sequencer{},
		pulse1osc: OSCPulse{
			frequency: 0,
			dutycycle: 0,
			amplitude: 1,
			pi:        3.14159,
			harmonics: 20,
		},
	}
}
