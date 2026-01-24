package aseplayer

import (
	"fmt"
	"image"
	"io/fs"
	"slices"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/setanarut/aseplayer/aseparser"
	"github.com/setanarut/v"
)

type CropMode int

const (

	// All animation frames will be the same size as the canvas. Pivots are zero.
	Default CropMode = iota

	// If the Timeline tag and Slice names are the same, the animation frames are cropped according to the Slice boundaries.
	// Frame.Pivot is the pivot of the Slice. Do not use with sub-tags. Sub-tags always inherit the parent tag slice size.
	//
	// It is the position relative to the top-left corner of the Slice boundaries.
	//
	// https://www.aseprite.org/docs/slices#slices
	Slices

	// All cel images will be trimmed (removes the transparent edges). Frame.Pivot specifies the position on the Aseprite canvas (cel.position). Slices are ignored.
	//
	// https://www.aseprite.org/api/cel#celposition
	Trim
)

const Delta = time.Second / 60

type subImager interface {
	SubImage(image.Rectangle) image.Image
}

// AnimPlayer plays and manages Aseprite tag animations.
type AnimPlayer struct {
	// The frame of the animation currently being played.
	//
	// Example:
	//	dio.GeoM.Translate(-animPlayer.CurrentFrame.Pivot.X, -animPlayer.CurrentFrame.Pivot.Y)
	//	dio.GeoM.Translate(x, y)
	//	screen.DrawImage(g.animPlayer.CurrentFrame.Image, dio)
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

// Update advances the animation by the given delta time.
//
// It handles frame progression, looping, and repeat count logic.
// Does nothing if the animation is paused or has ended.
//
// Example:
//
//	myAnimPlayer.Update(aseplayer.Delta)
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
					a.CurrentFrame = &activeAnim.Frames[a.frameIndex]
					return
				}
				a.frameIndex = 0
			}
		}
	}
	a.CurrentFrame = &activeAnim.Frames[a.frameIndex]
}

// If Animation.Repeat is not zero, it returns true when the animation ends. If it is zero, it is always false.
func (a *AnimPlayer) IsEnded() bool {
	return a.isEnded
}

// Play rewinds and plays the animation.
func (a *AnimPlayer) Play(tag string) {
	a.CurrentAnimation = a.Animations[tag]
	a.CurrentFrame = &a.CurrentAnimation.Frames[0]
	a.Rewind()
}

// PlayIfNotCurrent rewinds and plays the animation with the given tag if it's not already playing
func (a *AnimPlayer) PlayIfNotCurrent(tag string) {
	if tag != a.CurrentAnimation.Name {
		a.Play(tag)
	}
}

// Rewinds animation
func (a *AnimPlayer) Rewind() {
	a.frameIndex = 0
	a.frameElapsedTime = 0
	a.CurrentFrame = &a.CurrentAnimation.Frames[0]
	a.isEnded = false
	a.repeatCount = 0
}

const animPlayerFormatString string = `Repeat count: %v
Ended: %v
Index: %v
Paused: %v
--- Animation ---
%v
`

func (a *AnimPlayer) String() string {
	return fmt.Sprintf(animPlayerFormatString,
		a.repeatCount,
		a.IsEnded(),
		a.frameIndex,
		a.Paused,
		a.CurrentAnimation,
	)
}

const animationFormatString string = `Tag: %v
Total Frames: %v
Repeat: %v
UserData:
%v
`

func (a *Animation) String() string {
	return fmt.Sprintf(animationFormatString,
		a.Name,
		len(a.Frames),
		a.Repeat,
		a.UserData)
}

// Animation for AnimPlayer
type Animation struct {
	// The animation tag name is identical to the Aseprite tags
	Name string
	// Animation frames
	Frames []Frame
	// Repeat specifies how many times the animation should loop.
	// A value of 0 means infinite looping.
	Repeat uint16
	// Text field of Aseprite Tag's User Data.
	//
	// It is useful for data transfer. It can be automated with Aseprite Lua scripting.
	//
	// https://www.aseprite.org/api/tag#tagdata
	UserData string
}

type Frame struct {
	*ebiten.Image
	// Pivot taken from Aseprite's Slice. A point in Frame.Image.Bounds().
	Pivot v.Vec
	// Frame duration from the Aseprite file
	Duration time.Duration
}

// NewAnimPlayerFromAsepriteFileSystem creates an AnimPlayer from an Aseprite file.
//
// The first Aseprite tag is automatically set as the current animation.
//
// The Aseprite file must contain at least one tag, otherwise an error will occur.
func NewAnimPlayerFromAsepriteFileSystem(fs fs.FS, asePath string, mode CropMode) *AnimPlayer {
	ase := aseparser.NewAsepriteFromFileSystem(fs, asePath)
	ap := animPlayerfromAseprite(ase, mode)
	ase = nil
	return ap
}

// NewAnimPlayerFromAsepriteFile creates an AnimPlayer from an Aseprite file.
//
// The first Aseprite tag is automatically set as the current animation.
//
// The Aseprite file must contain at least one tag, otherwise an error will occur.
func NewAnimPlayerFromAsepriteFile(asePath string, mode CropMode) *AnimPlayer {
	ase := aseparser.NewAsepriteFromFile(asePath)
	ap := animPlayerfromAseprite(ase, mode)
	ase = nil
	return ap
}

func animPlayerfromAseprite(ase *aseparser.Aseprite, mode CropMode) (ap *AnimPlayer) {

	if mode > 2 {
		panic("Unsupported CropMode!")
	}

	if len(ase.Tags) == 0 {
		panic("The Aseprite file does not have a tag.")
	}

	ap = &AnimPlayer{
		Animations: make(map[string]*Animation),
	}

	imageCache := make(map[uint16]*ebiten.Image)

	var sliceIndex int

	for _, tag := range ase.Tags {

		ap.Animations[tag.Name] = &Animation{
			Name:     tag.Name,
			Repeat:   tag.Repeat,
			UserData: tag.UserData.Text,
		}

		tagLen := tag.Hi - tag.Lo + 1
		frames := make([]Frame, 0, tagLen)

		if mode == Slices {
			sliceIndex = slices.IndexFunc(ase.Slices, func(e aseparser.Slice) bool {
				return e.Name == tag.Name
			})
		}

		frameIdx := 0
		for i := tag.Lo; i <= tag.Hi; i++ {
			frames = append(frames, Frame{})

			frameBounds := ase.Frames[i].Bounds

			switch mode {
			case Slices:
				if sliceIndex != -1 {
					frameBounds = ase.Slices[sliceIndex].Frames[i].Bounds.Add(frameBounds.Min)
					frames[frameIdx].Pivot = v.Vec{
						X: float64(ase.Slices[sliceIndex].Frames[i].Pivot.X),
						Y: float64(ase.Slices[sliceIndex].Frames[i].Pivot.Y),
					}
				}
			case Trim:
				frameBounds = ase.Frames[i].CelBounds.Add(ase.Frames[i].Bounds.Min)
				frames[frameIdx].Pivot = v.Vec{
					X: float64(ase.Frames[i].CelBounds.Min.X),
					Y: float64(ase.Frames[i].CelBounds.Min.Y),
				}
			}

			if cachedImage, exists := imageCache[i]; exists {
				// shallow copy of sub tag image
				frames[frameIdx].Image = cachedImage
			} else {
				atlasSubImage := ase.Image.(subImager).SubImage(frameBounds)
				newImage := ebiten.NewImageFromImage(atlasSubImage)
				imageCache[i] = newImage
				frames[frameIdx].Image = newImage
			}

			frames[frameIdx].Duration = ase.Frames[i].Duration
			frameIdx++
		}

		switch tag.LoopDirection {
		case aseparser.PingPong:
			for i := len(frames) - 2; i > 0; i-- {
				originalFrame := frames[i]
				frameCopy := Frame{
					Image:    originalFrame.Image,
					Pivot:    originalFrame.Pivot,
					Duration: originalFrame.Duration,
				}
				frames = append(frames, frameCopy)
			}
		case aseparser.Reverse:
			slices.Reverse(frames)
		}

		ap.Animations[tag.Name].Frames = frames

	}

	ap.CurrentAnimation = ap.Animations[ase.Tags[0].Name]
	ap.CurrentFrame = &ap.CurrentAnimation.Frames[0]

	return
}
