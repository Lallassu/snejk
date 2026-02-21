//go:build android

package game

import (
	"bytes"
	"fmt"
	"image"
	"image/draw"
	"image/png"

	"golang.org/x/mobile/gl"
)

const mobileTextScaleBoost = float32(1.5)

const textVertSrcMobile = `
attribute vec2 aPos;
attribute vec2 aUV;
attribute vec4 aColor;
uniform vec2 uResolution;
varying vec2 vUV;
varying vec4 vColor;
void main() {
  vec2 ndc = (aPos / uResolution) * 2.0 - 1.0;
  ndc.y = -ndc.y;
  gl_Position = vec4(ndc, 0.0, 1.0);
  vUV = aUV;
  vColor = aColor;
}`

const textFragSrcMobile = `
precision mediump float;
uniform sampler2D uFontTex;
varying vec2 vUV;
varying vec4 vColor;
void main() {
  vec4 t = texture2D(uFontTex, vUV);
  if (t.a < 0.01) discard;
  gl_FragColor = vec4(t.rgb * vColor.rgb, t.a * vColor.a);
}`

func (g *mobileGame) initTextGL(glctx gl.Context) error {
	img, err := png.Decode(bytes.NewReader(fontPNGMobile))
	if err != nil {
		return fmt.Errorf("decode font_alt.png: %w", err)
	}
	b := img.Bounds()
	nrgba, ok := img.(*image.NRGBA)
	if !ok || nrgba.Stride != b.Dx()*4 {
		nrgba = image.NewNRGBA(b)
		draw.Draw(nrgba, b, img, b.Min, draw.Src)
	}

	g.textFont = glctx.CreateTexture()
	glctx.BindTexture(gl.TEXTURE_2D, g.textFont)
	glctx.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.NEAREST)
	glctx.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.NEAREST)
	glctx.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_S, gl.CLAMP_TO_EDGE)
	glctx.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_T, gl.CLAMP_TO_EDGE)
	glctx.TexImage2D(gl.TEXTURE_2D, 0, int(gl.RGBA), b.Dx(), b.Dy(), gl.RGBA, gl.UNSIGNED_BYTE, nrgba.Pix)

	g.textProg, err = linkProgram(glctx, textVertSrcMobile, textFragSrcMobile)
	if err != nil {
		return fmt.Errorf("text program: %w", err)
	}
	g.textVBO = glctx.CreateBuffer()

	g.textAPos = glctx.GetAttribLocation(g.textProg, "aPos")
	g.textAUV = glctx.GetAttribLocation(g.textProg, "aUV")
	g.textACol = glctx.GetAttribLocation(g.textProg, "aColor")
	g.textURes = glctx.GetUniformLocation(g.textProg, "uResolution")
	g.textUFont = glctx.GetUniformLocation(g.textProg, "uFontTex")
	glctx.UseProgram(g.textProg)
	glctx.Uniform1i(g.textUFont, 2)
	return nil
}

func (g *mobileGame) destroyTextGL(glctx gl.Context) {
	if g.textVBO != (gl.Buffer{}) {
		glctx.DeleteBuffer(g.textVBO)
	}
	if g.textFont != (gl.Texture{}) {
		glctx.DeleteTexture(g.textFont)
	}
	if g.textProg != (gl.Program{}) {
		glctx.DeleteProgram(g.textProg)
	}
}

func (g *mobileGame) drawCharMobile(ch rune, sx, sy, scale float32, col RGB) {
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

	g.textBuf = append(g.textBuf,
		sx, sy, u0, v0, cr, cg, cb, 1,
		sx+w, sy, u1, v0, cr, cg, cb, 1,
		sx, sy+h, u0, v1, cr, cg, cb, 1,
		sx+w, sy, u1, v0, cr, cg, cb, 1,
		sx+w, sy+h, u1, v1, cr, cg, cb, 1,
		sx, sy+h, u0, v1, cr, cg, cb, 1,
	)
}

func (g *mobileGame) drawStringMobile(text string, sx, sy int, scale float32, col RGB) {
	scale *= mobileTextScaleBoost
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
		g.drawCharMobile(ch, x, y, scale, col)
		x += advance
	}
}

func (g *mobileGame) flushTextGL(glctx gl.Context, fbW, fbH int) {
	if len(g.textBuf) == 0 {
		return
	}
	const stride = 8 * 4
	glctx.UseProgram(g.textProg)
	glctx.BindBuffer(gl.ARRAY_BUFFER, g.textVBO)
	glctx.BufferData(gl.ARRAY_BUFFER, f32bytes(g.textBuf), gl.STREAM_DRAW)

	glctx.EnableVertexAttribArray(g.textAPos)
	glctx.EnableVertexAttribArray(g.textAUV)
	glctx.EnableVertexAttribArray(g.textACol)
	glctx.VertexAttribPointer(g.textAPos, 2, gl.FLOAT, false, stride, 0)
	glctx.VertexAttribPointer(g.textAUV, 2, gl.FLOAT, false, stride, 8)
	glctx.VertexAttribPointer(g.textACol, 4, gl.FLOAT, false, stride, 16)

	glctx.Uniform2f(g.textURes, float32(fbW), float32(fbH))
	glctx.ActiveTexture(gl.TEXTURE2)
	glctx.BindTexture(gl.TEXTURE_2D, g.textFont)
	glctx.Uniform1i(g.textUFont, 2)

	glctx.Enable(gl.BLEND)
	glctx.BlendFunc(gl.SRC_ALPHA, gl.ONE_MINUS_SRC_ALPHA)
	glctx.DrawArrays(gl.TRIANGLES, 0, len(g.textBuf)/8)
	glctx.Disable(gl.BLEND)
	glctx.ActiveTexture(gl.TEXTURE0)
	g.textBuf = g.textBuf[:0]
}

func TextWidth(text string, scale float32) int {
	scale *= mobileTextScaleBoost
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

func textLineCount(text string) int {
	if len(text) == 0 {
		return 0
	}
	lines := 1
	for _, ch := range text {
		if ch == '\n' {
			lines++
		}
	}
	return lines
}

func TextHeight(text string, scale float32) int {
	lines := textLineCount(text)
	if lines <= 0 {
		return 0
	}
	scale *= mobileTextScaleBoost
	return int(float32(lines*FontCellH) * scale)
}

func mobileUISp(px int) int {
	return int(float32(px)*mobileTextScaleBoost + 0.5)
}

func repeatChar(ch byte, n int) string {
	if n <= 0 {
		return ""
	}
	b := make([]byte, n)
	for i := range b {
		b[i] = ch
	}
	return string(b)
}

func (g *mobileGame) renderHUDMobile(glctx gl.Context, fbW, fbH int) {
	session := g.session
	peds := g.peds
	snake := g.snake
	white := RGB{R: 255, G: 255, B: 255}
	green := RGB{R: 100, G: 255, B: 100}
	red := RGB{R: 255, G: 80, B: 80}
	yellow := RGB{R: 255, G: 255, B: 100}
	blue := RGB{R: 60, G: 140, B: 255}

	switch session.State {
	case StateMenu:
		title := "QAKE THE SNAKE"
		titleScale := float32(3.0)
		msg := "TOUCH"
		msgScale := float32(1.0)
		hint := "Eat humans to grow"
		hintScale := float32(0.65)

		titleH := TextHeight(title, titleScale)
		msgH := TextHeight(msg, msgScale)
		hintH := TextHeight(hint, hintScale)
		gap1 := mobileUISp(14)
		gap2 := mobileUISp(10)
		totalH := titleH + gap1 + msgH + gap2 + hintH
		topY := fbH/2 - totalH/2
		minTop := mobileUISp(12)
		if topY < minTop {
			topY = minTop
		}

		titleY := topY
		msgY := titleY + titleH + gap1
		hintY := msgY + msgH + gap2
		g.drawStringMobile(title, fbW/2-TextWidth(title, titleScale)/2, titleY, titleScale, green)
		g.drawStringMobile(msg, fbW/2-TextWidth(msg, msgScale)/2, msgY, msgScale, white)
		g.drawStringMobile(hint, fbW/2-TextWidth(hint, hintScale)/2, hintY, hintScale, yellow)

	case StatePlaying:
		hudMul := float32(1.30)
		hs := func(v float32) float32 { return v * hudMul }
		s := hs(0.75)
		topPad := mobileUISp(8)
		scoreStr := fmt.Sprintf("Score: %d", session.Score)
		g.drawStringMobile(scoreStr, topPad, topPad, s, white)
		timeStr := fmt.Sprintf("%.1fs", session.LevelTimer)
		g.drawStringMobile(timeStr, fbW/2-TextWidth(timeStr, s)/2, topPad, s, white)
		if peds != nil {
			remaining := peds.AliveCount()
			pedStr := fmt.Sprintf("Humans: %d", remaining)
			g.drawStringMobile(pedStr, fbW-TextWidth(pedStr, s)-topPad, topPad, s, green)
		}
		if snake != nil {
			barScale := hs(0.85)
			const barChars = 16
			barY := fbH - mobileUISp(56)
			barX := mobileUISp(10)
			filledStars := int(snake.WantedLevel)
			wantedStr := repeatChar('*', filledStars) + repeatChar('.', 5-filledStars)
			wantedLabel := fmt.Sprintf("WANTED [%s]", wantedStr)
			wantedCol := blue
			if snake.WantedLevel >= WantedMax {
				wantedCol = red
			} else if snake.WantedLevel >= WantedTier2 {
				wantedCol = yellow
			}
			g.drawStringMobile("WANTED", barX, barY-mobileUISp(22), hs(0.65), blue)
			g.drawStringMobile(fmt.Sprintf("[%s]", wantedStr), barX, barY, barScale, wantedCol)

			hpFrac := snake.HP.Fraction()
			hpBar := fmt.Sprintf("[%-*s]", barChars, repeatChar('#', int(float64(barChars)*hpFrac)))
			hpCol := HealthBarColor(hpFrac)
			hpBarX := barX + TextWidth(wantedLabel, barScale) + 8
			g.drawStringMobile("HP", hpBarX, barY-mobileUISp(22), hs(0.65), white)
			g.drawStringMobile(hpBar, hpBarX, barY, barScale, hpCol)
			if snake.EvoLevel > 0 {
				evoStr := fmt.Sprintf("EVO %d", snake.EvoLevel)
				evoCol := snake.evoHeadColor()
				evoX := hpBarX + TextWidth(hpBar, barScale) + 16
				g.drawStringMobile(evoStr, evoX, barY, hs(0.75), evoCol)
			}
			if snake.FlamethrowerTimer > 0 {
				fireStr := fmt.Sprintf("FIRE %.1fs", snake.FlamethrowerTimer)
				fireScale := hs(0.75)
				fireX := fbW/2 - TextWidth(fireStr, fireScale)/2
				g.drawStringMobile(fireStr, fireX, fbH-mobileUISp(74), fireScale, RGB{R: 255, G: 120, B: 30})
			}
			if snake.SpeedBoost > 0 {
				boostStr := fmt.Sprintf("BOOST %.1fs", snake.SpeedBoost)
				boostScale := hs(0.65)
				g.drawStringMobile(boostStr, fbW/2-TextWidth(boostStr, boostScale)/2, fbH-mobileUISp(28), boostScale, yellow)
			}
			if snake.PowerupTimer > 0 {
				alpha := snake.PowerupTimer
				if alpha > 1.0 {
					alpha = 1.0
				}
				col := RGB{
					R: uint8(float64(snake.PowerupCol.R) * alpha),
					G: uint8(float64(snake.PowerupCol.G) * alpha),
					B: uint8(float64(snake.PowerupCol.B) * alpha),
				}
				popScale := hs(1.2)
				popStr := snake.PowerupMsg
				g.drawStringMobile(popStr, fbW/2-TextWidth(popStr, popScale)/2, fbH/2-80, popScale, col)
			}
			if snake.KillMsgTimer > 0 {
				alpha := snake.KillMsgTimer
				if alpha > 1.0 {
					alpha = 1.0
				}
				kcol := RGB{
					R: uint8(float64(snake.KillMsgCol.R) * alpha),
					G: uint8(float64(snake.KillMsgCol.G) * alpha),
					B: uint8(float64(snake.KillMsgCol.B) * alpha),
				}
				killScale := hs(1.6)
				killStr := snake.KillMsg
				g.drawStringMobile(killStr, fbW/2-TextWidth(killStr, killScale)/2, fbH/2+mobileUISp(12), killScale, kcol)
			}
		}

	case StateLevelComplete:
		msg1 := "LEVEL COMPLETE!"
		msg2 := fmt.Sprintf("Level %d - Score: %d   Time: %.1fs", session.CurrentLevel, session.Score, session.LevelTimer)
		next := "Tap for next level"
		s1 := float32(1.5)
		s2 := float32(0.75)
		s3 := float32(0.75)
		h1 := TextHeight(msg1, s1)
		h2 := TextHeight(msg2, s2)
		h3 := TextHeight(next, s3)
		gapA := mobileUISp(14)
		gapB := mobileUISp(12)
		totalH := h1 + gapA + h2 + gapB + h3
		topY := fbH/2 - totalH/2
		g.drawStringMobile(msg1, fbW/2-TextWidth(msg1, s1)/2, topY, s1, green)
		y2 := topY + h1 + gapA
		g.drawStringMobile(msg2, fbW/2-TextWidth(msg2, s2)/2, y2, s2, white)
		y3 := y2 + h2 + gapB
		g.drawStringMobile(next, fbW/2-TextWidth(next, s3)/2, y3, s3, white)

	case StateLevelFailed:
		msg1 := "GAME OVER"
		msg2 := fmt.Sprintf("Final Score: %d", session.Score)
		msg3 := "Tap to retry"
		s1 := float32(2.0)
		s2 := float32(0.9)
		s3 := float32(0.75)
		h1 := TextHeight(msg1, s1)
		h2 := TextHeight(msg2, s2)
		h3 := TextHeight(msg3, s3)
		gapA := mobileUISp(12)
		gapB := mobileUISp(12)
		totalH := h1 + gapA + h2 + gapB + h3
		topY := fbH/2 - totalH/2
		g.drawStringMobile(msg1, fbW/2-TextWidth(msg1, s1)/2, topY, s1, red)
		y2 := topY + h1 + gapA
		g.drawStringMobile(msg2, fbW/2-TextWidth(msg2, s2)/2, y2, s2, yellow)
		y3 := y2 + h2 + gapB
		g.drawStringMobile(msg3, fbW/2-TextWidth(msg3, s3)/2, y3, s3, white)
	}

	g.flushTextGL(glctx, fbW, fbH)
}
