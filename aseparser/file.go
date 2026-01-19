package aseparser

import (
	"bytes"
	"encoding/binary"
	"errors"
	"image"
	"image/color"
	"image/draw"
	"io"
	"math"
	"time"

	"github.com/setanarut/aseplayer/aseparser/blend"
)

var errInvalidMagic = errors.New("invalid magic number")

type cel struct {
	image    image.Image
	mask     image.Uniform
	UserData // Embedding UserData struct
}

func makeCelImage8(f *file, bounds image.Rectangle, opacity byte, pix []byte) cel {
	img := image.Paletted{
		Pix:     pix,
		Stride:  bounds.Dx(),
		Rect:    bounds,
		Palette: f.palette,
	}

	mask := image.Uniform{color.Alpha{opacity}}

	return cel{image: &img, mask: mask}
}

func makeCelImage16(f *file, bounds image.Rectangle, opacity byte, pix []byte) cel {
	img := image.NewNRGBA(bounds)

	// 16 bpp grayscale+alpha -> NRGBA
	stride := bounds.Dx() * 2
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			i := (y-bounds.Min.Y)*stride + (x-bounds.Min.X)*2
			grayValue := pix[i]    // 8-bit grey
			alphaValue := pix[i+1] // 8-bit alpha

			finalAlpha := uint16(alphaValue) * uint16(opacity) / 255

			img.SetNRGBA(x, y, color.NRGBA{
				R: grayValue,
				G: grayValue,
				B: grayValue,
				A: byte(finalAlpha),
			})
		}
	}
	mask := image.Uniform{color.Alpha{opacity}}
	return cel{image: img, mask: mask}
}

func makeCelImage32(f *file, bounds image.Rectangle, opacity byte, pix []byte) cel {
	img := image.NRGBA{
		Pix:    pix,
		Stride: bounds.Dx() * 4,
		Rect:   bounds,
	}

	mask := image.Uniform{color.Alpha{opacity}}

	return cel{image: &img, mask: mask}
}

type Layer struct {
	Name      string
	flags     uint16
	blendMode uint16
	opacity   byte
	data      []byte
	UserData  // Embedding UserData struct
}

func (l *Layer) Parse(raw []byte) error {
	if typ := binary.LittleEndian.Uint16(raw[2:]); typ == 2 {
		return errors.New("tilemap layers not supported")
	}
	l.flags = binary.LittleEndian.Uint16(raw)
	l.blendMode = binary.LittleEndian.Uint16(raw[10:])
	l.opacity = raw[12]
	// Skip three zero bytes which are reserved for future by specification
	l.Name = string(raw[16:]) // 12+3=15
	return nil
}

type chunk struct {
	typ int
	raw []byte
}

func (c chunk) Reader() io.Reader {
	return bytes.NewReader(c.raw)
}

func (c *chunk) Read(raw []byte) ([]byte, error) {
	chunkLen := binary.LittleEndian.Uint32(raw)
	c.typ = int(binary.LittleEndian.Uint16(raw[4:]))
	c.raw = raw[6:chunkLen]
	return raw[chunkLen:], nil
}

type frame struct {
	dur    time.Duration
	chunks []chunk
	cels   []cel
}

func (f *frame) Read(raw []byte) ([]byte, error) {
	if magic := binary.LittleEndian.Uint16(raw[4:]); magic != 0xF1FA {
		return nil, errInvalidMagic
	}

	// frameLen := binary.LittleEndian.Uint32(raw[0:])
	oldChunks := binary.LittleEndian.Uint16(raw[6:])
	durationMS := binary.LittleEndian.Uint16(raw[8:])
	newChunks := binary.LittleEndian.Uint32(raw[12:])

	f.dur = time.Millisecond * time.Duration(durationMS)

	nchunks := int(newChunks)
	if nchunks == 0 {
		nchunks = int(oldChunks)
	}

	f.chunks = make([]chunk, nchunks)

	raw = raw[16:]

	for i := 0; i < nchunks; i++ {
		var c chunk
		raw, _ = c.Read(raw)
		f.chunks[i] = c
	}

	return raw, nil
}

type file struct {
	framew      int
	frameh      int
	flags       uint16
	bpp         uint16
	transparent uint8
	palette     color.Palette
	frames      []frame
	celBounds   []image.Rectangle
	Layers      []Layer
	makeCel     func(f *file, bounds image.Rectangle, opacity byte, pix []byte) cel
}

func (f *file) ReadFrom(r io.Reader) (int64, error) {
	var hdr [128]byte

	raw := hdr[:]

	if n, err := io.ReadFull(r, raw); err != nil {
		return int64(n), err
	}

	if magic := binary.LittleEndian.Uint16(raw[4:]); magic != 0xA5E0 {
		return 128, errInvalidMagic
	}

	if pixw, pixh := raw[34], raw[35]; pixw != pixh {
		return 128, errors.New("unsupported pixel ratio")
	}

	f.bpp = binary.LittleEndian.Uint16(raw[12:])
	f.flags = binary.LittleEndian.Uint16(raw[14:])
	f.frames = make([]frame, 0, binary.LittleEndian.Uint16(raw[6:]))
	f.framew = int(binary.LittleEndian.Uint16(raw[8:]))
	f.frameh = int(binary.LittleEndian.Uint16(raw[10:]))
	f.palette = make(color.Palette, binary.LittleEndian.Uint16(raw[32:]))
	f.transparent = raw[28]

	switch f.bpp {
	case 8:
		f.makeCel = makeCelImage8
	case 16:
		f.makeCel = makeCelImage16
	case 32:
		f.makeCel = makeCelImage32
	default:
		return 0, errors.New("invalid color depth")
	}

	for i := range f.palette {
		f.palette[i] = color.Black
	}
	f.palette[f.transparent] = color.Transparent

	fileSize := int64(binary.LittleEndian.Uint32(raw))
	raw = make([]byte, fileSize-128)

	if n, err := io.ReadFull(r, raw); err != nil {
		return int64(128 + n), err
	}

	for len(raw) > 0 {
		var fr frame
		var err error
		if raw, err = fr.Read(raw); err != nil {
			return fileSize, err
		}

		f.frames = append(f.frames, fr)
	}

	return fileSize, nil
}

func (f *file) buildAtlas() (atlas draw.Image, framesr []image.Rectangle) {
	var atlasr image.Rectangle
	atlasr, framesr = makeAtlasFrames(len(f.frames), f.framew, f.frameh)

	switch f.bpp {
	case 8:
		atlas = image.NewPaletted(atlasr, f.palette)
	case 16:
		atlas = image.NewNRGBA(atlasr)
	default:
		atlas = image.NewRGBA(atlasr)
	}

	framebounds := image.Rect(0, 0, f.framew, f.frameh)
	f.celBounds = make([]image.Rectangle, 0)

	dstblend := image.NewRGBA(framebounds)
	dst := image.NewRGBA(framebounds)

	transparent := &image.Uniform{color.Transparent}

	for i, fr := range f.frames {

		var celRect image.Rectangle

		draw.Draw(dst, framebounds, transparent, image.Point{}, draw.Src)

		for layerIndex, cel := range fr.cels {

			// hücre boşsa atla
			if cel.image == nil {
				continue
			}

			celRect = cel.image.Bounds()

			src := cel.image
			sr := src.Bounds()
			sp := sr.Min

			// Correction to avoid palette index errors if a color has been deleted from the Aseprite palette.
			if imgPaletted, ok := src.(*image.Paletted); ok {
				for i := range imgPaletted.Pix {
					if int(imgPaletted.Pix[i]) >= len(f.palette) {
						// Assign a transparent index if the index is outside the palette range.
						imgPaletted.Pix[i] = f.transparent
					}
				}
			}

			if mode := f.Layers[layerIndex].blendMode; mode > 0 && int(mode) < len(blend.Modes) {
				draw.Draw(dstblend, framebounds, transparent, image.Point{}, draw.Src)
				blend.Blend(dstblend, sr.Sub(sp), src, sp, dst, sp, blend.Modes[mode])
				src = dstblend
				sp = image.Point{}
			}
			draw.DrawMask(dst, sr, src, sp, &cel.mask, image.Point{}, draw.Over)
		}

		f.celBounds = append(f.celBounds, celRect)

		draw.Draw(atlas, framesr[i], dst, image.Point{}, draw.Src)
	}
	return
}

func (f *file) buildUserData() []byte {
	n := 0

	for _, l := range f.Layers {
		if l.flags&1 != 0 {
			n += len(l.Text) // data -> Text
		}
	}

	for _, fr := range f.frames {
		for _, c := range fr.cels {
			n += len(c.Text) // data -> Text
		}
	}

	return make([]byte, 0, n)
}
func (f *file) buildLayerData(userdata []byte) [][]byte {
	ld := make([][]byte, 0, len(f.Layers))
	for _, l := range f.Layers {
		if l.flags&1 != 0 && len(l.Text) > 0 {
			ofs := len(userdata)
			userdata = append(userdata, l.Text...)
			ld = append(ld, userdata[ofs:])
		}
	}
	return ld
}

func (f *file) buildFrames(framesr []image.Rectangle, userdata []byte) ([]Frame, []byte) {
	frames := make([]Frame, len(f.frames))

	for i, fr := range f.frames {
		frames[i].Duration = fr.dur
		frames[i].Bounds = framesr[i]
		frameUserDatas := make([]UserData, 0, len(fr.cels))
		for _, c := range fr.cels {
			if c.Text != "" || c.Color != nil {
				frameUserDatas = append(frameUserDatas, UserData{Text: c.Text, Color: c.Color})
			}
		}
		frames[i].Layers = frameUserDatas
	}
	

	return frames, userdata
}

func makeAtlasFrames(nframes, framew, frameh int) (atlasr image.Rectangle, framesr []image.Rectangle) {

	fw, fh := factorPowerOfTwo(nframes)
	if framew > frameh {
		fw, fh = fh, fw
	}

	atlasr = image.Rect(0, 0, fw*framew, fh*frameh)

	for i := range nframes {
		x, y := i%fw, i/fw
		framesr = append(framesr, image.Rectangle{
			Min: image.Pt(x*framew, y*frameh),
			Max: image.Pt((x+1)*framew, (y+1)*frameh),
		})
	}

	return
}

// factorPowerOfTwo computes n=a*b, where a, b are powers of two and a >= b.
func factorPowerOfTwo(n int) (a, b int) {
	x := int(math.Ceil(math.Log2(float64(n))))
	a = 1 << (x - x/2)
	b = 1 << (x / 2)
	return
}
