//go:build !android

package game

import (
	"bytes"
	_ "embed"
	"fmt"
	"image"
	"image/draw"
	"image/png"

	"github.com/go-gl/gl/v4.1-core/gl"
)

//go:embed font_alt.png
var fontPNG []byte

// InitFont loads the font atlas and sets up the text rendering pipeline.
func (r *Renderer) InitFont() error {
	img, err := png.Decode(bytes.NewReader(fontPNG))
	if err != nil {
		return fmt.Errorf("decode font_alt.png: %w", err)
	}
	b := img.Bounds()
	nrgba, ok := img.(*image.NRGBA)
	if !ok || nrgba.Stride != b.Dx()*4 {
		nrgba = image.NewNRGBA(b)
		draw.Draw(nrgba, b, img, b.Min, draw.Src)
	}

	// Upload font atlas to GL texture.
	var tex uint32
	gl.GenTextures(1, &tex)
	gl.BindTexture(gl.TEXTURE_2D, tex)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.NEAREST)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.NEAREST)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_S, gl.CLAMP_TO_EDGE)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_T, gl.CLAMP_TO_EDGE)
	gl.TexImage2D(gl.TEXTURE_2D, 0, gl.RGBA8,
		int32(b.Dx()), int32(b.Dy()), 0,
		gl.RGBA, gl.UNSIGNED_BYTE, gl.Ptr(nrgba.Pix))
	r.fontTex = tex

	// Text shader program.
	prog, err := linkProgram(textVertSrc, textFragSrc)
	if err != nil {
		return fmt.Errorf("text program: %w", err)
	}
	r.textProg = prog
	gl.UseProgram(prog)
	r.textURes = gl.GetUniformLocation(prog, gl.Str("uResolution\x00"))
	r.textUFontTex = gl.GetUniformLocation(prog, gl.Str("uFontTex\x00"))
	gl.Uniform1i(r.textUFontTex, 2) // texture unit 2

	// Text VAO/VBO: per-vertex pos(2) + uv(2) + color(4) = 8 floats.
	var vao, vbo uint32
	gl.GenVertexArrays(1, &vao)
	gl.GenBuffers(1, &vbo)
	gl.BindVertexArray(vao)
	gl.BindBuffer(gl.ARRAY_BUFFER, vbo)

	stride := int32(8 * 4)
	gl.BufferData(gl.ARRAY_BUFFER, 512*6*int(stride), nil, gl.STREAM_DRAW)
	gl.EnableVertexAttribArray(0) // aPos
	gl.VertexAttribPointer(0, 2, gl.FLOAT, false, stride, glOffset(0))
	gl.EnableVertexAttribArray(1) // aUV
	gl.VertexAttribPointer(1, 2, gl.FLOAT, false, stride, glOffset(2*4))
	gl.EnableVertexAttribArray(2) // aColor
	gl.VertexAttribPointer(2, 4, gl.FLOAT, false, stride, glOffset(4*4))

	r.textVAO = vao
	r.textVBO = vbo
	gl.BindVertexArray(0)
	return nil
}

// DrawChar queues a single character as a textured quad in screen pixel space.
func (r *Renderer) DrawChar(ch rune, sx, sy, scale float32, col RGB) {
	if ch < 32 || ch > 126 {
		return
	}
	c := int(ch)
	column := c % FontCols
	row := c / FontCols

	u0 := float32(column*FontCellW) / float32(FontAtlasW)
	v0 := float32(row*FontCellH) / float32(FontAtlasH)
	u1 := float32((column+1)*FontCellW) / float32(FontAtlasW)
	v1 := float32((row+1)*FontCellH) / float32(FontAtlasH)

	w := float32(FontCellW) * scale
	h := float32(FontCellH) * scale

	cr := float32(col.R) / 255.0
	cg := float32(col.G) / 255.0
	cb := float32(col.B) / 255.0

	// Two triangles: TL, TR, BL then TR, BR, BL.
	r.textBuf = append(r.textBuf,
		sx, sy, u0, v0, cr, cg, cb, 1,
		sx+w, sy, u1, v0, cr, cg, cb, 1,
		sx, sy+h, u0, v1, cr, cg, cb, 1,
		sx+w, sy, u1, v0, cr, cg, cb, 1,
		sx+w, sy+h, u1, v1, cr, cg, cb, 1,
		sx, sy+h, u0, v1, cr, cg, cb, 1,
	)
}

// DrawString queues a string at screen pixel position (sx, sy) with given scale.
func (r *Renderer) DrawString(text string, sx, sy int, scale float32, col RGB) {
	advance := float32(FontCellW) * scale
	lineAdvance := float32(FontCellH) * scale
	baseX := float32(sx)
	x := float32(sx)
	y := float32(sy)
	for _, ch := range text {
		if ch == '\n' {
			x = baseX
			y += lineAdvance
			continue
		}
		r.DrawChar(ch, x, y, scale, col)
		x += advance
	}
}

// TextWidth returns the width in screen pixels of a string at given scale.
func TextWidth(text string, scale float32) int {
	lineLen := 0
	maxLineLen := 0
	for _, ch := range text {
		if ch == '\n' {
			if lineLen > maxLineLen {
				maxLineLen = lineLen
			}
			lineLen = 0
			continue
		}
		lineLen++
	}
	if lineLen > maxLineLen {
		maxLineLen = lineLen
	}
	return int(float32(maxLineLen*FontCellW) * scale)
}

// FlushText draws all buffered text quads and clears the buffer.
func (r *Renderer) FlushText(fbW, fbH int) {
	if len(r.textBuf) == 0 {
		return
	}

	gl.UseProgram(r.textProg)
	gl.BindVertexArray(r.textVAO)
	gl.BindBuffer(gl.ARRAY_BUFFER, r.textVBO)

	gl.Uniform2f(r.textURes, float32(fbW), float32(fbH))

	gl.ActiveTexture(gl.TEXTURE2)
	gl.BindTexture(gl.TEXTURE_2D, r.fontTex)

	gl.Enable(gl.BLEND)
	gl.BlendFunc(gl.SRC_ALPHA, gl.ONE_MINUS_SRC_ALPHA)

	count := len(r.textBuf) / 8
	gl.BufferData(gl.ARRAY_BUFFER, len(r.textBuf)*4, gl.Ptr(r.textBuf), gl.STREAM_DRAW)
	gl.DrawArrays(gl.TRIANGLES, 0, int32(count))

	gl.Disable(gl.BLEND)
	gl.ActiveTexture(gl.TEXTURE0)
	r.textBuf = r.textBuf[:0]
}

// DrawHealthBars renders small health bars above injured entities.
func (r *Renderer) DrawHealthBars(peds *PedestrianSystem, traffic *TrafficSystem, cam Camera, fbW, fbH int) {
	var buf []float32
	barWidth := 3

	// Ped health bars.
	for i := range peds.P {
		p := &peds.P[i]
		if !p.Alive {
			continue
		}
		frac := p.HP.Fraction()
		col := HealthBarColor(frac)
		bx := float32(p.X) - float32(barWidth)*0.5
		by := float32(p.Y) - 2.0
		filled := int(float64(barWidth) * frac)
		if filled < 1 {
			filled = 1
		}
		for px := 0; px < barWidth; px++ {
			var c RGB
			if px < filled {
				c = col
			} else {
				c = RGB{R: 40, G: 40, B: 40}
			}
			buf = append(buf,
				bx+float32(px), by, 1.0,
				float32(c.R)/255.0, float32(c.G)/255.0, float32(c.B)/255.0, 0.9, 0,
			)
		}
	}

	// Car health bars.
	for i := range traffic.Cars {
		c := &traffic.Cars[i]
		if !c.Alive {
			continue
		}
		frac := c.HP.Fraction()
		col := HealthBarColor(frac)
		bx := float32(c.X) - float32(barWidth)*0.5
		by := float32(c.Y) - float32(c.Size) - 1.0
		filled := int(float64(barWidth) * frac)
		if filled < 1 {
			filled = 1
		}
		for px := 0; px < barWidth; px++ {
			var cc RGB
			if px < filled {
				cc = col
			} else {
				cc = RGB{R: 40, G: 40, B: 40}
			}
			buf = append(buf,
				bx+float32(px), by, 1.0,
				float32(cc.R)/255.0, float32(cc.G)/255.0, float32(cc.B)/255.0, 0.9, 0,
			)
		}
	}

	if len(buf) > 0 {
		r.DrawSprites(buf, cam, fbW, fbH, false)
	}
}
