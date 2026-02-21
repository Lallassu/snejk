package game

import (
	"io"
	"math"
	"sync/atomic"
	"time"

	"github.com/hajimehoshi/oto/v2"
)

const (
	SampleRate   = 44100
	ChannelCount = 2
	BitDepth     = 0 // 32-bit float (oto.FormatFloat32LE)
)

// SoundKind identifies different sound effects.
type SoundKind int

const (
	SoundEat SoundKind = iota
	SoundEatInfected
	SoundExplosion
	SoundBonus
	SoundFire
	SoundHurt
	SoundLevelUp
	SoundGameOver
	SoundMenuSelect
	SoundSplatter
	SoundPoliceSiren
	SoundHelicopter
	SoundGunshot
	SoundScream
	SoundChopperLoop
)

// AudioSystem manages procedural sound effects.
type AudioSystem struct {
	ctx         *oto.Context
	ready       chan struct{}
	musicPlayer oto.Player
}

var globalAudio *AudioSystem

// activeExplosions limits simultaneous explosion sounds to avoid speaker clipping.
var activeExplosions int32
var splatterVariantCounter uint64
var explosionVariantCounter uint64
var policeSirenVariantCounter uint64

// InitAudio initializes the audio system.
func InitAudio() error {
	ctx, ready, err := oto.NewContext(SampleRate, ChannelCount, BitDepth)
	if err != nil {
		return err
	}
	globalAudio = &AudioSystem{ctx: ctx, ready: ready}
	return nil
}

// PlaySound plays a procedurally generated sound effect.
func PlaySound(kind SoundKind) {
	playSoundWithGain(kind, 1.0)
}

func PlaySoundWithGain(kind SoundKind, gain float64) {
	playSoundWithGain(kind, gain)
}

func PlayPoliceSirenSpatial(gain, doppler, pan float64) {
	if globalAudio == nil || gain <= 0 {
		return
	}
	select {
	case <-globalAudio.ready:
	default:
		return
	}
	samples := genPoliceSirenSpatial(doppler, pan)
	if len(samples) == 0 {
		return
	}
	go func() {
		reader := &soundReader{data: samples}
		player := globalAudio.ctx.NewPlayer(reader)
		player.SetVolume(sfxVolume * clampF(gain, 0, 1))
		player.Play()
		for player.IsPlaying() {
			time.Sleep(10 * time.Millisecond)
		}
		player.Close()
	}()
}

func playSoundWithGain(kind SoundKind, gain float64) {
	if globalAudio == nil {
		return
	}
	if gain <= 0 {
		return
	}
	select {
	case <-globalAudio.ready:
	default:
		return
	}
	// Limit simultaneous explosions to 2 — more causes speaker clipping.
	if kind == SoundExplosion {
		if atomic.LoadInt32(&activeExplosions) >= 2 {
			return
		}
		atomic.AddInt32(&activeExplosions, 1)
	}
	samples := generateSound(kind)
	if len(samples) == 0 {
		if kind == SoundExplosion {
			atomic.AddInt32(&activeExplosions, -1)
		}
		return
	}
	go func() {
		if kind == SoundExplosion {
			defer atomic.AddInt32(&activeExplosions, -1)
		}
		reader := &soundReader{data: samples}
		player := globalAudio.ctx.NewPlayer(reader)
		player.SetVolume(sfxVolume * clampF(gain, 0, 1))
		player.Play()
		for player.IsPlaying() {
			time.Sleep(10 * time.Millisecond)
		}
		player.Close()
	}()
}

// PlayExplosionSound plays an explosion whose timbre scales with blast magnitude.
// magnitude is expected to be in world-pixel radius units used by ExplodeAt.
func PlayExplosionSound(magnitude float64) {
	if globalAudio == nil {
		return
	}
	select {
	case <-globalAudio.ready:
	default:
		return
	}
	if atomic.LoadInt32(&activeExplosions) >= 2 {
		return
	}
	atomic.AddInt32(&activeExplosions, 1)
	samples := genExplosionScaled(magnitude)
	if len(samples) == 0 {
		atomic.AddInt32(&activeExplosions, -1)
		return
	}
	go func() {
		defer atomic.AddInt32(&activeExplosions, -1)
		reader := &soundReader{data: samples}
		player := globalAudio.ctx.NewPlayer(reader)
		player.SetVolume(sfxVolume)
		player.Play()
		for player.IsPlaying() {
			time.Sleep(10 * time.Millisecond)
		}
		player.Close()
	}()
}

type soundReader struct {
	data []byte
	pos  int
}

func (r *soundReader) Read(p []byte) (int, error) {
	if r.pos >= len(r.data) {
		return 0, io.EOF
	}
	n := copy(p, r.data[r.pos:])
	r.pos += n
	return n, nil
}

// putStereoF32 writes a [-1,1] sample as float32 LE to both stereo channels at frame i.
func putStereoF32(buf []byte, i int, sample float64) {
	v := math.Float32bits(float32(sample))
	buf[i*8] = byte(v)
	buf[i*8+1] = byte(v >> 8)
	buf[i*8+2] = byte(v >> 16)
	buf[i*8+3] = byte(v >> 24)
	buf[i*8+4] = byte(v)
	buf[i*8+5] = byte(v >> 8)
	buf[i*8+6] = byte(v >> 16)
	buf[i*8+7] = byte(v >> 24)
}

// putStereoF32LR writes independent left/right samples in [-1,1].
func putStereoF32LR(buf []byte, i int, left, right float64) {
	lv := math.Float32bits(float32(left))
	rv := math.Float32bits(float32(right))
	buf[i*8] = byte(lv)
	buf[i*8+1] = byte(lv >> 8)
	buf[i*8+2] = byte(lv >> 16)
	buf[i*8+3] = byte(lv >> 24)
	buf[i*8+4] = byte(rv)
	buf[i*8+5] = byte(rv >> 8)
	buf[i*8+6] = byte(rv >> 16)
	buf[i*8+7] = byte(rv >> 24)
}

// softSat applies gentle tanh-like saturation — no harsh clipping.
func softSat(x float64) float64 {
	if x > 1.0 {
		return 1.0 - 0.5/(x)
	}
	if x < -1.0 {
		return -1.0 + 0.5/(-x)
	}
	return x - x*x*x/3.0
}

// adsr returns an envelope at normalized progress [0,1].
// attack/decay/release are fractions of the total duration.
func adsr(progress, attack, decay, sustain, release float64) float64 {
	switch {
	case progress < attack:
		return progress / attack
	case progress < attack+decay:
		return 1.0 - (progress-attack)/decay*(1.0-sustain)
	case progress < 1.0-release:
		return sustain
	default:
		return sustain * (1.0 - (progress-(1.0-release))/release)
	}
}

// fm returns an FM-synthesized sample.
// carrier: base frequency, modRatio: modulator/carrier ratio, modIdx: modulation depth.
func fm(t, carrier, modRatio, modIdx float64) float64 {
	mod := math.Sin(2 * math.Pi * carrier * modRatio * t)
	return math.Sin(2*math.Pi*carrier*t + modIdx*mod)
}

// lcg advances an LCG seed and returns a noise sample in [-1,1].
func lcg(seed *uint64) float64 {
	*seed = *seed*6364136223846793005 + 1442695040888963407
	return float64(int64(*seed>>33)-int64(1<<30)) / float64(1<<30)
}

// makeBuf allocates a stereo float32 buffer for n samples.
func makeBuf(n int) []byte { return make([]byte, n*8) }

// ---- Sound effects -------------------------------------------------------

func generateSound(kind SoundKind) []byte {
	switch kind {
	case SoundEat:
		return genEat()
	case SoundEatInfected:
		return genEatInfected()
	case SoundExplosion:
		return genExplosion()
	case SoundBonus:
		return genBonus()
	case SoundFire:
		return genFire()
	case SoundHurt:
		return genHurt()
	case SoundLevelUp:
		return genLevelUp()
	case SoundGameOver:
		return genGameOver()
	case SoundMenuSelect:
		return genMenuSelect()
	case SoundSplatter:
		return genSplatter()
	case SoundPoliceSiren:
		return genPoliceSiren()
	case SoundHelicopter:
		return genHelicopterSnd(0.5)
	case SoundGunshot:
		return genGunshot()
	case SoundScream:
		return genScream()
	case SoundChopperLoop:
		return genHelicopterSnd(0.8)
	}
	return nil
}

// genEat: snappy FM pop — ascending pitch with bell attack.
func genEat() []byte {
	n := int(0.09 * SampleRate)
	buf := makeBuf(n)
	for i := 0; i < n; i++ {
		t := float64(i) / SampleRate
		p := float64(i) / float64(n)
		env := adsr(p, 0.01, 0.5, 0.0, 0.1)
		freq := 480 + 720*p
		s := fm(t, freq, 2.0, 3.5*env) * env * 0.5
		// Thin harmonic layer for clarity.
		s += math.Sin(2*math.Pi*freq*3*t) * env * 0.06
		putStereoF32(buf, i, softSat(s))
	}
	return buf
}

// genEatInfected: gooey low thud + heavily filtered noise.
func genEatInfected() []byte {
	n := int(0.20 * SampleRate)
	buf := makeBuf(n)
	seed := uint64(11111)
	lp := 0.0
	for i := 0; i < n; i++ {
		t := float64(i) / SampleRate
		p := float64(i) / float64(n)
		env := math.Exp(-p * 7)
		lp = lp*0.88 + lcg(&seed)*0.12 // strong lowpass (~600 Hz)
		thump := fm(t, 75, 0.5, 1.2) * math.Exp(-p*20)
		s := (lp*0.55 + thump*0.55) * env
		putStereoF32(buf, i, softSat(s))
	}
	return buf
}

// genExplosion: phase-correct sub boom + crack + bandpassed body + rumble.
// Kept short (0.50s) so overlapping explosions don't stack into clipping.
func genExplosion() []byte {
	return genExplosionScaled(float64(ExplosionRadius))
}

// genExplosionScaled adapts explosion timbre to blast size:
// larger blasts are deeper, longer, and rumblier; small blasts are snappier.
func genExplosionScaled(magnitude float64) []byte {
	norm := clampF((magnitude-3.0)/27.0, 0, 1)
	dur := 0.26 + 0.64*norm
	n := int(dur * SampleRate)
	buf := makeBuf(n)
	seed := atomic.AddUint64(&explosionVariantCounter, 1) ^
		uint64(time.Now().UnixNano()) ^
		uint64(magnitude*4096)
	lp1, lp2 := 0.0, 0.0 // two lowpasses for bandpass body
	rumLP := 0.0
	subPhase := 0.0
	for i := 0; i < n; i++ {
		p := float64(i) / float64(n)

		// Sub boom: deeper and longer for larger blasts.
		subStart := 155.0 - 65.0*norm
		subEnd := 34.0 - 18.0*norm
		if subEnd < 10 {
			subEnd = 10
		}
		subFreq := subStart * math.Pow(subEnd/subStart, p*(1.6+1.5*norm))
		subPhase += 2 * math.Pi * subFreq / SampleRate
		sub := math.Sin(subPhase) * math.Exp(-p*(7.0-3.8*norm)) * (0.44 + 0.34*norm)

		// Hard transient crack: stronger for small blasts.
		crack := 0.0
		crackWin := 0.038 - 0.020*norm
		if crackWin < 0.010 {
			crackWin = 0.010
		}
		if p < crackWin {
			crack = lcg(&seed) * (1 - p/crackWin) * (0.88 - 0.28*norm)
		}

		// Bandpassed body (~120–2200 Hz).
		raw := lcg(&seed)
		lp1 = lp1*0.76 + raw*0.24   // upper lowpass
		lp2 = lp2*0.975 + raw*0.025 // lower lowpass
		body := (lp1 - lp2) * math.Exp(-p*(6.2-2.2*norm)) * (0.30 + 0.17*norm)

		// Low rumble tail becomes more prominent with magnitude.
		rumLP = rumLP*0.95 + lcg(&seed)*0.05
		rumble := rumLP * math.Exp(-p*(3.0-1.5*norm)) * (0.06 + 0.20*norm)

		// High "snap" gives small explosions more bite.
		spark := math.Sin(2*math.Pi*(2400-900*p)*float64(i)/SampleRate) *
			math.Exp(-p*30) * (0.08 * (1.0 - 0.55*norm))

		s := sub + crack + body + rumble + spark
		putStereoF32(buf, i, softSat(s*0.86))
	}
	return buf
}

// genBonus: ascending FM bell arpeggio — rich, musical.
func genBonus() []byte {
	freqs := []float64{523.25, 659.25, 783.99, 1046.5} // C5 E5 G5 C6
	noteLen := SampleRate * 75 / 1000
	tail := int(0.18 * SampleRate)
	total := len(freqs)*noteLen + tail
	mix := make([]float64, total)

	for fi, freq := range freqs {
		start := fi * noteLen
		dur := total - start
		for j := 0; j < dur; j++ {
			t := float64(start+j) / SampleRate
			np := float64(j) / float64(dur)
			env := adsr(np, 0.004, 0.55, 0.05, 0.35)
			s := fm(t, freq, 2.756, 5.0*env) * env * 0.38
			s += math.Sin(2*math.Pi*freq*2*t) * env * 0.09
			mix[start+j] += s
		}
	}
	buf := makeBuf(total)
	for i, s := range mix {
		putStereoF32(buf, i, softSat(s))
	}
	return buf
}

// genFire: crackling noise with low-frequency amplitude modulation.
func genFire() []byte {
	n := int(0.13 * SampleRate)
	buf := makeBuf(n)
	seed := uint64(33333)
	lp := 0.0
	for i := 0; i < n; i++ {
		t := float64(i) / SampleRate
		p := float64(i) / float64(n)
		raw := lcg(&seed)
		lp = lp*0.65 + raw*0.35
		mod := 0.5 + 0.5*math.Sin(2*math.Pi*16*t)
		env := (1 - p) * 0.38
		s := (raw*0.3 + lp*0.55) * mod * env
		putStereoF32(buf, i, softSat(s))
	}
	return buf
}

// genHurt: descending FM tone — "oof".
func genHurt() []byte {
	n := int(0.16 * SampleRate)
	buf := makeBuf(n)
	for i := 0; i < n; i++ {
		t := float64(i) / SampleRate
		p := float64(i) / float64(n)
		env := adsr(p, 0.015, 0.55, 0.1, 0.25)
		freq := 320 - 220*p
		s := fm(t, freq, 1.5, 2.8*(1-p)) * env * 0.52
		// Add warm second harmonic.
		s += math.Sin(2*math.Pi*freq*2*t) * env * 0.1
		putStereoF32(buf, i, softSat(s))
	}
	return buf
}

// genLevelUp: ascending FM bell staircase — each note rings over the next.
func genLevelUp() []byte {
	notes := []float64{440, 554.37, 659.25, 880, 1108.73}
	noteStep := int(0.09 * SampleRate)
	total := len(notes)*noteStep + int(0.25*SampleRate)
	mix := make([]float64, total)

	for fi, freq := range notes {
		start := fi * noteStep
		dur := total - start
		for j := 0; j < dur; j++ {
			t := float64(start+j) / SampleRate
			np := float64(j) / float64(dur)
			env := adsr(np, 0.003, 0.65, 0.04, 0.28)
			s := fm(t, freq, 3.5, 5.5*env) * env * 0.28
			s += math.Sin(2*math.Pi*freq*2*t) * env * 0.07
			mix[start+j] += s
		}
	}
	buf := makeBuf(total)
	for i, s := range mix {
		putStereoF32(buf, i, softSat(s))
	}
	return buf
}

// genGameOver: slow descending minor chord, staggered.
func genGameOver() []byte {
	dur := 0.75
	n := int(dur * SampleRate)
	notes := []struct{ freq, onset float64 }{
		{329.63, 0.00}, // E4
		{261.63, 0.14}, // C4
		{220.00, 0.28}, // A3
	}
	mix := make([]float64, n)
	for _, note := range notes {
		start := int(note.onset * SampleRate)
		for i := start; i < n; i++ {
			t := float64(i) / SampleRate
			np := float64(i-start) / float64(n-start)
			env := adsr(np, 0.008, 0.25, 0.3, 0.45)
			freq := note.freq * (1 - np*0.025) // slight pitch drop
			s := fm(t, freq, 2.0, 2.0*env) * env * 0.32
			s += math.Sin(2*math.Pi*freq*0.5*t) * env * 0.1 // sub
			mix[i] += s
		}
	}
	buf := makeBuf(n)
	for i, s := range mix {
		putStereoF32(buf, i, softSat(s))
	}
	return buf
}

// genMenuSelect: crisp click + brief high tone.
func genMenuSelect() []byte {
	n := SampleRate * 65 / 1000
	buf := makeBuf(n)
	for i := 0; i < n; i++ {
		t := float64(i) / SampleRate
		p := float64(i) / float64(n)
		env := adsr(p, 0.004, 0.55, 0.0, 0.1)
		freq := 1400 - 700*p
		s := fm(t, freq, 1.0, 0.6) * env * 0.38
		putStereoF32(buf, i, softSat(s))
	}
	return buf
}

// genSplatter: sub thump + wet filtered noise burst.
func genSplatter() []byte {
	seed := atomic.AddUint64(&splatterVariantCounter, 1) ^ uint64(time.Now().UnixNano())
	switch int(seed & 3) {
	case 0:
		return genSplatterWet(seed)
	case 1:
		return genSplatterCrunch(seed)
	case 2:
		return genSplatterGush(seed)
	default:
		return genSplatterPop(seed)
	}
}

// genSplatterWet: bassy wet thump.
func genSplatterWet(seed uint64) []byte {
	n := int(0.15 * SampleRate)
	buf := makeBuf(n)
	lp := 0.0
	for i := 0; i < n; i++ {
		t := float64(i) / SampleRate
		p := float64(i) / float64(n)
		thump := fm(t, 90, 0.5, 0.8) * math.Exp(-p*16) * 0.5
		lp = lp*0.55 + lcg(&seed)*0.45
		wet := lp * math.Exp(-p*14) * 0.45
		// Pitch blip.
		blip := math.Sin(2*math.Pi*(180-140*p)*t) * math.Exp(-p*25) * 0.12
		s := thump + wet + blip
		putStereoF32(buf, i, softSat(s*0.72))
	}
	return buf
}

// genSplatterCrunch: short crunchy crack + low body.
func genSplatterCrunch(seed uint64) []byte {
	n := int(0.11 * SampleRate)
	buf := makeBuf(n)
	lp := 0.0
	for i := 0; i < n; i++ {
		t := float64(i) / SampleRate
		p := float64(i) / float64(n)
		crack := 0.0
		if p < 0.18 {
			crack = lcg(&seed) * (1.0 - p/0.18) * 0.42
		}
		lp = lp*0.62 + lcg(&seed)*0.38
		body := lp * math.Exp(-p*12) * 0.3
		thumpFreq := 130 - 70*p
		thump := math.Sin(2*math.Pi*thumpFreq*t) * math.Exp(-p*18) * 0.33
		s := crack + body + thump
		putStereoF32(buf, i, softSat(s*0.78))
	}
	return buf
}

// genSplatterGush: longer sloshy burst with wobble.
func genSplatterGush(seed uint64) []byte {
	n := int(0.19 * SampleRate)
	buf := makeBuf(n)
	lp := 0.0
	phase := 0.0
	for i := 0; i < n; i++ {
		p := float64(i) / float64(n)
		wob := 55 + 40*math.Sin(2*math.Pi*p*3.2)
		phase += 2 * math.Pi * wob / SampleRate
		sub := math.Sin(phase) * math.Exp(-p*6.5) * 0.34
		lp = lp*0.72 + lcg(&seed)*0.28
		slosh := lp * math.Exp(-p*8.5) * 0.36
		hiss := lcg(&seed) * math.Exp(-p*20) * 0.08
		s := sub + slosh + hiss
		putStereoF32(buf, i, softSat(s*0.72))
	}
	return buf
}

// genSplatterPop: quick punctuated splat.
func genSplatterPop(seed uint64) []byte {
	n := int(0.09 * SampleRate)
	buf := makeBuf(n)
	for i := 0; i < n; i++ {
		t := float64(i) / SampleRate
		p := float64(i) / float64(n)
		env := math.Exp(-p * 16)
		pop := fm(t, 210-120*p, 1.2, 0.9) * env * 0.32
		noise := lcg(&seed) * math.Exp(-p*22) * 0.24
		s := pop + noise
		putStereoF32(buf, i, softSat(s*0.82))
	}
	return buf
}

func genPoliceSiren() []byte {
	return genPoliceSirenSpatial(1.0, 0.0)
}

func genPoliceSirenSpatial(doppler, pan float64) []byte {
	if doppler <= 0 {
		doppler = 1.0
	}
	pan = clampF(pan, -1.0, 1.0)

	variant := int(atomic.AddUint64(&policeSirenVariantCounter, 1) % 3)
	dur := 0.92
	if variant == 1 {
		dur = 0.86
	} else if variant == 2 {
		dur = 1.00
	}
	n := int(dur * SampleRate)
	buf := makeBuf(n)
	phase := 0.0
	seed := uint64(0xC0D51E7) ^ uint64(variant+1) ^ uint64(time.Now().UnixNano())
	for i := 0; i < n; i++ {
		t := float64(i) / SampleRate

		freq := 720.0
		switch variant {
		case 0:
			// Wail: smooth up/down sweep with a long crest.
			cycle := 0.74
			c := math.Mod(t, cycle) / cycle
			tri := 1.0 - math.Abs(2*c-1.0)   // 0..1
			shape := tri * tri * (3 - 2*tri) // smoothstep
			freq = 620.0 + 440.0*shape
		case 1:
			// Yelp: short urgent chirps.
			cycle := 0.22
			c := math.Mod(t, cycle) / cycle
			if c < 0.62 {
				u := c / 0.62
				freq = 820.0 + 520.0*u*u
			} else {
				u := (c - 0.62) / 0.38
				freq = 1140.0 - 300.0*u
			}
		default:
			// Hi-Lo: stepped two-tone with tiny glide.
			cycle := 0.33
			c := math.Mod(t, cycle) / cycle
			if c < 0.5 {
				freq = 980.0 - 60.0*c
			} else {
				freq = 720.0 + 90.0*(c-0.5)
			}
		}
		freq *= 1.0 + 0.006*math.Sin(2*math.Pi*(5.0+0.4*float64(variant))*t+float64(variant)*0.9)
		freq *= doppler

		phase += 2 * math.Pi * freq / SampleRate

		// Cleaner horn stack: lower upper-harmonic energy, mild grit.
		raw := math.Sin(phase)*0.84 +
			math.Sin(phase*2.0+0.22)*0.18 +
			math.Sin(phase*3.0+0.55)*0.07 +
			lcg(&seed)*0.012

		am := 0.90 + 0.10*math.Sin(2*math.Pi*(2.7+0.4*float64(variant))*t)
		if variant == 1 {
			am *= 0.88 + 0.12*math.Sin(2*math.Pi*7.5*t)
		}
		s := softSat(raw*1.55) * 0.30 * am

		// Click-free onset/offset.
		env := clampF(t*34.0, 0, 1) * clampF((dur-t)*24.0, 0, 1)

		// Static world-space pan plus subtle motion keeps directionality clear.
		basePan := 0.5 + 0.5*pan
		panner := clampF(basePan+0.08*math.Sin(2*math.Pi*(0.16+0.02*float64(variant))*t+float64(variant)*1.2), 0, 1)
		left := softSat(s * env * math.Sqrt(1.0-panner))
		right := softSat(s * env * math.Sqrt(panner))
		if variant == 2 {
			// Keep hi-lo a bit more centered.
			mid := 0.5 * (left + right)
			left = left*0.8 + mid*0.2
			right = right*0.8 + mid*0.2
		}
		putStereoF32LR(buf, i, left, right)
	}
	return buf
}

// genHelicopterSnd: rotor pulse + turbine whine + broadband noise.
func genHelicopterSnd(dur float64) []byte {
	n := int(dur * SampleRate)
	buf := makeBuf(n)
	seed := uint64(55555)
	lp := 0.0
	for i := 0; i < n; i++ {
		t := float64(i) / SampleRate
		p := float64(i) / float64(n)
		// Main rotor: fast-attack, sharp pulse at ~17.5 Hz.
		rotPhase := math.Mod(t*17.5, 1.0)
		rotor := math.Exp(-rotPhase*10) * 0.58
		// Turbine: warm FM drone.
		turbine := fm(t, 310, 1.5, 1.8) * 0.11
		// Air broadband.
		lp = lp*0.78 + lcg(&seed)*0.22
		env := 1 - p*0.18
		s := (rotor + turbine + lp*0.14) * 0.5 * env
		putStereoF32(buf, i, softSat(s))
	}
	return buf
}

// genGunshot: transient crack + sub pitch-drop + noise body.
func genGunshot() []byte {
	n := int(0.11 * SampleRate)
	buf := makeBuf(n)
	seed := uint64(77777)
	for i := 0; i < n; i++ {
		t := float64(i) / SampleRate
		p := float64(i) / float64(n)
		// Sharp transient (first 15ms).
		crack := 0.0
		if p < 0.014 {
			crack = lcg(&seed) * (1 - p/0.014) * 0.88
		}
		// Pitched sub drop: 200→35 Hz.
		thumpFreq := 200 * math.Pow(0.04, p*4)
		thump := math.Sin(2*math.Pi*thumpFreq*t) * math.Exp(-p*22) * 0.62
		// Noise body with decay.
		body := lcg(&seed) * math.Pow(1-p, 5) * 0.28
		// High-frequency ring.
		ring := math.Sin(2*math.Pi*3400*t) * math.Exp(-p*35) * 0.09
		s := crack + thump + body + ring
		putStereoF32(buf, i, softSat(s*0.82))
	}
	return buf
}

// genScream: voiced "aaah" with glottal harmonics, formants, and vibrato.
func genScream() []byte {
	n := int(0.40 * SampleRate)
	buf := makeBuf(n)
	seed := uint64(99999)
	for i := 0; i < n; i++ {
		t := float64(i) / SampleRate
		p := float64(i) / float64(n)
		env := adsr(p, 0.025, 0.04, 0.88, 0.12)
		vib := 1 + 0.022*math.Sin(2*math.Pi*5.5*t)
		pitch := (235 - 50*p) * vib

		// Glottal source: sum of harmonics (sawtooth approximation).
		src := 0.0
		for h := 1; h <= 9; h++ {
			src += math.Sin(2*math.Pi*pitch*float64(h)*t) / float64(h)
		}
		src *= 0.10

		// Formants for "ah" vowel.
		f1 := 760 + 35*math.Sin(t*2.8)
		f2 := 1220 + 55*math.Sin(t*3.3)
		formant := math.Sin(2*math.Pi*f1*t)*0.22 + math.Sin(2*math.Pi*f2*t)*0.13

		breath := lcg(&seed) * 0.035
		s := (src + formant + breath) * env * 0.52
		putStereoF32(buf, i, softSat(s))
	}
	return buf
}

// ---- Music system -------------------------------------------------------

type musicReader struct {
	t        float64
	beat     int
	seed     uint64
	measure  int
	chordIdx int
	menuMode bool
	level    int
	section  int
	lp       float64 // shared lowpass state for percussion noise
	lp2      float64
}

var musicVolume float64 = 0.08
var sfxVolume float64 = 0.58
var currentMusicLevel int = 1

func StartMenuMusic()       { startMusic(true, 1, 0.24) }
func StartBackgroundMusic() { StartLevelMusic(currentMusicLevel) }
func StartLevelMusic(level int) {
	currentMusicLevel = level
	startMusic(false, level, 0.14)
}
func SetMusicVolume(vol float64) {
	musicVolume = vol
	if globalAudio != nil && globalAudio.musicPlayer != nil {
		globalAudio.musicPlayer.SetVolume(vol)
	}
}

func SetSFXVolume(vol float64) {
	if vol < 0 {
		vol = 0
	} else if vol > 1 {
		vol = 1
	}
	sfxVolume = vol
}

func startMusic(menuMode bool, level int, volume float64) {
	if globalAudio == nil {
		return
	}
	select {
	case <-globalAudio.ready:
	default:
		return
	}
	if globalAudio.musicPlayer != nil {
		globalAudio.musicPlayer.Close()
	}
	musicVolume = volume
	reader := &musicReader{
		seed:     uint64(time.Now().UnixNano()),
		menuMode: menuMode,
		level:    level,
	}
	player := globalAudio.ctx.NewPlayer(reader)
	player.SetVolume(volume)
	globalAudio.musicPlayer = player
	player.Play()
}

func (m *musicReader) Read(p []byte) (int, error) {
	samples := len(p) / 8
	if samples == 0 {
		return 0, nil
	}
	if m.menuMode {
		return m.readMenuMusic(p, samples)
	}
	return m.readGameMusic(p, samples)
}

// ---- Music instruments (stateless per-sample, driven by m.t) ------------

// kick returns a kick drum sample given time-since-trigger (trig) in seconds.
// Uses a pitch-swept sine with a transient click and short air tail.
func kick(trig float64) float64 {
	if trig > 0.25 {
		return 0
	}
	phase := 2 * math.Pi * 185 / 12.5 * (1 - math.Exp(-trig*12.5))
	body := math.Sin(phase) * math.Exp(-trig*18.0) * 0.80
	click := math.Sin(2*math.Pi*2100*trig) * math.Exp(-trig*250.0) * 0.24
	air := math.Sin(2*math.Pi*330*trig) * math.Exp(-trig*38.0) * 0.12
	return softSat(body + click + air)
}

// snare returns a snare sample given time-since-trigger.
func snare(trig float64, seed *uint64) float64 {
	if trig > 0.2 {
		return 0
	}
	env := math.Exp(-trig * 26.0)
	body := (math.Sin(2*math.Pi*188*trig)*0.24 + math.Sin(2*math.Pi*356*trig)*0.10) * env
	n1 := lcg(seed)
	n2 := lcg(seed)
	bandNoise := (n1 - n2*0.55) * env * (0.55 + 0.25*math.Exp(-trig*8.0))
	snap := math.Sin(2*math.Pi*2800*trig) * math.Exp(-trig*120.0) * 0.10
	return softSat(body + bandNoise + snap)
}

// hihat returns a closed hi-hat sample. open=true for longer decay.
func hihat(trig float64, open bool, seed *uint64) float64 {
	decay := 42.0
	limit := 0.06
	if open {
		decay = 15.0
		limit = 0.18
	}
	if trig > limit {
		return 0
	}
	n := lcg(seed)
	metal := math.Sin(2*math.Pi*7300*trig) + math.Sin(2*math.Pi*9200*trig)*0.6
	s := (n*0.8 + metal*0.2) * math.Exp(-trig*decay) * 0.07
	return softSat(s)
}

// fmBass returns a warm FM bass sample — low modRatio gives smooth tone.
func fmBass(t, freq, env float64) float64 {
	b := fm(t, freq, 0.5, 1.25*env) * env * 0.48
	b += math.Sin(2*math.Pi*freq*t) * env * 0.26
	b += math.Sin(2*math.Pi*freq*0.5*t) * env * 0.10
	return softSat(b)
}

// fmPad returns a lush pad sample from a chord — three detuned FM oscillators per note.
func fmPad(t float64, chord []float64, env float64) float64 {
	s := 0.0
	detunes := [4]float64{-0.004, -0.001, 0.002, 0.005}
	for _, freq := range chord {
		for _, d := range detunes {
			f := freq * (1 + d)
			vib := 1 + 0.003*math.Sin(2*math.Pi*(0.23+f*0.0007)*t)
			s += fm(t, f*vib, 1.45, 0.75*env) * 0.048
		}
	}
	return softSat(s)
}

// fmArp returns an FM arpeggio sample for one note.
func fmArp(t, freq, env float64) float64 {
	s := fm(t, freq, 2.0, 3.2*env) * env * 0.20
	s += math.Sin(2*math.Pi*freq*2*t) * env * 0.08
	return softSat(s)
}

// fmLead returns an FM lead/melody sample.
func fmLead(t, freq, env float64) float64 {
	vib := 1 + 0.01*math.Sin(2*math.Pi*5.4*t)
	s := fm(t, freq*vib, 1.55, 2.7*env) * env * 0.26
	s += math.Sin(2*math.Pi*freq*2.98*t) * env * 0.07
	return softSat(s)
}

func triWave(phase float64) float64 {
	return (2.0 / math.Pi) * math.Asin(math.Sin(phase))
}

func softSquareWave(phase float64) float64 {
	return math.Tanh(math.Sin(phase) * 3.4)
}

// ---- Menu music ---------------------------------------------------------

func (m *musicReader) readMenuMusic(p []byte, samples int) (int, error) {
	chords := [][]float64{
		{261.6, 329.6, 392.0, 493.9}, // Cmaj7
		{220.0, 261.6, 329.6, 392.0}, // Am7
		{174.6, 220.0, 261.6, 349.2}, // Fmaj7
		{196.0, 246.9, 293.7, 392.0}, // G
		{261.6, 329.6, 392.0, 523.2}, // Cadd9
		{146.8, 174.6, 220.0, 293.7}, // Dm7
		{164.8, 207.7, 246.9, 329.6}, // Em7
		{196.0, 246.9, 293.7, 370.0}, // G7
	}
	const tempo = 1.95 // 117 BPM
	const beatsPerChord = 4
	const step16Len = 1.0 / (tempo * 4.0)
	const step8Len = 1.0 / (tempo * 2.0)

	kickPattern := [16]bool{
		true, false, false, false,
		true, false, false, false,
		true, false, false, true,
		true, false, false, false,
	}
	snarePattern := [16]bool{
		false, false, false, false,
		true, false, false, false,
		false, false, false, false,
		true, false, false, false,
	}
	bassPattern := [8]bool{true, false, true, false, true, false, false, true}
	// Absolute-note hook for a clearly different melodic identity.
	leadNotesA := [16]float64{
		523.25, 0, 587.33, 0,
		659.25, 0, 587.33, 0,
		523.25, 0, 493.88, 0,
		523.25, 0, 659.25, 783.99,
	}
	leadNotesB := [16]float64{
		523.25, 0, 587.33, 0,
		659.25, 0, 698.46, 0,
		659.25, 0, 587.33, 0,
		523.25, 0, 659.25, 880.00,
	}
	arpOrder := [8]int{0, 1, 2, 1, 3, 2, 1, 0}

	for i := 0; i < samples && i*8+7 < len(p); i++ {
		m.t += 1.0 / SampleRate
		beatLen := 1.0 / tempo
		beatTrig := math.Mod(m.t, beatLen)
		beatPos := beatTrig / beatLen
		step16Trig := math.Mod(m.t, step16Len)
		step8Trig := math.Mod(m.t, step8Len)
		step16 := int(m.t*tempo*4) % 16
		step8 := int(m.t*tempo*2) % 8
		currentBeat := int(m.t * tempo)

		if currentBeat/4 != m.measure {
			m.measure = currentBeat / 4
		}
		m.section = (m.measure / 8) % 3 // verse, response, lift

		chordStep := (currentBeat / beatsPerChord) % len(chords)
		chord := chords[chordStep]
		chordProg := math.Mod(m.t*tempo, beatsPerChord) / beatsPerChord

		s := 0.0

		// "E-piano"-like chord bed (harmonic stack, no FM timbre).
		chordEnv := 0.55 + 0.45*math.Min(1.0, chordProg*1.2)
		for ni := 0; ni < len(chord); ni++ {
			freq := chord[ni]
			ph := 2 * math.Pi * freq * m.t
			vox := math.Sin(ph)*0.68 + math.Sin(ph*2.0)*0.22 + triWave(ph*0.5)*0.10
			detune := math.Sin(2*math.Pi*(freq*1.003)*m.t) * 0.08
			s += (vox + detune) * chordEnv * 0.09
		}

		// Pulse bass with triangle body.
		if bassPattern[step8] {
			bassFreq := chord[0] / 2
			if step8 == 2 || step8 == 6 {
				bassFreq = chord[1] / 2
			}
			if step8 == 7 {
				bassFreq = chord[0]
			}
			bEnv := adsr(math.Mod(m.t*tempo*2, 1.0), 0.02, 0.52, 0.26, 0.2)
			bPh := 2 * math.Pi * bassFreq * m.t
			bass := triWave(bPh)*0.58 + softSquareWave(bPh*0.5)*0.24
			s += bass * bEnv * 0.42
		}

		// Fresh drum timbre: muted analog-style kick/snare + light shaker.
		if kickPattern[step16] {
			kf := 120.0*math.Exp(-step16Trig*16.0) + 42.0
			kickTone := math.Sin(2*math.Pi*kf*step16Trig) * math.Exp(-step16Trig*13.0) * 0.46
			kickClick := math.Sin(2*math.Pi*1800*step16Trig) * math.Exp(-step16Trig*120.0) * 0.08
			s += kickTone + kickClick
		}
		if snarePattern[step16] {
			n := lcg(&m.seed)
			snBody := math.Sin(2*math.Pi*195*step16Trig) * math.Exp(-step16Trig*22.0) * 0.16
			snNoise := n * math.Exp(-step16Trig*30.0) * 0.20
			s += snBody + snNoise
		}
		if step16%2 == 1 {
			shake := lcg(&m.seed) * math.Exp(-step8Trig*20.0) * 0.09
			s += shake
		}

		// Pluck counter line.
		arpIdx := arpOrder[step8]
		if arpIdx >= len(chord) {
			arpIdx = len(chord) - 1
		}
		arpFreq := chord[arpIdx]
		if step8%2 == 1 {
			arpFreq *= 2.0
		}
		arpEnv := adsr(math.Mod(m.t*tempo*2, 1.0), 0.01, 0.34, 0.14, 0.2)
		arpPh := 2 * math.Pi * arpFreq * m.t
		pluck := softSquareWave(arpPh)*0.65 + math.Sin(arpPh*2.0)*0.2
		s += pluck * arpEnv * 0.20

		// Distinct hook melody (A/B response) with different voice.
		hook := leadNotesA
		if m.section == 1 {
			hook = leadNotesB
		} else if m.section == 2 && m.measure%2 == 1 {
			hook = leadNotesB
		}
		note := hook[step16]
		if note > 0 {
			leadEnv := adsr(math.Mod(m.t*tempo*4, 1.0), 0.02, 0.42, 0.36, 0.2)
			lph := 2 * math.Pi * note * m.t
			lead := softSquareWave(lph)*0.58 + triWave(lph*2.0)*0.22 + math.Sin(lph*0.5)*0.10
			leadGain := 0.24
			if m.section == 2 {
				leadGain = 0.30
			}
			s += lead * leadEnv * leadGain
		}

		duck := 1.0 - 0.08*math.Exp(-beatTrig*12.0)
		s = softSat(s * duck * 0.90)
		pan := 0.11*math.Sin(2*math.Pi*0.10*m.t) + 0.03*math.Sin(2*math.Pi*0.27*m.t+1.0)
		sparkle := math.Sin(2*math.Pi*chord[3]*3.0*m.t) * 0.008 * (0.5 + 0.5*math.Sin(2*math.Pi*0.18*m.t+beatPos))
		left := softSat(s*(1-pan) + sparkle)
		right := softSat(s*(1+pan) - sparkle)
		putStereoF32LR(p, i, left, right)
	}
	return len(p), nil
}

// ---- Game music ---------------------------------------------------------

func (m *musicReader) readGameMusic(p []byte, samples int) (int, error) {
	const gameMusicStyles = 12
	style := (m.level - 1) % gameMusicStyles
	if style < 0 {
		style += gameMusicStyles
	}
	chords, tempo, _ := m.getLevelSong(style)

	for i := 0; i < samples && i*8+7 < len(p); i++ {
		m.t += 1.0 / SampleRate

		beatLen := 1.0 / tempo
		trig := math.Mod(m.t, beatLen)
		beatPos := trig / beatLen
		currentBeat := int(m.t * tempo)

		chordLen := 2
		if style == 2 || style == 4 || style == 7 {
			chordLen = 4
		}
		if currentBeat/chordLen != m.measure {
			m.measure = currentBeat / chordLen
			m.chordIdx = (m.chordIdx + 1) % len(chords)
		}
		m.section = (currentBeat / 32) % 4 // 8-bar macro sections
		chord := chords[m.chordIdx]

		var s float64
		switch style {
		case 0:
			s = m.mixFunkyGroove(chord, tempo, trig, beatPos, currentBeat)
		case 1:
			s = m.mixDarkElectronic(chord, tempo, trig, beatPos, currentBeat)
		case 2:
			s = m.mixChillSynthwave(chord, tempo, trig, beatPos, currentBeat)
		case 3:
			s = m.mixIntenseAction(chord, tempo, trig, beatPos, currentBeat)
		case 4:
			s = m.mixAmbientMysterious(chord, tempo, trig, beatPos, currentBeat)
		case 5:
			s = m.mixRetroArcade(chord, tempo, trig, beatPos, currentBeat)
		case 6:
			s = m.mixIndustrialDNB(chord, tempo, trig, beatPos, currentBeat)
		case 7:
			s = m.mixNoirPulse(chord, tempo, trig, beatPos, currentBeat)
		case 8:
			s = m.mixNeonDrive(chord, tempo, trig, beatPos, currentBeat)
		case 9:
			s = m.mixMetalRush(chord, tempo, trig, beatPos, currentBeat)
		case 10:
			s = m.mixDesertRun(chord, tempo, trig, beatPos, currentBeat)
		case 11:
			s = m.mixSkylineCruise(chord, tempo, trig, beatPos, currentBeat)
		}

		energy := [4]float64{0.80, 0.92, 1.00, 0.88}[m.section]
		if m.section == 3 && currentBeat%8 == 7 {
			s += snare(trig, &m.seed) * 0.45
		}
		duck := 1.0 - 0.16*math.Exp(-trig*20.0)
		s = softSat(s * energy * duck)
		pan := 0.09*math.Sin(2*math.Pi*(0.09+float64(style)*0.003)*m.t+float64(style)*0.6) + 0.015*math.Sin(2*math.Pi*0.31*m.t)
		color := math.Sin(2*math.Pi*chord[2]*2.0*m.t) * 0.01 * (0.35 + 0.65*float64(m.section)/3.0)
		left := softSat(s*(1-pan) + color)
		right := softSat(s*(1+pan) - color)
		putStereoF32LR(p, i, left, right)
	}
	return len(p), nil
}

func (m *musicReader) getLevelSong(style int) ([][]float64, float64, float64) {
	switch style {
	case 0: // Funky groove — 120 BPM, Dm7 family
		return [][]float64{
			{146.8, 174.6, 220.0}, // Dm7
			{130.8, 164.8, 196.0}, // Cmaj7
			{116.5, 146.8, 174.6}, // Bbmaj7
			{110.0, 138.6, 164.8}, // Am7
			{146.8, 185.0, 220.0}, // Dm9
			{130.8, 164.8, 207.7}, // C6
			{123.5, 155.6, 185.0}, // Bdim
			{110.0, 130.8, 164.8}, // Am
		}, 2.0, 0.6

	case 1: // Dark electronic — 132 BPM, minor
		return [][]float64{
			{82.4, 103.8, 123.5},
			{73.4, 92.5, 110.0},
			{65.4, 82.4, 98.0},
			{69.3, 87.3, 103.8},
			{82.4, 103.8, 130.8},
			{73.4, 92.5, 116.5},
			{65.4, 82.4, 103.8},
			{61.7, 77.8, 92.5},
		}, 2.2, 0.0

	case 2: // Chill synthwave — 108 BPM
		return [][]float64{
			{220.0, 277.2, 329.6},
			{196.0, 246.9, 293.7},
			{174.6, 220.0, 261.6},
			{164.8, 207.7, 246.9},
			{220.0, 261.6, 329.6},
			{196.0, 246.9, 311.1},
			{174.6, 220.0, 277.2},
			{164.8, 207.7, 261.6},
		}, 1.8, 0.0

	case 3: // Intense action — 156 BPM
		return [][]float64{
			{130.8, 164.8, 196.0},
			{146.8, 185.0, 220.0},
			{164.8, 207.7, 246.9},
			{146.8, 185.0, 220.0},
			{174.6, 220.0, 261.6},
			{164.8, 207.7, 246.9},
			{146.8, 185.0, 220.0},
			{130.8, 164.8, 196.0},
		}, 2.6, 0.0

	case 4: // Ambient mysterious — 84 BPM
		return [][]float64{
			{98.0, 123.5, 146.8},
			{110.0, 138.6, 164.8},
			{116.5, 146.8, 174.6},
			{103.8, 130.8, 155.6},
			{98.0, 123.5, 155.6},
			{110.0, 146.8, 164.8},
			{116.5, 138.6, 174.6},
			{103.8, 123.5, 164.8},
		}, 1.4, 0.0

	case 5: // Retro arcade — 168 BPM
		return [][]float64{
			{261.6, 329.6, 392.0},
			{220.0, 277.2, 329.6},
			{196.0, 246.9, 293.7},
			{174.6, 220.0, 261.6},
			{261.6, 311.1, 392.0},
			{220.0, 261.6, 329.6},
			{196.0, 246.9, 311.1},
			{174.6, 220.0, 277.2},
		}, 2.8, 0.0

	case 6: // Industrial DnB — 174 BPM
		return [][]float64{
			{82.4, 103.8, 123.5},
			{87.3, 110.0, 130.8},
			{73.4, 92.5, 110.0},
			{65.4, 82.4, 98.0},
			{92.5, 116.5, 138.6},
			{82.4, 103.8, 123.5},
			{69.3, 87.3, 103.8},
			{73.4, 92.5, 110.0},
		}, 2.9, 0.0

	case 7: // Noir pulse — 96 BPM
		return [][]float64{
			{110.0, 130.8, 164.8},
			{98.0, 123.5, 155.6},
			{92.5, 110.0, 138.6},
			{82.4, 98.0, 123.5},
			{116.5, 138.6, 174.6},
			{103.8, 130.8, 164.8},
			{98.0, 116.5, 146.8},
			{92.5, 110.0, 138.6},
		}, 1.6, 0.0

	case 8: // Neon drive — 128 BPM
		return [][]float64{
			{146.8, 185.0, 220.0},
			{130.8, 164.8, 196.0},
			{123.5, 155.6, 185.0},
			{110.0, 138.6, 164.8},
			{164.8, 207.7, 246.9},
			{146.8, 185.0, 220.0},
			{130.8, 164.8, 207.7},
			{123.5, 155.6, 196.0},
		}, 2.13, 0.0

	case 9: // Metal rush — 182 BPM
		return [][]float64{
			{98.0, 123.5, 146.8},
			{92.5, 116.5, 138.6},
			{110.0, 138.6, 164.8},
			{87.3, 110.0, 130.8},
			{116.5, 146.8, 174.6},
			{98.0, 123.5, 155.6},
			{92.5, 110.0, 138.6},
			{82.4, 103.8, 123.5},
		}, 3.03, 0.0

	case 10: // Desert run — 112 BPM
		return [][]float64{
			{130.8, 164.8, 196.0},
			{146.8, 174.6, 220.0},
			{174.6, 220.0, 261.6},
			{155.6, 196.0, 233.1},
			{146.8, 185.0, 220.0},
			{130.8, 164.8, 207.7},
			{116.5, 146.8, 185.0},
			{123.5, 155.6, 196.0},
		}, 1.87, 0.0

	case 11: // Skyline cruise — 104 BPM
		return [][]float64{
			{220.0, 261.6, 329.6},
			{196.0, 246.9, 311.1},
			{174.6, 220.0, 277.2},
			{164.8, 207.7, 261.6},
			{233.1, 293.7, 349.2},
			{220.0, 277.2, 329.6},
			{196.0, 246.9, 293.7},
			{174.6, 220.0, 261.6},
		}, 1.73, 0.0

	default:
		return [][]float64{{130.8, 164.8, 196.0}}, 2.0, 0.0
	}
}

// ---- Per-style game music mixers ----------------------------------------

func (m *musicReader) mixFunkyGroove(chord []float64, tempo, trig, beatPos float64, beat int) float64 {
	s := fmPad(m.t, chord, 0.65) * 0.7

	// Groovy sub bass with syncopation.
	bassFreq := chord[0] / 2
	bassOnBeats := beat%4 == 0 || beat%4 == 3 || (beat%4 == 1 && beatPos > 0.5)
	if bassOnBeats {
		be := math.Exp(-trig * 14)
		s += fmBass(m.t, bassFreq, be)
	}

	// Funky chord stabs on offbeats.
	stabPattern := [8]bool{false, false, true, false, false, true, false, true}
	sixteenth := int(m.t*tempo*4) % 8
	sixPos := math.Mod(m.t*tempo*4, 1.0)
	if stabPattern[sixteenth] && sixPos < 0.18 {
		env := adsr(sixPos, 0.01, 0.4, 0.0, 0.1)
		s += fmArp(m.t, chord[1]*2, env) * 0.8
	}

	// Arp on 16ths.
	arpIdx := int(m.t*tempo*4) % len(chord)
	arpEnv := math.Exp(-math.Mod(m.t*tempo*4, 1.0) / tempo * 4 * 12)
	s += fmArp(m.t, chord[arpIdx], arpEnv) * 0.5

	// Kick on 1 and 3.
	if beat%2 == 0 {
		s += kick(trig) * 0.9
	}
	// Snare on 2 and 4.
	if beat%2 == 1 {
		s += snare(trig, &m.seed) * 0.8
	}
	// Swung hi-hat.
	hhStep := math.Mod(m.t*tempo*2, 1.0)
	hhTrig := hhStep / (tempo * 2)
	open := beat%4 == 3 && hhStep > 0.5
	s += hihat(hhTrig, open, &m.seed)

	return s
}

func (m *musicReader) mixDarkElectronic(chord []float64, tempo, trig, beatPos float64, beat int) float64 {
	// Dark detuned pad.
	s := fmPad(m.t, chord, 0.8) * 0.85

	// Pulsing sub bass.
	bassFreq := chord[0] / 2
	pulse := 0.5 + 0.5*math.Sin(m.t*tempo*math.Pi)
	s += fmBass(m.t, bassFreq, pulse*0.8)

	// 4-on-floor kick with hard attack.
	s += kick(trig) * 1.0

	// Snare on 2 and 4.
	if beat%2 == 1 {
		s += snare(trig, &m.seed) * 0.9
	}

	// 16th hi-hats.
	hhTrig := math.Mod(m.t*tempo*4, 1.0) / (tempo * 4)
	s += hihat(hhTrig, false, &m.seed) * 1.1

	// Dark arpeggio — descending.
	arpIdx := 2 - (int(m.t*tempo*4) % 3)
	if arpIdx < 0 {
		arpIdx = 0
	}
	arpEnv := math.Exp(-math.Mod(m.t*tempo*4, 1.0) * 8)
	s += fmArp(m.t, chord[arpIdx]*2, arpEnv) * 0.55

	return s
}

func (m *musicReader) mixChillSynthwave(chord []float64, tempo, trig, beatPos float64, beat int) float64 {
	// Lush wide pad.
	s := fmPad(m.t, chord, 0.9) * 1.0

	// Smooth bass.
	bassFreq := chord[0] / 2
	if beat%4 == 0 || beat%4 == 2 {
		be := adsr(beatPos, 0.04, 0.5, 0.3, 0.3)
		s += fmBass(m.t, bassFreq, be) * 0.85
	}

	// Soft kick on 1 and 3.
	if beat%2 == 0 {
		s += kick(trig) * 0.65
	}
	// Side-stick on 2 and 4.
	if beat%2 == 1 && trig < 0.02 {
		s += math.Sin(2*math.Pi*900*trig) * math.Exp(-trig*200) * 0.12
	}

	// Dreamy arpeggio on every 8th.
	arpRate := tempo * 2
	arpIdx := int(m.t*arpRate) % len(chord)
	arpEnv := adsr(math.Mod(m.t*arpRate, 1.0), 0.01, 0.6, 0.1, 0.2)
	s += fmArp(m.t, chord[arpIdx]*2, arpEnv) * 0.65

	// Warm lead melody.
	noteIdx := int(m.t*tempo) % 8
	scale := [8]float64{1.0, 1.125, 1.25, 1.5, 1.667, 1.5, 1.25, 1.0}
	leadFreq := chord[0] * 4 * scale[noteIdx]
	leadEnv := adsr(beatPos, 0.02, 0.5, 0.2, 0.25)
	s += fmLead(m.t, leadFreq, leadEnv) * 0.5

	// Open hi-hat on 2 and 4.
	if beat%2 == 1 {
		s += hihat(trig, true, &m.seed) * 1.2
	}

	return s
}

func (m *musicReader) mixIntenseAction(chord []float64, tempo, trig, beatPos float64, beat int) float64 {
	// Hard pad.
	s := fmPad(m.t, chord, 1.0) * 0.75

	// Pumping bass with sidechain-style dip.
	bassFreq := chord[0] / 2
	pumpEnv := math.Min(1.0, beatPos*4)
	s += fmBass(m.t, bassFreq, pumpEnv) * 0.9

	// Hard kick every beat.
	s += kick(trig) * 1.1

	// Layered snare on 2 and 4.
	if beat%2 == 1 {
		s += snare(trig, &m.seed) * 1.0
	}

	// Fast 16th hi-hats.
	hhTrig := math.Mod(m.t*tempo*4, 1.0) / (tempo * 4)
	s += hihat(hhTrig, false, &m.seed) * 1.2

	// Driving 8th-note arpeggio.
	arpIdx := int(m.t*tempo*2) % len(chord)
	arpEnv := adsr(math.Mod(m.t*tempo*2, 1.0), 0.005, 0.3, 0.1, 0.15)
	s += fmArp(m.t, chord[arpIdx]*2, arpEnv) * 0.75

	// Aggressive lead.
	leadIdx := int(m.t*tempo) % 4
	leadNotes := [4]float64{1.0, 1.25, 1.5, 1.25}
	leadEnv := adsr(beatPos, 0.005, 0.35, 0.15, 0.15)
	s += fmLead(m.t, chord[1]*2*leadNotes[leadIdx], leadEnv) * 0.65

	return s
}

func (m *musicReader) mixAmbientMysterious(chord []float64, tempo, trig, beatPos float64, beat int) float64 {
	// Evolving soft pad with slow modulation.
	padMod := 0.5 + 0.5*math.Sin(m.t*0.25)
	s := fmPad(m.t, chord, padMod) * 1.1

	// Sparse bass notes every 8 beats.
	bassFreq := chord[0] / 2
	if beat%8 == 0 {
		be := adsr(beatPos, 0.05, 0.55, 0.3, 0.4)
		s += fmBass(m.t, bassFreq, be) * 0.7
	}

	// Very sparse kick every 4 beats.
	if beat%4 == 0 {
		s += kick(trig) * 0.45
	}

	// Slow arpeggio (quarter-notes).
	arpIdx := int(m.t*tempo) % len(chord)
	arpEnv := adsr(beatPos, 0.03, 0.7, 0.1, 0.3)
	s += fmArp(m.t, chord[arpIdx]*2, arpEnv) * 0.55

	// Ethereal high shimmer.
	shimmerFreq := chord[2] * 4 * (1 + 0.01*math.Sin(m.t*0.7))
	shimmerEnv := 0.5 + 0.5*math.Sin(m.t*1.5)
	s += fm(m.t, shimmerFreq, 2.0, 1.5) * shimmerEnv * 0.04

	// Occasional deep tone.
	if beat%16 == 0 && beatPos < 0.4 {
		deepEnv := adsr(beatPos/0.4, 0.05, 0.5, 0.2, 0.3)
		s += fm(m.t, chord[0]/4, 1.0, 0.8) * deepEnv * 0.18
	}

	return s
}

func (m *musicReader) mixRetroArcade(chord []float64, tempo, trig, beatPos float64, beat int) float64 {
	// Bright chiptune-like stack over a light pad bed.
	s := fmPad(m.t, chord, 0.45) * 0.45

	// Punchy bass on each beat.
	bassEnv := math.Exp(-trig * 15)
	s += fmBass(m.t, chord[0]/2, bassEnv) * 0.92

	// Drum backbone.
	if beat%2 == 0 {
		s += kick(trig) * 0.95
	} else {
		s += snare(trig, &m.seed) * 0.88
	}

	// Rapid closed hats.
	hhTrig := math.Mod(m.t*tempo*4, 1.0) / (tempo * 4)
	s += hihat(hhTrig, false, &m.seed) * 1.3

	// 16th-note arpeggio for gamey movement.
	arpStep := int(m.t*tempo*4) % len(chord)
	arpEnv := adsr(math.Mod(m.t*tempo*4, 1.0), 0.003, 0.22, 0.06, 0.10)
	s += fmArp(m.t, chord[arpStep]*2.0, arpEnv) * 0.88

	// Square-ish top lead.
	leadStep := int(m.t*tempo*2) % 8
	scale := [8]float64{1.0, 1.25, 1.5, 2.0, 1.5, 1.25, 1.125, 1.0}
	leadFreq := chord[0] * 2 * scale[leadStep]
	leadEnv := adsr(beatPos, 0.004, 0.24, 0.10, 0.11)
	sq := 0.0
	if math.Sin(2*math.Pi*leadFreq*m.t) >= 0 {
		sq = 1
	} else {
		sq = -1
	}
	s += sq * leadEnv * 0.13

	return s
}

func (m *musicReader) mixIndustrialDNB(chord []float64, tempo, trig, beatPos float64, beat int) float64 {
	// Dark pad foundation.
	s := fmPad(m.t, chord, 0.75) * 0.78

	// Rolling reese-like bass motion.
	base := chord[0] / 2
	reese := math.Sin(2*math.Pi*base*m.t) + math.Sin(2*math.Pi*base*1.01*m.t)
	reese *= 0.22 * (0.55 + 0.45*math.Sin(m.t*tempo*math.Pi))
	s += reese

	// DnB drum feel: hard kick each beat, snare emphasis on 2/4.
	s += kick(trig) * 1.05
	if beat%2 == 1 {
		s += snare(trig, &m.seed) * 1.05
	}

	// High-rate hats with occasional open accents.
	hhTrig := math.Mod(m.t*tempo*8, 1.0) / (tempo * 8)
	open := beat%4 == 3 && beatPos > 0.62
	s += hihat(hhTrig, open, &m.seed) * 1.25

	// Metallic stab accents.
	stabStep := int(m.t*tempo*2) % 8
	if stabStep == 3 || stabStep == 7 {
		env := adsr(math.Mod(m.t*tempo*2, 1.0), 0.005, 0.18, 0.0, 0.07)
		s += fmLead(m.t, chord[1]*2.5, env) * 0.55
	}

	return s
}

func (m *musicReader) mixNoirPulse(chord []float64, tempo, trig, beatPos float64, beat int) float64 {
	// Slow moody bed.
	padMod := 0.4 + 0.6*math.Sin(m.t*0.18)
	s := fmPad(m.t, chord, padMod) * 0.95

	// Walking bass flavor.
	walkIdx := int(m.t*tempo) % len(chord)
	walkFreq := chord[walkIdx] / 2
	walkEnv := adsr(beatPos, 0.02, 0.45, 0.20, 0.20)
	s += fmBass(m.t, walkFreq, walkEnv) * 0.72

	// Sparse kit.
	if beat%4 == 0 {
		s += kick(trig) * 0.62
	}
	if beat%4 == 2 {
		s += snare(trig, &m.seed) * 0.60
	}

	// Gentle brushed hats.
	hhTrig := math.Mod(m.t*tempo*2, 1.0) / (tempo * 2)
	s += hihat(hhTrig, beat%8 == 7, &m.seed) * 0.72

	// Noir lead with slight vibrato.
	leadScale := [8]float64{1.0, 1.2, 1.33, 1.5, 1.33, 1.2, 1.125, 1.0}
	leadIdx := int(m.t*tempo) % len(leadScale)
	vib := 1 + 0.01*math.Sin(2*math.Pi*5.2*m.t)
	leadFreq := chord[0] * 2 * leadScale[leadIdx] * vib
	leadEnv := adsr(beatPos, 0.02, 0.38, 0.25, 0.20)
	s += fmLead(m.t, leadFreq, leadEnv) * 0.42

	return s
}

func (m *musicReader) mixNeonDrive(chord []float64, tempo, trig, beatPos float64, beat int) float64 {
	s := fmPad(m.t, chord, 0.84) * 0.72

	// Steady drive bass with occasional octave jumps.
	bassStep := int(m.t*tempo*2) % 8
	bassFreq := chord[0] / 2
	if bassStep == 3 || bassStep == 7 {
		bassFreq = chord[0]
	}
	bassEnv := adsr(math.Mod(m.t*tempo*2, 1.0), 0.01, 0.35, 0.16, 0.16)
	s += fmBass(m.t, bassFreq, bassEnv) * 0.90

	s += kick(trig) * 0.95
	if beat%2 == 1 {
		s += snare(trig, &m.seed) * 0.88
	}

	hhTrig := math.Mod(m.t*tempo*4, 1.0) / (tempo * 4)
	s += hihat(hhTrig, beat%8 == 7, &m.seed) * 0.98

	arpIdx := int(m.t*tempo*4) % len(chord)
	arpEnv := adsr(math.Mod(m.t*tempo*4, 1.0), 0.008, 0.28, 0.12, 0.10)
	s += fmArp(m.t, chord[arpIdx]*2.0, arpEnv) * 0.70

	leadIdx := int(m.t*tempo) % 8
	scale := [8]float64{1.0, 1.25, 1.5, 1.25, 1.667, 1.5, 1.25, 1.125}
	leadEnv := adsr(beatPos, 0.02, 0.35, 0.20, 0.15)
	s += fmLead(m.t, chord[0]*2*scale[leadIdx], leadEnv) * 0.52

	return s
}

func (m *musicReader) mixMetalRush(chord []float64, tempo, trig, beatPos float64, beat int) float64 {
	s := fmPad(m.t, chord, 0.58) * 0.62

	// Fast rolling bass.
	roll := int(m.t*tempo*4) % 16
	bassFreq := chord[0] / 2
	if roll%4 == 3 {
		bassFreq = chord[1] / 2
	}
	bassEnv := adsr(math.Mod(m.t*tempo*4, 1.0), 0.004, 0.24, 0.10, 0.08)
	s += fmBass(m.t, bassFreq, bassEnv) * 0.86

	s += kick(trig) * 1.10
	if beat%2 == 1 {
		s += snare(trig, &m.seed) * 1.02
	}

	hhTrig := math.Mod(m.t*tempo*8, 1.0) / (tempo * 8)
	s += hihat(hhTrig, roll%8 == 7, &m.seed) * 1.24

	leadStep := int(m.t*tempo*2) % 8
	leadScale := [8]float64{1.0, 1.25, 1.5, 1.25, 1.0, 1.125, 1.33, 1.5}
	leadEnv := adsr(math.Mod(m.t*tempo*2, 1.0), 0.005, 0.22, 0.10, 0.10)
	s += fmLead(m.t, chord[2]*leadScale[leadStep], leadEnv) * 0.56

	return s
}

func (m *musicReader) mixDesertRun(chord []float64, tempo, trig, beatPos float64, beat int) float64 {
	s := fmPad(m.t, chord, 0.76) * 0.86

	// Tumbao-like sync bass.
	step8 := int(m.t*tempo*2) % 8
	if step8 == 0 || step8 == 3 || step8 == 5 || step8 == 7 {
		bassFreq := chord[0] / 2
		if step8 == 5 {
			bassFreq = chord[2] / 2
		}
		bassEnv := adsr(math.Mod(m.t*tempo*2, 1.0), 0.02, 0.32, 0.18, 0.2)
		s += fmBass(m.t, bassFreq, bassEnv) * 0.88
	}

	if beat%4 == 0 || beat%4 == 2 {
		s += kick(trig) * 0.82
	}
	if beat%4 == 1 || beat%4 == 3 {
		s += snare(trig, &m.seed) * 0.66
	}
	hhTrig := math.Mod(m.t*tempo*4, 1.0) / (tempo * 4)
	s += hihat(hhTrig, beat%8 == 6, &m.seed) * 0.72

	arpIdx := int(m.t*tempo*2) % len(chord)
	arpEnv := adsr(math.Mod(m.t*tempo*2, 1.0), 0.02, 0.45, 0.12, 0.16)
	s += fmArp(m.t, chord[arpIdx]*1.5, arpEnv) * 0.52

	return s
}

func (m *musicReader) mixSkylineCruise(chord []float64, tempo, trig, beatPos float64, beat int) float64 {
	s := fmPad(m.t, chord, 0.96) * 0.92

	// Long bass notes with light bounce.
	if beat%2 == 0 {
		bassEnv := adsr(beatPos, 0.03, 0.56, 0.26, 0.25)
		s += fmBass(m.t, chord[0]/2, bassEnv) * 0.82
	}

	if beat%4 == 0 {
		s += kick(trig) * 0.62
	}
	if beat%4 == 2 {
		s += snare(trig, &m.seed) * 0.54
	}
	hhTrig := math.Mod(m.t*tempo*2, 1.0) / (tempo * 2)
	s += hihat(hhTrig, beat%8 == 7, &m.seed) * 0.58

	arpRate := tempo * 2
	arpIdx := int(m.t*arpRate) % len(chord)
	arpEnv := adsr(math.Mod(m.t*arpRate, 1.0), 0.015, 0.5, 0.14, 0.18)
	s += fmArp(m.t, chord[arpIdx]*2.0, arpEnv) * 0.48

	leadScale := [8]float64{1.0, 1.125, 1.25, 1.5, 1.667, 1.5, 1.33, 1.125}
	leadStep := int(m.t*tempo) % len(leadScale)
	leadEnv := adsr(beatPos, 0.03, 0.42, 0.30, 0.2)
	s += fmLead(m.t, chord[0]*2*leadScale[leadStep], leadEnv) * 0.38

	return s
}
