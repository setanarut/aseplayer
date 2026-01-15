package aseplayer

import (
	"io/fs"
	"os"
	"slices"
	"time"

	"github.com/askeladdk/aseprite"
	"github.com/hajimehoshi/ebiten/v2"
)

// AnimPlayer plays and manages Aseprite tag animations.
type AnimPlayer struct {
	// The frame of the animation currently being played
	CurrentFrame *ebiten.Image
	// The animation currently being played
	CurrentAnimation *Animation
	// Animations accessible by their Aseprite tag names
	Animations map[string]*Animation
	// Sprite atlas containing all animations
	Atlas *ebiten.Image
	// If true, the animation is paused
	Paused bool
	// Time elapsed since the current frame started displaying
	FrameElapsedTime time.Duration
	// Current frame index of the playing animation
	Index int
	// isJustEnded returns true only on the frame when the animation just ended
	isJustEnded bool
}

const fixedDelta = time.Second / 60

func (ap *AnimPlayer) Update() {
	if ap.Paused {
		return
	}
	ap.isJustEnded = false // Her update'te sıfırla
	a := ap.CurrentAnimation
	ap.FrameElapsedTime += fixedDelta
	if ap.FrameElapsedTime >= a.Durations[ap.Index] {
		ap.FrameElapsedTime = 0
		ap.Index++
		if ap.Index >= len(a.Frames) {
			ap.isJustEnded = true // ← Animasyon tam bitti
			ap.Index = 0
		}
	}
	ap.CurrentFrame = a.Frames[ap.Index]
}

// SetAnim sets the animation state and resets to the first frame.
//
// Do not call this in every Update() frame. Set it only once in the Enter/Exit events,
// otherwise, the animation will always reset to the first index.
func (ap *AnimPlayer) SetAnim(tag string) {
	ap.CurrentAnimation = ap.Animations[tag]
	ap.Index = 0
	ap.FrameElapsedTime = 0
}

// IsJustEnded returns true only on the frame when the animation just completed its last frame
//
// Use this for triggering events, transitions, or one-time effects.
func (ap *AnimPlayer) IsJustEnded() bool {
	return ap.isJustEnded
}

// CheckAndSetAnim changes the animation and resets to the first frame if the animation state is not the current state.
//
// It can be called on every Update() frame.
//
// For optimization, it is recommended to use SeAnim() only during state transitions.
func (ap *AnimPlayer) CheckAndSetAnim(tag string) {
	if tag != ap.CurrentAnimation.Tag {
		ap.CurrentAnimation = ap.Animations[tag]
		ap.Index = 0
		ap.FrameElapsedTime = 0
	}
}

// Animation for AnimPlayer
type Animation struct {
	// The animation tag name is identical to the Aseprite file
	Tag string
	// Animation frames
	Frames []*ebiten.Image
	// Frame durations retrieved from the Aseprite file
	Durations []time.Duration
}

// The first Aseprite tag will be assigned as CurrentAnimation. You can then set it with SetAnim()
//
// Do not read .ase files that do not have a tag.
func NewAnimPlayerFromAsepriteFileSystem(fs fs.FS, asePath string) *AnimPlayer {
	ase := newAseFromFileSystem(fs, asePath)
	ap := fromAseprite(ase)
	ase = nil
	return ap
}

// The first Aseprite tag will be assigned as CurrentAnimation. You can then set it with SetAnim()
//
// Do not read .ase files that do not have a tag.
func NewAnimPlayerFromAsepriteFile(asePath string) *AnimPlayer {
	ase := newAseFromFile(asePath)
	ap := fromAseprite(ase)
	ase = nil
	return ap
}

func fromAseprite(ase *aseprite.Aseprite) (ap *AnimPlayer) {

	if len(ase.Tags) == 0 {
		panic("The Aseprite file does not have a tag.")
	}

	ap = &AnimPlayer{
		Animations: make(map[string]*Animation),
		Atlas:      ebiten.NewImageFromImage(ase.Image),
	}
	for _, tag := range ase.Tags {
		frameCount := tag.Hi - tag.Lo + 1
		frames := make([]*ebiten.Image, 0, frameCount)
		durations := make([]time.Duration, 0, frameCount)

		// kare ve süreleri çek
		for i := tag.Lo; i <= tag.Hi; i++ {
			durations = append(durations, ase.Frames[i].Duration)
			frames = append(frames, ap.Atlas.SubImage(ase.Frames[i].Bounds).(*ebiten.Image))
		}

		switch tag.LoopDirection {
		case aseprite.PingPong:
			for i := len(frames) - 2; i > 0; i-- {
				frames = append(frames, frames[i])
				durations = append(durations, durations[i])
			}
		case aseprite.Reverse:
			slices.Reverse(frames)
			slices.Reverse(durations)
		}

		ap.Animations[tag.Name] = &Animation{
			Tag:       tag.Name,
			Frames:    frames,
			Durations: durations,
		}
	}
	ap.CurrentAnimation = ap.Animations[ase.Tags[0].Name]
	ap.CurrentFrame = ap.CurrentAnimation.Frames[0]
	return
}

func newAseFromFile(path string) (ase *aseprite.Aseprite) {
	f, err := os.Open(path)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	ase, err = aseprite.Read(f)
	if err != nil {
		panic(err)
	}
	return
}

func newAseFromFileSystem(fs fs.FS, path string) (ase *aseprite.Aseprite) {
	file, err := fs.Open(path)
	if err != nil {
		panic(err)
	}
	defer file.Close()
	ase, err = aseprite.Read(file)
	if err != nil {
		panic(err)
	}
	return
}
