//go:build android && audio_stub

package game

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

func InitAudio() error                               { return nil }
func PlaySound(kind SoundKind)                       {}
func PlaySoundWithGain(kind SoundKind, gain float64) {}
func PlayPoliceSirenSpatial(gain, doppler, pan float64) {
}
func PlayExplosionSound(magnitude float64) {}
func StartMenuMusic()                      {}
func StartBackgroundMusic()                {}
func StartLevelMusic(level int)            {}
func SetMusicVolume(vol float64)           {}
func SetSFXVolume(vol float64)             {}
