package game

// Chunk is a ChunkSize x ChunkSize tile of the world.
// Pixels is RGBA8: RGB is the base colour, A encodes shade factor (255=lit, lower=shadow).
type Chunk struct {
	CX, CY int

	Pixels      []uint8 // RGBA8
	Height      []uint8 // per-pixel height (0=ground, >0=solid)
	Unbreakable []uint8 // 1=indestructible

	Tex uint32 // OpenGL texture id (created lazily)

	NeedsUpload bool
	NeedsShadow bool
}

func NewChunk(cx, cy int) *Chunk {
	n := ChunkSize * ChunkSize
	return &Chunk{
		CX:          cx,
		CY:          cy,
		Pixels:      make([]uint8, n*4),
		Height:      make([]uint8, n),
		Unbreakable: make([]uint8, n),
		NeedsUpload: true,
		NeedsShadow: true,
	}
}

func (c *Chunk) WorldOrigin() (int, int) {
	return c.CX * ChunkSize, c.CY * ChunkSize
}

func (c *Chunk) idx(x, y int) int {
	return y*ChunkSize + x
}

func (c *Chunk) pixOff(i int) int {
	return i * 4
}

func (c *Chunk) set(i int, col RGB, height uint8, shade uint8, unbreakable uint8) {
	o := c.pixOff(i)
	c.Pixels[o+0] = col.R
	c.Pixels[o+1] = col.G
	c.Pixels[o+2] = col.B
	c.Pixels[o+3] = shade
	c.Height[i] = height
	c.Unbreakable[i] = unbreakable
}

func (c *Chunk) setRGBKeepHeight(i int, col RGB) {
	o := c.pixOff(i)
	c.Pixels[o+0] = col.R
	c.Pixels[o+1] = col.G
	c.Pixels[o+2] = col.B
}

func (c *Chunk) setShade(i int, shade uint8) {
	c.Pixels[c.pixOff(i)+3] = shade
}

// RecomputeShadows recalculates per-pixel directional shadows using the height map.
// Uses the World's continuous sun angle for smooth shadow rotation.
func (c *Chunk) RecomputeShadows(w *World) {
	baseX, baseY := c.WorldOrigin()
	chunkX0 := baseX
	chunkY0 := baseY
	chunkX1 := baseX + ChunkSize
	chunkY1 := baseY + ChunkSize

	cosA := w.sunCosA
	sinA := w.sunSinA
	slope := w.sunSlope

	for y := 0; y < ChunkSize; y++ {
		wy := baseY + y
		for x := 0; x < ChunkSize; x++ {
			wx := baseX + x
			i := c.idx(x, y)

			if wx < 0 || wy < 0 || wx >= WorldWidth || wy >= WorldHeight {
				c.setShade(i, ShadeDark)
				continue
			}

			h := c.Height[i]
			shadowed := false
			fwx := float64(wx)
			fwy := float64(wy)
			prevSX, prevSY := wx, wy
			for step := 1; step <= MaxShadowDist; step++ {
				t := float64(step)
				sx := int(fwx + cosA*t + 0.5)
				sy := int(fwy + sinA*t + 0.5)
				// Skip if we haven't moved to a new pixel.
				if sx == prevSX && sy == prevSY {
					continue
				}
				prevSX = sx
				prevSY = sy
				if sx < 0 || sy < 0 || sx >= WorldWidth || sy >= WorldHeight {
					break
				}

				var oh uint8
				if sx >= chunkX0 && sx < chunkX1 && sy >= chunkY0 && sy < chunkY1 {
					oh = c.Height[(sy-chunkY0)*ChunkSize+(sx-chunkX0)]
				} else {
					oh = w.HeightAt(sx, sy)
				}

				if float64(oh)-t*slope > float64(h) {
					shadowed = true
					break
				}
			}

			if shadowed {
				c.setShade(i, ShadeDark)
			} else {
				c.setShade(i, ShadeLit)
			}
		}
	}

	c.NeedsShadow = false
	c.NeedsUpload = true
}
