// Package aseparser implements a decoder/parser for Aseprite sprite files.
//
// Aseprite file format spec: https://github.com/aseprite/aseprite/blob/main/docs/ase-file-specs.md
package aseparser

import (
	"image"
	"image/color"
	"io"
	"slices"
	"time"
)

type subImager interface {
	SubImage(image.Rectangle) image.Image
}

// LoopDirection enumerates all loop animation directions.
type LoopDirection uint8

const (
	Forward LoopDirection = iota
	Reverse
	PingPong
	PingPongReverse
)

// Tag is an animation tag.
type Tag struct {
	// Name is the name of the tag. Can be duplicate.
	Name string
	// Lo is the first frame in the animation.
	Lo uint16
	// Hi is the last frame in the animation.
	Hi uint16
	// Repeat specifies how many times to repeat the animation.
	Repeat uint16
	// LoopDirection is the looping direction of the animation.
	LoopDirection LoopDirection
	// UserData is optional user data.
	UserData
}

type UserData struct {
	Color color.Color
	Text  string
}

// Frame represents a single frame in the sprite.
type Frame struct {
	// Bounds is the image bounds of the frame in the sprite's atlas.
	Bounds image.Rectangle

	CelBounds image.Rectangle
	// Duration is the time in seconds that the frame should be displayed for
	// in a tag animation loop.
	Duration time.Duration
	// Layers lists all optional UserData set in the cels that make up the frame.
	// The UserData of invisible and reference layers is not included.
	Layers []UserData
}

// Slice represents Aseprite's slice.
type Slice struct {
	// Name is the name of the slice. Can be duplicate.
	Name string
	// Frames contains the slice geometry for each animation frame.
	// Index corresponds to frame number; expanded from sparse keyframes.
	Frames []SliceFrame
	// UserData is optional user data.
	UserData
}

type SliceFrame struct {

	// Bounds is the bounds of the slice.
	Bounds image.Rectangle

	// Center is the 9-slices center relative to Bounds.
	Center image.Rectangle

	// Pivot is the pivot point relative to Bounds.
	Pivot image.Point
}

// Aseprite holds the results of a parsed Aseprite image file.
type Aseprite struct {

	// Image contains all frame images in a single image.
	// Frame bounds specify where the frame images are located.
	image.Image

	// Frames lists all frames that make up the sprite.
	Frames []Frame

	// Tags lists all animation tags.
	Tags []Tag

	// Slices lists all slices.
	Slices []Slice

	// LayerData lists the user data of all visible layers.
	LayerData [][]byte
}

// GetFrameImage returns the image for the specified frame index.
func (a *Aseprite) GetFrameImage(frameIndex uint16) image.Image {
	return a.Image.(subImager).SubImage(a.Frames[frameIndex].Bounds)
}

// GetSliceImage returns the image for the specified slice name and frame index.
func (a *Aseprite) GetSliceImage(sliceName string, frameIndex uint16) image.Image {
	sliceIndex := slices.IndexFunc(a.Slices, func(e Slice) bool {
		return e.Name == sliceName
	})
	if sliceIndex != -1 {
		rect := a.Slices[sliceIndex].Frames[frameIndex].Bounds.Add(a.Frames[frameIndex].Bounds.Min)
		return a.Image.(subImager).SubImage(rect)
	} else {
		return nil
	}
}

func (a *Aseprite) readFrom(r io.Reader) error {
	var f file

	if _, err := f.ReadFrom(r); err != nil {
		return err
	}

	f.initPalette()

	if err := f.initLayers(); err != nil {
		return err
	}

	if err := f.initCels(); err != nil {
		return err
	}

	userdata := f.buildUserData()
	var framesr []image.Rectangle
	a.Image, framesr = f.buildAtlas()
	a.Frames, userdata = f.buildFrames(framesr, userdata)
	a.LayerData = f.buildLayerData(userdata)
	a.Tags = f.buildTags()
	a.Slices = f.buildSlices()

	for i := range a.Frames {
		a.Frames[i].CelBounds = f.celBounds[i]
	}

	return nil
}
