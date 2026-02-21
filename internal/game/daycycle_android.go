//go:build android

package game

import "math"

const (
	DayCyclePeriod = 90.0
	SunAmbientMin  = 0.38
	SunAmbientMax  = 1.00
	SunNightStart  = 0.65
)

func SunCycleLight(gameTime float64) (ambient, tintR, tintG, tintB float32) {
	phase := math.Mod(gameTime, DayCyclePeriod) / DayCyclePeriod
	sunHeight := math.Sin(phase * 2 * math.Pi)

	mid := float64(SunAmbientMin+SunAmbientMax) * 0.5
	amp := float64(SunAmbientMax-SunAmbientMin) * 0.5
	ambient = float32(mid + amp*sunHeight)

	horizonFactor := 1.0 - math.Abs(sunHeight)
	warmth := horizonFactor * horizonFactor * 0.35
	tintR = float32(1.0 + warmth*0.4)
	tintG = float32(1.0 - warmth*0.15)
	tintB = float32(1.0 - warmth*0.5)

	if sunHeight < -0.3 {
		nightFactor := float32((-sunHeight - 0.3) / 0.7)
		tintR -= nightFactor * 0.07
		tintG -= nightFactor * 0.035
		tintB += nightFactor * 0.10
	}
	return
}

func NightIntensityFromAmbient(ambient float32) float32 {
	denom := float64(SunNightStart - SunAmbientMin)
	if denom <= 0 {
		return 0
	}
	return float32(clampF((float64(SunNightStart)-float64(ambient))/denom, 0, 1))
}

func SunCycleShadow(gameTime float64) (angle, slope float64) {
	phase := math.Mod(gameTime, DayCyclePeriod) / DayCyclePeriod
	sunHeight := math.Sin(phase * 2 * math.Pi)
	angle = -phase * 2 * math.Pi
	if sunHeight > 0 {
		slope = 1.0 + sunHeight*2.0
	} else {
		slope = 1.0
	}
	return
}
