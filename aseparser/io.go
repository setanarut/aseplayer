package aseparser

import (
	"image"
	"image/color"
	"io"
	"io/fs"
	"os"
)

func init() {
	image.RegisterFormat("aseprite", "????\xE0\xA5", Decode, DecodeConfig)
}

// NewAsepriteFromFile loads and parses an Aseprite file from the given path.
// It panics if the file cannot be opened or parsed.
func NewAsepriteFromFile(path string) (ase *Aseprite) {
	f, err := os.Open(path)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	ase, err = Read(f)
	if err != nil {
		panic(err)
	}
	return
}

// NewAsepriteFromFileSystem loads and parses an Aseprite file from the given fs path.
// It panics if the file cannot be opened or parsed.
func NewAsepriteFromFileSystem(fs fs.FS, path string) (ase *Aseprite) {
	file, err := fs.Open(path)
	if err != nil {
		panic(err)
	}
	defer file.Close()
	ase, err = Read(file)
	if err != nil {
		panic(err)
	}
	return
}

// Read decodes an Aseprite image from r.
func Read(r io.Reader) (*Aseprite, error) {
	var spr Aseprite
	if err := spr.readFrom(r); err != nil {
		return nil, err
	}

	return &spr, nil
}

// Decode decodes an Aseprite image from r and returns it as an image.Image.
func Decode(r io.Reader) (image.Image, error) {
	return Read(r)
}

// DecodeConfig returns the color model and dimensions of an Aseprite image
// without decoding the entire image.
func DecodeConfig(r io.Reader) (image.Config, error) {
	var f ase

	if _, err := f.ReadFrom(r); err != nil {
		return image.Config{}, err
	}

	fw, fh := factorPowerOfTwo(len(f.frames))
	if f.framew > f.frameh {
		fw, fh = fh, fw
	}

	var colorModel color.Model

	switch f.bpp {
	case 8:
		f.initPalette()
		colorModel = f.palette
	case 16:
		colorModel = color.Gray16Model
	default:
		colorModel = color.RGBAModel
	}

	return image.Config{
		ColorModel: colorModel,
		Width:      f.framew * fw,
		Height:     f.frameh * fh,
	}, nil
}
