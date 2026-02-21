//go:build !android

package game

import "fmt"

// RenderHUD draws all in-game UI elements using the font atlas.
func RenderHUD(r *Renderer, session *GameSession, peds *PedestrianSystem, snake *Snake, fbW, fbH int) {
	white := RGB{R: 255, G: 255, B: 255}
	green := RGB{R: 100, G: 255, B: 100}
	red := RGB{R: 255, G: 80, B: 80}
	yellow := RGB{R: 255, G: 255, B: 100}
	blue := RGB{R: 60, G: 140, B: 255}

	switch session.State {
	case StateMenu:
		title := "QAKE THE SNAKE"
		titleScale := float32(3.0)
		r.DrawString(title, fbW/2-TextWidth(title, titleScale)/2, fbH/2-80, titleScale, green)

		msg := "Press SPACE to Start"
		msgScale := float32(1.0)
		r.DrawString(msg, fbW/2-TextWidth(msg, msgScale)/2, fbH/2+20, msgScale, white)

		hint := "Eat humans to grow"
		hintScale := float32(0.65)
		r.DrawString(hint, fbW/2-TextWidth(hint, hintScale)/2, fbH/2+55, hintScale, yellow)

	case StatePlaying:
		hudMul := float32(1.14)
		hs := func(v float32) float32 { return v * hudMul }
		s := hs(0.75)

		// Top-left: score only (no level/theme name).
		scoreStr := fmt.Sprintf("Score: %d", session.Score)
		r.DrawString(scoreStr, 8, 8, s, white)

		// Top-center: timer.
		timeStr := fmt.Sprintf("%.1fs", session.LevelTimer)
		r.DrawString(timeStr, fbW/2-TextWidth(timeStr, s)/2, 8, s, white)

		// Top-right: peds remaining.
		if peds != nil {
			remaining := peds.AliveCount()
			pedStr := fmt.Sprintf("Humans: %d", remaining)
			r.DrawString(pedStr, fbW-TextWidth(pedStr, s)-8, 8, s, green)
		}

		// Bottom: wanted stars + HP bar.
		if snake != nil {
			barScale := hs(0.85)
			const barChars = 16
			barY := fbH - 50
			barX := 10

			// Wanted level: 5 stars, filled based on WantedLevel.
			filledStars := int(snake.WantedLevel)
			wantedStr := repeatChar('*', filledStars) + repeatChar('.', 5-filledStars)
			wantedLabel := fmt.Sprintf("WANTED [%s]", wantedStr)
			wantedCol := blue
			if snake.WantedLevel >= WantedMax {
				wantedCol = red
			} else if snake.WantedLevel >= WantedTier2 {
				wantedCol = yellow
			}
			r.DrawString("WANTED", barX, barY-22, hs(0.65), blue)
			r.DrawString(fmt.Sprintf("[%s]", wantedStr), barX, barY, barScale, wantedCol)

			// HP bar.
			hpFrac := snake.HP.Fraction()
			hpBar := fmt.Sprintf("[%-*s]", barChars, repeatChar('#', int(float64(barChars)*hpFrac)))
			hpCol := HealthBarColor(hpFrac)
			hpBarX := barX + TextWidth(wantedLabel, barScale) + 8
			r.DrawString("HP", hpBarX, barY-22, hs(0.65), white)
			r.DrawString(hpBar, hpBarX, barY, barScale, hpCol)

			// Evolution level indicator.
			if snake.EvoLevel > 0 {
				evoStr := fmt.Sprintf("EVO %d", snake.EvoLevel)
				evoCol := snake.evoHeadColor()
				evoX := hpBarX + TextWidth(hpBar, barScale) + 16
				r.DrawString(evoStr, evoX, barY, hs(0.75), evoCol)
			}

			// Flamethrower indicator.
			if snake.FlamethrowerTimer > 0 {
				fireStr := fmt.Sprintf("FIRE %.1fs", snake.FlamethrowerTimer)
				fireScale := hs(0.75)
				fireX := fbW/2 - TextWidth(fireStr, fireScale)/2
				r.DrawString(fireStr, fireX, fbH-68, fireScale, RGB{R: 255, G: 120, B: 30})
			}

			// Speed boost indicator.
			if snake.SpeedBoost > 0 {
				boostStr := fmt.Sprintf("BOOST %.1fs", snake.SpeedBoost)
				boostScale := hs(0.65)
				r.DrawString(boostStr, fbW/2-TextWidth(boostStr, boostScale)/2, fbH-26, boostScale, yellow)
			}

			// Powerup label: visible for 2.5s, fades out over last second.
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
				popScale := float32(1.2)
				popStr := snake.PowerupMsg
				r.DrawString(popStr, fbW/2-TextWidth(popStr, popScale)/2, fbH/2-80, popScale, col)
			}

			// Kill streak combo label: fades independently below the powerup area.
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
				killScale := float32(1.6)
				killStr := snake.KillMsg
				r.DrawString(killStr, fbW/2-TextWidth(killStr, killScale)/2, fbH/2+12, killScale, kcol)
			}
		}

	case StateLevelComplete:
		msg1 := "LEVEL COMPLETE!"
		r.DrawString(msg1, fbW/2-TextWidth(msg1, 1.5)/2, fbH/2-80, 1.5, green)

		msg2 := fmt.Sprintf("Level %d — Score: %d   Time: %.1fs", session.CurrentLevel, session.Score, session.LevelTimer)
		r.DrawString(msg2, fbW/2-TextWidth(msg2, 0.75)/2, fbH/2-20, 0.75, white)

		next := "Press SPACE for next level"
		r.DrawString(next, fbW/2-TextWidth(next, 0.75)/2, fbH/2+40, 0.75, white)

	case StateLevelFailed:
		msg1 := "GAME OVER"
		r.DrawString(msg1, fbW/2-TextWidth(msg1, 2.0)/2, fbH/2-60, 2.0, red)

		msg2 := fmt.Sprintf("Final Score: %d", session.Score)
		r.DrawString(msg2, fbW/2-TextWidth(msg2, 0.9)/2, fbH/2, 0.9, yellow)

		msg3 := "Press SPACE to retry"
		r.DrawString(msg3, fbW/2-TextWidth(msg3, 0.75)/2, fbH/2+50, 0.75, white)
	}

	r.FlushText(fbW, fbH)
}

// repeatChar returns a string of n copies of ch.
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
