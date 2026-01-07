package aseplayer

import (
	"io/fs"
	"log"
	"os"
	"slices"
	"time"

	"github.com/askeladdk/aseprite"
	"github.com/hajimehoshi/ebiten/v2"
)

// AnimPlayer plays and manages Aseprite tag animations.
type AnimPlayer struct {
	CurrentFrame     *ebiten.Image
	CurrentAnimation *Animation
	Animations       map[string]*Animation
	Atlas            *ebiten.Image
	Paused           bool
	ElapsedTime      time.Duration
	Index            int
}

const fixedDelta = time.Second / 60

func (ap *AnimPlayer) Update() {
	if ap.Paused {
		return
	}
	a := ap.CurrentAnimation
	ap.ElapsedTime += fixedDelta
	if ap.ElapsedTime >= a.Durations[ap.Index] {
		ap.ElapsedTime = 0
		ap.Index++
		if ap.Index >= len(a.Frames) {
			ap.Index = 0
		}
	}
	ap.CurrentFrame = a.Frames[ap.Index]
}

// SetAnim sets the animation state and resets to the first frame.
//
// Do not call this in every Update() frame. Set it only once in the Enter/Exit events,
// otherwise, the animation will always reset to the first index.
func (ap *AnimPlayer) SetAnim(name string) {
	ap.CurrentAnimation = ap.Animations[name]
	ap.Index = 0
	ap.ElapsedTime = 0
}

// CheckAndSetAnim changes the animation and resets to the first frame if the animation state is not the current state.
//
// It can be called on every Update() frame.
//
// For optimization, it is recommended to use SeAnim() only during state transitions.
func (ap *AnimPlayer) CheckAndSetAnim(name string) {
	if name != ap.CurrentAnimation.Name {
		ap.CurrentAnimation = ap.Animations[name]
		ap.Index = 0
		ap.ElapsedTime = 0
	}
}

// Animation for AnimPlayer
type Animation struct {
	Name      string          // Name of the aimation
	Frames    []*ebiten.Image // Animation frames
	Durations []time.Duration // Frame durations (milliseconds)
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
			Name:      tag.Name,
			Frames:    frames,
			Durations: durations,
		}
	}
	ap.CurrentAnimation = ap.Animations[ase.Tags[0].Name]
	return
}

func newAseFromFile(path string) (ase *aseprite.Aseprite) {
	f, err := os.Open(path)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	ase, err = aseprite.Read(f)
	if err != nil {
		log.Fatal(err)
	}
	return
}

func newAseFromFileSystem(fs fs.FS, path string) (ase *aseprite.Aseprite) {
	file, err := fs.Open(path)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()
	ase, err = aseprite.Read(file)
	if err != nil {
		log.Fatal(err)
	}
	return
}
