package aseplayer

import (
	"fmt"
	"io/fs"
	"os"
	"slices"
	"time"

	"github.com/askeladdk/aseprite"
	"github.com/hajimehoshi/ebiten/v2"
)

const Delta = time.Second / 60

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

	frameElapsedTime time.Duration
	frameIndex       int
	isEnded          bool
	repeatCount      uint16
}

func (a *AnimPlayer) Update(dt time.Duration) {
	if a.Paused || a.isEnded {
		return
	}
	activeAnim := a.CurrentAnimation
	a.frameElapsedTime += dt
	if a.frameElapsedTime >= activeAnim.Durations[a.frameIndex] {
		a.frameElapsedTime = 0
		a.frameIndex++
		if a.frameIndex >= len(activeAnim.Frames) {
			if activeAnim.Repeat == 0 {
				a.frameIndex = 0
			} else {
				a.repeatCount++
				if a.repeatCount >= activeAnim.Repeat {
					a.isEnded = true
					a.frameIndex = len(activeAnim.Frames) - 1
					a.CurrentFrame = activeAnim.Frames[a.frameIndex]
					return
				}
				a.frameIndex = 0
			}
		}
	}
	a.CurrentFrame = activeAnim.Frames[a.frameIndex]
}

// If Animation.Repeat is not zero, it returns true when the animation ends. If it is zero, it is always false.
func (a *AnimPlayer) IsEnded() bool {
	return a.isEnded
}

// Play rewinds and plays the animation.
func (a *AnimPlayer) Play(tag string) {
	a.CurrentAnimation = a.Animations[tag]
	a.CurrentFrame = a.CurrentAnimation.Frames[0]
	a.Rewind()
}

// PlayIfNotCurrent rewinds and plays the animation with the given tag if it's not already playing
func (a *AnimPlayer) PlayIfNotCurrent(tag string) {
	if tag != a.CurrentAnimation.Tag {
		a.Play(tag)
	}
}

// Rewinds animation
func (a *AnimPlayer) Rewind() {
	a.frameIndex = 0
	a.frameElapsedTime = 0
	a.CurrentFrame = a.CurrentAnimation.Frames[0]
	a.isEnded = false
	a.repeatCount = 0
}

func (a *AnimPlayer) String() string {
	return fmt.Sprintf(debugFormat, a.CurrentAnimation.Tag,
		a.repeatCount,
		a.IsEnded(),
		a.frameIndex,
		a.frameElapsedTime,
		a.Paused)
}

// Animation for AnimPlayer
type Animation struct {

	// The animation tag name is identical to the Aseprite file
	Tag string

	// Animation frames
	Frames []*ebiten.Image

	// Frame durations retrieved from the Aseprite file
	Durations []time.Duration

	// Repeat specifies how many times the animation should loop.
	// A value of 0 means infinite looping.
	Repeat uint16
}

// The first Aseprite tag will be assigned as CurrentAnimation.
//
// Do not read .ase/.aseprite files that do not have a tag.
func NewAnimPlayerFromAsepriteFileSystem(fs fs.FS, asePath string) *AnimPlayer {
	ase := newAseFromFileSystem(fs, asePath)
	ap := fromAseprite(ase)
	ase = nil
	return ap
}

// The first Aseprite tag will be assigned as CurrentAnimation.
//
// Do not read .ase/.aseprite files that do not have a tag.
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
			Repeat:    tag.Repeat,
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
