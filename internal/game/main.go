//go:build !android

package game

import (
	"fmt"
	"math"
	"os"
	"runtime"
	"strconv"
	"time"

	"github.com/go-gl/gl/v4.1-core/gl"
	"github.com/go-gl/glfw/v3.3/glfw"
)

func RunDesktop() {
	runtime.LockOSThread()

	window, err := initWindow()
	if err != nil {
		panic(err)
	}
	defer glfw.Terminate()
	defer window.Destroy()

	if err := gl.Init(); err != nil {
		panic(fmt.Errorf("gl init: %w", err))
	}

	// Initialize audio system.
	if err := InitAudio(); err != nil {
		fmt.Fprintf(os.Stderr, "audio init failed (continuing without sound): %v\n", err)
	} else {
		// Start menu music.
		go func() {
			time.Sleep(100 * time.Millisecond) // let audio context initialize
			StartMenuMusic()
		}()
	}

	// Seed from environment or clock.
	seed := uint64(time.Now().UnixNano())
	if s := os.Getenv("SNAKE_SEED"); s != "" {
		if v, err := strconv.ParseUint(s, 10, 64); err == nil {
			seed = v
		}
	}

	// GL state.
	gl.Disable(gl.DEPTH_TEST)
	gl.Disable(gl.CULL_FACE)
	gl.Enable(gl.PROGRAM_POINT_SIZE)
	gl.PixelStorei(gl.UNPACK_ALIGNMENT, 1)
	gl.ClearColor(
		float32(Palette.Lot.R)/255.0,
		float32(Palette.Lot.G)/255.0,
		float32(Palette.Lot.B)/255.0,
		1.0,
	)

	// World generation — random themed variant from the start.
	world := NewWorld(seed)
	startTheme, _ := PickLevelTheme(seed, 1, 0, -1)
	world.Theme = startTheme
	world.GenerateAll()
	world.BuildSpatialIndex()
	for cy := 0; cy <= world.maxCy; cy++ {
		for cx := 0; cx <= world.maxCx; cx++ {
			c := world.GetChunk(cx, cy)
			if c != nil {
				c.RecomputeShadows(world)
			}
		}
	}

	// Renderer.
	rend, err := NewRenderer()
	if err != nil {
		panic(fmt.Errorf("renderer: %w", err))
	}
	defer rend.Destroy()
	rend.InitCarTextures()
	if err := rend.InitFont(); err != nil {
		panic(fmt.Errorf("font: %w", err))
	}

	// Systems.
	peds := NewPedestrianSystem(400, seed^0xFED)
	traffic := NewTrafficSystem(seed ^ 0xCAFE)
	particles := NewParticleSystem(MaxParticles, seed^0xBEAD)
	weather := NewWeatherSystem(seed ^ 0x57A7)
	bonuses := NewBonusSystem(seed^0xB0B, 5)
	cops := NewCopSystem(seed ^ 0xC095)
	mil := NewMilitarySystem(seed ^ 0xA7A1)

	// Game session.
	session := NewGameSession()
	_ = NewEventBus()

	// Snake (nil until game starts).
	var snake *Snake

	cam := Camera{
		X:    float64(WorldWidth) / 2,
		Y:    float64(WorldHeight) / 2,
		Zoom: DefaultZoom,
	}
	input := NewInput()

	// Reusable render buffers.
	var glowBuf, normBuf []float32
	var carShadowBuf, carHeadBuf []float32

	last := glfw.GetTime()
	for !window.ShouldClose() {
		now := glfw.GetTime()
		dt := now - last
		last = now
		if dt > 0.1 {
			dt = 0.1
		}

		glfw.PollEvents()
		if window.GetKey(glfw.KeyEscape) == glfw.Press {
			window.SetShouldClose(true)
			continue
		}

		fbW, fbH := window.GetFramebufferSize()
		if fbW <= 0 || fbH <= 0 {
			continue
		}

		// State transitions.
		switch session.State {
		case StateMenu:
			if input.JustPressed(window, glfw.KeySpace) {
				PlaySound(SoundMenuSelect)
				StartLevelMusic(1)
				session.StartLevel(1, world, peds, traffic, bonuses, cops, mil, &snake, particles, seed)
				weather.Configure(session.Weather, session.WeatherSeed)
			}

		case StatePlaying:
			targetSelecting := false
			if snake != nil && snake.Alive && snake.TargetNukeTimer > 0 {
				targetSelecting = true
				snake.TargetNukeTimer -= dt
				rem := snake.TargetNukeTimer
				if rem < 0 {
					rem = 0
				}
				snake.PowerupTimer = 1.1
				snake.PowerupMsg, snake.PowerupCol = snake.TargetingPrompt(rem)

				if input.JustClicked(window, glfw.MouseButtonLeft) {
					targetCam := cam
					tx, ty := cam.EffectivePos()
					targetCam.X = tx
					targetCam.Y = ty
					mx, my := CursorWorldPos(window, targetCam, fbW, fbH)
					wx := clamp(int(math.Round(mx)), 0, WorldWidth-1)
					wy := clamp(int(math.Round(my)), 0, WorldHeight-1)
					snake.ActivateTargetAbilityAt(wx, wy, world, peds, traffic, particles, &cam, cops, mil)
				} else if snake.TargetNukeTimer <= 0 {
					snake.TargetNukeTimer = 0
					snake.PowerupMsg, snake.PowerupCol = snake.TargetingLockLostPrompt()
					snake.PowerupTimer = 1.6
				}
			}

			if !targetSelecting {
				// Steer snake (player input only when AI mode is not active).
				if snake != nil && snake.Alive {
					if snake.AITimer <= 0 {
						steer, idle := SnakeSteerTarget(window, input, snake, cam, fbW, fbH)
						hx, hy := snake.Head()
						// Auto-steer toward nearby bonus boxes.
						if bonuses != nil {
							bestDist := 5.0 // attraction range
							for i := range bonuses.Boxes {
								b := &bonuses.Boxes[i]
								if !b.Alive {
									continue
								}
								d := math.Hypot(b.X-hx, b.Y-hy)
								if d < bestDist {
									bestDist = d
									steer = math.Atan2(b.Y-hy, b.X-hx)
									idle = false
								}
							}
						}
						// Bounce override: while escaping a wall, hold the escape
						// heading so Steer() can't immediately re-enter the obstacle.
						if snake.BounceTimer > 0 {
							steer = snake.BounceDir
							idle = false
						} else if !idle && world != nil {
							// Proactive wall avoidance: look ahead and steer around
							// buildings before the snake reaches them.
							steer = WallAvoidAngle(hx, hy, steer, world)
						}
						snake.Idle = idle
						if !idle {
							snake.Steer(steer, dt)
						}
					} else {
						snake.Idle = false
					}
					// Flamethrower is automatic, no input needed.
					snake.Update(dt, world, peds, traffic, bonuses, particles, &cam, cops, mil)
				}

				// Update systems.
				session.Update(dt)
				cam.UpdateShake(dt, seed^uint64(now*1000))
				world.Update(dt)
				UpdateBurnVisuals(world, particles, dt)
				peds.Update(dt, world, snake, particles)
				sunAmbNow, _, _, _ := SunCycleLight(session.LevelTimer)
				traffic.NightFactor = NightIntensityFromAmbient(sunAmbNow)
				traffic.Update(dt, world, particles, peds, &cam)
				weather.UpdateAndSpawn(particles, dt)
				particles.UpdateWithShockwaveDamage(dt, world, peds, cops, mil)
				snakeHP := 1.0
				if snake != nil {
					snakeHP = snake.HP.Fraction()
				}
				bonuses.Update(dt, peds.AliveCount(), snakeHP)
				bonuses.SpawnSparks(particles, dt)

				cops.Update(dt, snake, world, particles, &cam, now)
				mil.Update(dt, snake, world, particles, &cam, now)

				// Cleanup dead entities.
				peds.RemoveDead()
				traffic.RemoveDead()
				cops.RemoveDead()
				mil.RemoveDead()

				session.CheckLevelEnd(peds, snake)
			}

		case StateLevelComplete:
			if input.JustPressed(window, glfw.KeySpace) {
				nextLevel := session.CurrentLevel + 1
				StartLevelMusic(nextLevel)
				session.StartLevel(nextLevel, world, peds, traffic, bonuses, cops, mil, &snake, particles, seed)
				weather.Configure(session.Weather, session.WeatherSeed)
			}
			particles.Update(dt, world)

		case StateLevelFailed:
			if input.JustPressed(window, glfw.KeySpace) {
				StartLevelMusic(session.CurrentLevel)
				session.StartLevel(session.CurrentLevel, world, peds, traffic, bonuses, cops, mil, &snake, particles, seed)
				weather.Configure(session.Weather, session.WeatherSeed)
			}
			particles.Update(dt, world)
		}

		// Always fit the full world on screen.
		UpdateAutoCamera(&cam, snake, dt, fbW, fbH)

		// Render with shake applied.
		renderCam := cam
		sx, sy := cam.EffectivePos()
		renderCam.X = sx
		renderCam.Y = sy

		// Sun cycle: compute lighting and shadow parameters from game time.
		sunAmb, sunTR, sunTG, sunTB := SunCycleLight(session.LevelTimer)
		sunAngle, sunSlope := SunCycleShadow(session.LevelTimer)
		world.UpdateSun(sunAngle, sunSlope)
		gl.ClearColor(
			float32(Palette.Lot.R)/255.0*sunAmb*sunTR,
			float32(Palette.Lot.G)/255.0*sunAmb*sunTG,
			float32(Palette.Lot.B)/255.0*sunAmb*sunTB,
			1.0,
		)

		rend.BeginFrame(renderCam, fbW, fbH)
		rend.SetSunLight(sunAmb, sunTR, sunTG, sunTB)
		rend.DrawChunks(world, renderCam, fbW, fbH)
		carShadowBuf = CarShadowSprites(traffic, cops, carShadowBuf)
		if len(carShadowBuf) > 0 {
			rend.DrawSprites(carShadowBuf, renderCam, fbW, fbH, false)
		}
		rend.DrawNPCCars(traffic, renderCam, fbW, fbH)
		rend.DrawCopCars(cops, renderCam, fbW, fbH)
		rend.DrawPedestrians(peds, renderCam, fbW, fbH, now)
		if copBuf := cops.CopRenderData(now); len(copBuf) > 0 {
			rend.DrawSprites(copBuf, renderCam, fbW, fbH, false)
		}
		if copGlow := cops.CopGlowData(now); len(copGlow) > 0 {
			rend.DrawGlowSprites(copGlow, renderCam, fbW, fbH)
		}
		if milBuf := mil.RenderData(now); len(milBuf) > 0 {
			rend.DrawSprites(milBuf, renderCam, fbW, fbH, false)
		}
		if milGlow := mil.GlowData(now); len(milGlow) > 0 {
			rend.DrawGlowSprites(milGlow, renderCam, fbW, fbH)
		}

		// Draw snake body as point sprites, plus bomb/vacuum glow effects.
		if snake != nil && snake.Alive {
			snakeBuf := snake.SnakeRenderData()
			if len(snakeBuf) > 0 {
				rend.SetSpriteAmbient(1.0, 1.0, 1.0, 1.0)
				rend.DrawSprites(snakeBuf, renderCam, fbW, fbH, false)
				rend.SetSpriteAmbient(sunAmb, sunTR, sunTG, sunTB)
			}
			if glowBuf := snake.GlowData(); len(glowBuf) > 0 {
				rend.DrawGlowSprites(glowBuf, renderCam, fbW, fbH)
			}
		}
		// Tactical nuke targeting marker under cursor while time remains.
		if snake != nil && snake.Alive && snake.TargetNukeTimer > 0 {
			mx, my := CursorWorldPos(window, renderCam, fbW, fbH)
			mx = clampF(mx, 0, float64(WorldWidth-1))
			my = clampF(my, 0, float64(WorldHeight-1))
			pulse := float32(1.0 + 0.15*math.Sin(now*9.0))
			outerR, outerG, outerB := float32(0.95), float32(0.22), float32(0.08)
			innerR, innerG, innerB := float32(1.0), float32(0.92), float32(0.25)
			switch snake.TargetAbilityKind {
			case BonusTargetWorms:
				outerR, outerG, outerB = 0.22, 0.92, 0.35
				innerR, innerG, innerB = 0.75, 1.0, 0.60
			case BonusTargetGunship:
				outerR, outerG, outerB = 1.0, 0.24, 0.20
				innerR, innerG, innerB = 1.0, 0.70, 0.28
			case BonusTargetHeliMissile:
				outerR, outerG, outerB = 1.0, 0.45, 0.18
				innerR, innerG, innerB = 1.0, 0.78, 0.24
			case BonusTargetBombBelt:
				outerR, outerG, outerB = 1.0, 0.60, 0.22
				innerR, innerG, innerB = 1.0, 0.84, 0.40
			case BonusTargetAirSupport:
				outerR, outerG, outerB = 0.38, 0.70, 1.0
				innerR, innerG, innerB = 0.74, 0.90, 1.0
			case BonusTargetPigs:
				outerR, outerG, outerB = 1.0, 0.40, 0.66
				innerR, innerG, innerB = 1.0, 0.70, 0.84
			case BonusTargetCars:
				outerR, outerG, outerB = 1.0, 0.46, 0.24
				innerR, innerG, innerB = 1.0, 0.72, 0.42
			case BonusTargetSnakes:
				outerR, outerG, outerB = 0.34, 0.95, 0.30
				innerR, innerG, innerB = 0.74, 1.0, 0.66
			}
			nukeAim := []float32{
				float32(mx), float32(my), 18.0 * pulse, outerR, outerG, outerB, 0.9, 0,
				float32(mx), float32(my), 9.0 * pulse, innerR, innerG, innerB, 1.0, 0,
				float32(mx), float32(my), 2.8, 1.0, 1.0, 1.0, 1.0, 0,
			}
			rend.DrawGlowSprites(nukeAim, renderCam, fbW, fbH)
		}

		// Draw bonus boxes: rotated crate sprites + soft color glow.
		bonusBuf := bonuses.RenderData()
		if len(bonusBuf) > 0 {
			rend.DrawBonusSprites(bonusBuf, renderCam, fbW, fbH)
			rend.DrawGlowSprites(bonuses.GlowData(), renderCam, fbW, fbH)
		}

		// Streetlights + car headlights: additive radial glow during dusk/night.
		lightBrightness := NightIntensityFromAmbient(sunAmb)
		if lightBrightness > 0.01 {
			if !world.Theme.NoRoads {
				rend.DrawGlowSprites(streetlightSprites(lightBrightness), renderCam, fbW, fbH)
			}
			carHeadBuf = CarHeadlightSprites(traffic, lightBrightness, carHeadBuf)
			if len(carHeadBuf) > 0 {
				rend.DrawGlowSprites(carHeadBuf, renderCam, fbW, fbH)
			}
		}

		// Particles: two passes (normal + glow).
		glowBuf, normBuf = particles.ParticleRenderData(glowBuf, normBuf)
		if len(normBuf) > 0 {
			rend.DrawSprites(normBuf, renderCam, fbW, fbH, false)
		}
		if len(glowBuf) > 0 {
			rend.SetSpriteAmbient(1.0, 1.0, 1.0, 1.0)
			rend.DrawSprites(glowBuf, renderCam, fbW, fbH, true)
			rend.SetSpriteAmbient(sunAmb, sunTR, sunTG, sunTB)
		}

		// HUD uses stable camera (no shake).
		RenderHUD(rend, session, peds, snake, fbW, fbH)

		rend.RestoreChunkProgram()
		window.SwapBuffers()
	}
}

// streetlightCache avoids rebuilding the streetlight sprite buffer every frame.
// Brightness changes gradually; we quantize to 1/200 steps (~0.5% granularity).
var streetlightCache struct {
	brightness float32
	buf        []float32
}

// streetlightSprites returns radial glow sprites for road intersection lights.
// RGB is pre-multiplied by brightness for additive blending via DrawGlowSprites.
func streetlightSprites(brightness float32) []float32 {
	// Quantize to avoid rebuilding on every tiny floating-point change.
	q := float32(int(brightness*200)) / 200.0
	if streetlightCache.buf != nil && streetlightCache.brightness == q {
		return streetlightCache.buf
	}
	buf := make([]float32, 0, 128)
	for y := 0; y+RoadWidth < WorldHeight; y += Pattern {
		for x := 0; x+RoadWidth < WorldWidth; x += Pattern {
			fx := float32(x + RoadWidth)
			fy := float32(y + RoadWidth)
			// Outer warm halo: large radius, soft orange-yellow, pre-multiplied.
			buf = append(buf, fx, fy, 10.0, 0.5*brightness, 0.42*brightness, 0.15*brightness, 1, 0)
			// Bright white-yellow core: small, full intensity.
			buf = append(buf, fx, fy, 2.0, 1.0*brightness, 1.0*brightness, 0.7*brightness, 1, 0)
		}
	}
	streetlightCache.brightness = q
	streetlightCache.buf = buf
	return buf
}

// Unused but kept for compatibility.
var _ = math.Pi
