package aseplayer

import (
	"fmt"
	"image"
	"io/fs"
	"slices"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/setanarut/aseprite"
	"github.com/setanarut/v"
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
	//	dio.GeoM.Translate(animPlayer.CurrentFrame.Position.X, animPlayer.CurrentFrame.Position.Y)
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
	// Position represents the Cel's top-left coordinates relative to the Aseprite canvas.
	Position v.Vec
	// Duration of the frame as defined in the Aseprite file.
	Duration time.Duration
}

// NewAnimPlayerFromAsepriteFileSystem creates an AnimPlayer from an Aseprite file.
//
// The first Aseprite tag is automatically set as the current animation.
//
// The Aseprite file must contain at least one tag, otherwise an error will occur.
func NewAnimPlayerFromAsepriteFileSystem(fs fs.FS, asePath string) *AnimPlayer {
	ase, _ := aseprite.ReadFs(fs, asePath)
	ap := animPlayerfromAseprite(&ase)
	return ap
}

// NewAnimPlayerFromAsepriteFile creates an AnimPlayer from an Aseprite file.
//
// The first Aseprite tag is automatically set as the current animation.
//
// The Aseprite file must contain at least one tag, otherwise an error will occur.
func NewAnimPlayerFromAsepriteFile(asePath string) *AnimPlayer {
	ase, _ := aseprite.Read(asePath)
	ap := animPlayerfromAseprite(&ase)
	return ap
}

func animPlayerfromAseprite(ase *aseprite.Ase) (ap *AnimPlayer) {

	TopmostVisibleLayerIndex := len(ase.Layers) - 1

	if len(ase.Tags) == 0 {
		panic("The Aseprite file does not have a tag.")
	}

	ap = &AnimPlayer{
		Animations: make(map[string]*Animation),
	}

	imageCache := make(map[uint16]*ebiten.Image)

	for _, tag := range ase.Tags {

		ap.Animations[tag.Name] = &Animation{
			Name:     tag.Name,
			Repeat:   tag.Repeat,
			UserData: tag.UserData.Text,
		}

		tagLen := tag.Hi - tag.Lo + 1
		frames := make([]Frame, 0, tagLen)

		frameIdx := 0
		for i := tag.Lo; i <= tag.Hi; i++ {
			frames = append(frames, Frame{})

			pivot := ase.Frames[i].Cels[TopmostVisibleLayerIndex].Image.Bounds().Min
			frames[frameIdx].Position = v.Vec{
				X: float64(pivot.X),
				Y: float64(pivot.Y),
			}
			if cachedImage, exists := imageCache[i]; exists {
				// shallow copy of sub tag image
				frames[frameIdx].Image = cachedImage
			} else {
				newImage := ebiten.NewImageFromImage(ase.Frames[i].Cels[TopmostVisibleLayerIndex].Image)
				imageCache[i] = newImage
				frames[frameIdx].Image = newImage
			}

			frames[frameIdx].Duration = ase.Frames[i].Dur
			frameIdx++
		}

		switch tag.LoopDirection {
		// pingpong
		case 2:
			for i := len(frames) - 2; i > 0; i-- {
				originalFrame := frames[i]
				frameCopy := Frame{
					Image:    originalFrame.Image,
					Position: originalFrame.Position,
					Duration: originalFrame.Duration,
				}
				frames = append(frames, frameCopy)
			}
		// reverse
		case 1:
			slices.Reverse(frames)
		}

		ap.Animations[tag.Name].Frames = frames

	}

	ap.CurrentAnimation = ap.Animations[ase.Tags[0].Name]
	ap.CurrentFrame = &ap.CurrentAnimation.Frames[0]

	return
}
