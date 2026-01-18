package aseplayer

import (
	"fmt"
	"image"
	"io/fs"
	"os"
	"slices"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/setanarut/aseplayer/parser"
	"github.com/setanarut/v"
)

const Delta = time.Second / 60

type subImager interface {
	SubImage(image.Rectangle) image.Image
}

// AnimPlayer plays and manages Aseprite tag animations.
type AnimPlayer struct {

	// The frame of the animation currently being played
	CurrentFrame *Frame

	// The animation currently being played
	CurrentAnimation *Animation

	// Animations accessible by their Aseprite tag names
	Animations map[string]*Animation

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
	if a.frameElapsedTime >= a.CurrentFrame.Duration {
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

const formatString string = `Tag: %v
Repeat count: %v
Ended: %v
Index: %v
Frame elapsed: %v
Paused: %v`

func (a *AnimPlayer) String() string {
	return fmt.Sprintf(formatString, a.CurrentAnimation.Tag,
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
	Frames []*Frame

	// Frame durations retrieved from the Aseprite file
	// Durations []time.Duration

	// Repeat specifies how many times the animation should loop.
	// A value of 0 means infinite looping.
	Repeat uint16
}

type Frame struct {
	Image *ebiten.Image
	// Pivot taken from Aseprite's Slice. A point in Frame.Image.Bounds().
	Pivot v.Vec
	// Frame duration from the Aseprite file
	Duration time.Duration
}

// NewAnimPlayerFromAsepriteFileSystem creates an AnimPlayer from an Aseprite file.
//
// The first Aseprite tag is automatically set as the current animation.
//
// When smartSlice is true, the Smart Slice algorithm performs the following:
//   - Finds a Slice whose name matches the Timeline tag name
//   - Crops the Frame.Image to the Slice's bounds
//   - Extracts the Pivot information from the Slice and sets Frame.Pivot
//
// When smartSlice is false, the Frame.Image size matches the Aseprite canvas size,
// and Frame.Pivot default to the top-left of the image. (zero)
//
// The Aseprite file must contain at least one tag, otherwise an error will occur.
func NewAnimPlayerFromAsepriteFileSystem(fs fs.FS, asePath string, smartSlice bool) *AnimPlayer {
	ase := newAseFromFileSystem(fs, asePath)
	ap := fromAseprite(ase, smartSlice)
	ase = nil
	return ap
}

// NewAnimPlayerFromAsepriteFile creates an AnimPlayer from an Aseprite file.
//
// The first Aseprite tag is automatically set as the current animation.
//
// When smartSlice is true, the Smart Slice algorithm performs the following:
//   - Finds a Slice whose name matches the Timeline tag name
//   - Crops the Frame.Image to the Slice's bounds
//   - Extracts the Pivot information from the Slice and sets Frame.Pivot
//
// When smartSlice is false, the Frame.Image size matches the Aseprite canvas size,
// and Frame.Pivot default to the top-left of the image. (zero)
//
// The Aseprite file must contain at least one tag, otherwise an error will occur.
func NewAnimPlayerFromAsepriteFile(asePath string, smartSlice bool) *AnimPlayer {
	ase := newAseFromFile(asePath)
	ap := fromAseprite(ase, smartSlice)
	ase = nil
	return ap
}

func fromAseprite(ase *parser.Aseprite, smartSliceEnabled bool) (ap *AnimPlayer) {

	if len(ase.Tags) == 0 {
		panic("The Aseprite file does not have a tag.")
	}

	ap = &AnimPlayer{
		Animations: make(map[string]*Animation),
	}

	var sliceIndex int

	for _, tag := range ase.Tags {

		ap.Animations[tag.Name] = &Animation{
			Tag:    tag.Name,
			Repeat: tag.Repeat,
		}

		tagLen := tag.Hi - tag.Lo + 1
		frames := make([]*Frame, 0, tagLen)

		if smartSliceEnabled {
			sliceIndex = slices.IndexFunc(ase.Slices, func(e parser.Slice) bool {
				return e.Name == tag.Name
			})
		}

		frameIdx := 0
		for i := tag.Lo; i <= tag.Hi; i++ {
			frames = append(frames, &Frame{})
			var atlasSubImage image.Image
			frameBounds := ase.Frames[i].Bounds

			if smartSliceEnabled {
				if sliceIndex != -1 {
					frameBounds = ase.Slices[sliceIndex].Frames[i].Bounds.Add(frameBounds.Min)
					frames[frameIdx].Pivot = v.Vec{
						X: float64(ase.Slices[sliceIndex].Frames[i].Pivot.X),
						Y: float64(ase.Slices[sliceIndex].Frames[i].Pivot.Y),
					}

				}
			}

			atlasSubImage = ase.Image.(subImager).SubImage(frameBounds)
			frames[frameIdx].Image = ebiten.NewImageFromImage(atlasSubImage)
			frames[frameIdx].Duration = ase.Frames[i].Duration
			frameIdx++
		}

		ap.Animations[tag.Name].Frames = frames

		switch tag.LoopDirection {
		case parser.PingPong:
			for i := len(frames) - 2; i > 0; i-- {
				frames = append(frames, frames[i])
			}
		case parser.Reverse:
			slices.Reverse(frames)
		}

	}

	ap.CurrentAnimation = ap.Animations[ase.Tags[0].Name]
	ap.CurrentFrame = ap.CurrentAnimation.Frames[0]

	return
}

func newAseFromFile(path string) (ase *parser.Aseprite) {
	f, err := os.Open(path)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	ase, err = parser.Read(f)
	if err != nil {
		panic(err)
	}
	return
}

func newAseFromFileSystem(fs fs.FS, path string) (ase *parser.Aseprite) {
	file, err := fs.Open(path)
	if err != nil {
		panic(err)
	}
	defer file.Close()
	ase, err = parser.Read(file)
	if err != nil {
		panic(err)
	}
	return
}
