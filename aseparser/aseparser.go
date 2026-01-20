package aseparser

import (
	"bytes"
	"compress/zlib"
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
	UserData

	image image.Image
	mask  image.Uniform
}

func makeCelImage8(f *ase, bounds image.Rectangle, opacity byte, pix []byte) cel {
	img := image.Paletted{
		Pix:     pix,
		Stride:  bounds.Dx(),
		Rect:    bounds,
		Palette: f.palette,
	}

	mask := image.Uniform{color.Alpha{opacity}}

	return cel{image: &img, mask: mask}
}

func makeCelImage16(f *ase, bounds image.Rectangle, opacity byte, pix []byte) cel {
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

func makeCelImage32(f *ase, bounds image.Rectangle, opacity byte, pix []byte) cel {
	img := image.NRGBA{
		Pix:    pix,
		Stride: bounds.Dx() * 4,
		Rect:   bounds,
	}

	mask := image.Uniform{color.Alpha{opacity}}

	return cel{image: &img, mask: mask}
}

type Layer struct {
	UserData
	Name string

	flags     uint16
	blendMode uint16
	opacity   byte
	data      []byte
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

type ase struct {
	framew      int
	frameh      int
	flags       uint16
	bpp         uint16
	transparent uint8
	palette     color.Palette
	frames      []frame
	celBounds   []image.Rectangle
	Layers      []Layer
	makeCel     func(f *ase, bounds image.Rectangle, opacity byte, pix []byte) cel
}

func (ase *ase) ReadFrom(r io.Reader) (int64, error) {
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

	ase.bpp = binary.LittleEndian.Uint16(raw[12:])
	ase.flags = binary.LittleEndian.Uint16(raw[14:])
	ase.frames = make([]frame, 0, binary.LittleEndian.Uint16(raw[6:]))
	ase.framew = int(binary.LittleEndian.Uint16(raw[8:]))
	ase.frameh = int(binary.LittleEndian.Uint16(raw[10:]))
	ase.palette = make(color.Palette, binary.LittleEndian.Uint16(raw[32:]))
	ase.transparent = raw[28]

	switch ase.bpp {
	case 8:
		ase.makeCel = makeCelImage8
	case 16:
		ase.makeCel = makeCelImage16
	case 32:
		ase.makeCel = makeCelImage32
	default:
		return 0, errors.New("invalid color depth")
	}

	for i := range ase.palette {
		ase.palette[i] = color.Black
	}
	ase.palette[ase.transparent] = color.Transparent

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

		ase.frames = append(ase.frames, fr)
	}

	return fileSize, nil
}

// Slice Chunk (0x2022)
func (ase *ase) parseSliceChunk0x2022(s *Slice, flags uint32, raw []byte) []byte {
	var key SliceFrame

	x := int32(binary.LittleEndian.Uint32(raw[4:]))
	y := int32(binary.LittleEndian.Uint32(raw[8:]))
	w := binary.LittleEndian.Uint32(raw[12:])
	h := binary.LittleEndian.Uint32(raw[16:])
	raw = raw[20:]

	key.Bounds = image.Rect(int(x), int(y), int(x)+int(w), int(y)+int(h))

	var cx, cy int32
	var cw, ch uint32

	if flags&1 != 0 {
		cx = int32(binary.LittleEndian.Uint32(raw))
		cy = int32(binary.LittleEndian.Uint32(raw[4:]))
		cw = binary.LittleEndian.Uint32(raw[8:])
		ch = binary.LittleEndian.Uint32(raw[12:])
		raw = raw[16:]

		key.Center = image.Rect(int(cx), int(cy), int(cx)+int(cw), int(cy)+int(ch))
	}

	var px, py int32

	if flags&2 != 0 {
		px = int32(binary.LittleEndian.Uint32(raw))
		py = int32(binary.LittleEndian.Uint32(raw[4:]))
		raw = raw[8:]
		key.Pivot = image.Pt(int(px), int(py))
	}

	s.Frames = append(s.Frames, key)

	return raw
}

// Tags Chunk (0x2018)
func (ase *ase) parseTagsChunk0x2018(t *Tag, raw []byte) []byte {
	t.Lo = binary.LittleEndian.Uint16(raw)
	t.Hi = binary.LittleEndian.Uint16(raw[2:])
	t.LoopDirection = LoopDirection(raw[4])
	t.Repeat = binary.LittleEndian.Uint16(raw[5:])
	t.Name = parseString(raw[17:])
	return raw[19+len(t.Name):]
}

// User Data Chunk (0x2020)
func (ase *ase) parseUserDataChunk0x2020(raw []byte) (data []byte, col color.Color) {
	flags := binary.LittleEndian.Uint32(raw)
	raw = raw[4:]

	if flags&1 != 0 {
		n := binary.LittleEndian.Uint16(raw)
		data, raw = raw[2:2+n], raw[2+n:]
	}

	if flags&2 != 0 {
		col = parseColor(raw)
		raw = raw[4:]
	}
	return data, col
}

// Palette Chunk (0x2019)
func (ase *ase) parsePaletteChunk0x2019(raw []byte) {
	entries := binary.LittleEndian.Uint32(raw[0:])
	lo := binary.LittleEndian.Uint32(raw[4:])

	raw = raw[20:]

	for i := range entries {
		flags := binary.LittleEndian.Uint16(raw)
		ase.palette[lo+i] = parseColor(raw[2:])
		raw = raw[6:]

		if flags&1 != 0 {
			raw = skipString(raw)
		}
	}
}

// Old palette chunk (0x0011)
// https://github.com/aseprite/aseprite/blob/main/docs/ase-file-specs.md#old-palette-chunk-0x0011
func (ase *ase) parseOldPaletteChunk0x0011(raw []byte) {
	packets := binary.LittleEndian.Uint16(raw)
	raw = raw[2:]

	currentIndex := 0

	for i := 0; i < int(packets); i++ {
		skip := int(raw[0])
		currentIndex += skip

		n := int(raw[1])
		if n == 0 {
			n = 256
		}
		raw = raw[2:]

		for j := 0; j < n && currentIndex < len(ase.palette); j++ {
			ase.palette[currentIndex] = color.NRGBA{
				R: raw[0] * 4,
				G: raw[1] * 4,
				B: raw[2] * 4,
				A: 255,
			}
			raw = raw[3:]
			currentIndex++
		}
	}
}

// Cel Chunk (0x2005)
func (ase *ase) parseCelChunk0x2005(frame int, raw []byte) (*cel, error) {
	layer := binary.LittleEndian.Uint16(raw)
	xpos := int(int16(binary.LittleEndian.Uint16(raw[2:])))
	ypos := int(int16(binary.LittleEndian.Uint16(raw[4:])))
	opacity := raw[6]
	celtype := binary.LittleEndian.Uint16(raw[7:])

	if ase.Layers[layer].flags&1 == 0 || ase.Layers[layer].flags&64 != 0 {
		return nil, nil
	}

	raw = raw[16:]
	opacity = byte((int(opacity) * int(ase.Layers[layer].opacity)) / 255)

	var pix []byte

	switch celtype {
	case 0: // uncompressed
		pix = raw[4:]
	case 1: // linked
		srcFrame := int(binary.LittleEndian.Uint16(raw))
		ase.frames[frame].cels[layer] = ase.frames[srcFrame].cels[layer]
		return &ase.frames[frame].cels[layer], nil
	case 2: // compressed
		zr, err := zlib.NewReader(bytes.NewReader(raw[4:]))
		if err != nil {
			return nil, err
		}
		defer zr.Close()

		data, err := io.ReadAll(zr)
		if err != nil {
			return nil, err
		}
		pix = data
	default:
		return nil, errors.New("unsupported cel type")
	}

	width := int(binary.LittleEndian.Uint16(raw))
	height := int(binary.LittleEndian.Uint16(raw[2:]))
	bounds := image.Rect(xpos, ypos, xpos+width, ypos+height)

	ase.frames[frame].cels[layer] = ase.makeCel(ase, bounds, opacity, pix)
	return &ase.frames[frame].cels[layer], nil
}

// Old palette chunk (0x0004)
// https://github.com/aseprite/aseprite/blob/main/docs/ase-file-specs.md#old-palette-chunk-0x0004
func (ase *ase) parseOldPaletteChunk0x0004(raw []byte) {
	packets := binary.LittleEndian.Uint16(raw)
	raw = raw[2:]

	currentIndex := 0

	for i := 0; i < int(packets); i++ {
		skip := int(raw[0])
		currentIndex += skip

		n := int(raw[1])
		if n == 0 {
			n = 256
		}
		raw = raw[2:]

		for j := 0; j < n && currentIndex < len(ase.palette); j++ {
			ase.palette[currentIndex] = color.NRGBA{
				R: raw[0],
				G: raw[1],
				B: raw[2],
				A: 255,
			}
			raw = raw[3:]
			currentIndex++
		}
	}
}

func (ase *ase) initPalette() {
	var chunk0004 []byte
	var chunk0011 []byte
	found2019 := false

	for _, ch := range ase.frames[0].chunks {
		if ch.typ == 0x2019 {
			ase.parsePaletteChunk0x2019(ch.raw)
			found2019 = true
			break
		}
		if ch.typ == 0x0004 {
			chunk0004 = ch.raw
		}
		if ch.typ == 0x0011 {
			chunk0011 = ch.raw
		}
	}

	if !found2019 {
		if chunk0004 != nil {
			ase.parseOldPaletteChunk0x0004(chunk0004)
		} else if chunk0011 != nil {
			ase.parseOldPaletteChunk0x0011(chunk0011)
		}
	}

	if ase.flags&1 != 0 {
		ase.palette[ase.transparent] = color.Transparent
	}
}

func (ase *ase) initLayers() error {
	chunks := ase.frames[0].chunks
	for i, ch := range chunks {
		if ch.typ == 0x2004 {
			var l Layer
			if err := l.Parse(ch.raw); err != nil {
				return err
			}

			if i < len(chunks)-1 {
				if ch2 := chunks[i+1]; ch2.typ == 0x2020 {
					data, col := ase.parseUserDataChunk0x2020(ch2.raw)
					l.Text = string(data)
					l.Color = col
				}
			}

			ase.Layers = append(ase.Layers, l)
		}
	}

	nlayers := len(ase.Layers)
	for i := range ase.frames {
		ase.frames[i].cels = make([]cel, nlayers)
	}

	return nil
}

func (ase *ase) initCels() error {
	for i := range ase.frames {
		chunks := ase.frames[i].chunks
		for j, ch := range chunks {
			if ch.typ == 0x2005 {
				cel, err := ase.parseCelChunk0x2005(i, ch.raw)
				if err != nil {
					return err
				} else if cel != nil && j < (len(chunks)-1) {
					// user data chunk
					if ch2 := chunks[j+1]; ch2.typ == 0x2020 {
						data, col := ase.parseUserDataChunk0x2020(ch2.raw)
						cel.Text = string(data)
						cel.Color = col
					}
				}
			}
		}
	}

	return nil
}

func (ase *ase) buildAtlas() (atlas draw.Image, framesr []image.Rectangle) {
	var atlasr image.Rectangle
	atlasr, framesr = makeAtlasFrames(len(ase.frames), ase.framew, ase.frameh)

	switch ase.bpp {
	case 8:
		atlas = image.NewPaletted(atlasr, ase.palette)
	case 16:
		atlas = image.NewNRGBA(atlasr)
	default:
		atlas = image.NewRGBA(atlasr)
	}

	framebounds := image.Rect(0, 0, ase.framew, ase.frameh)
	ase.celBounds = make([]image.Rectangle, 0)

	dstblend := image.NewRGBA(framebounds)
	dst := image.NewRGBA(framebounds)

	transparent := &image.Uniform{color.Transparent}

	for i, fr := range ase.frames {

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
					if int(imgPaletted.Pix[i]) >= len(ase.palette) {
						// Assign a transparent index if the index is outside the palette range.
						imgPaletted.Pix[i] = ase.transparent
					}
				}
			}

			if mode := ase.Layers[layerIndex].blendMode; mode > 0 && int(mode) < len(blend.Modes) {
				draw.Draw(dstblend, framebounds, transparent, image.Point{}, draw.Src)
				blend.Blend(dstblend, sr.Sub(sp), src, sp, dst, sp, blend.Modes[mode])
				src = dstblend
				sp = image.Point{}
			}
			draw.DrawMask(dst, sr, src, sp, &cel.mask, image.Point{}, draw.Over)
		}

		ase.celBounds = append(ase.celBounds, celRect)

		draw.Draw(atlas, framesr[i], dst, image.Point{}, draw.Src)
	}
	return
}

func (ase *ase) buildUserDataText() []byte {
	n := 0

	for _, l := range ase.Layers {
		if l.flags&1 != 0 {
			n += len(l.Text) // data -> Text
		}
	}

	for _, fr := range ase.frames {
		for _, c := range fr.cels {
			n += len(c.Text) // data -> Text
		}
	}

	return make([]byte, 0, n)
}
func (ase *ase) buildLayerUserData(userdata []byte) [][]byte {
	ld := make([][]byte, 0, len(ase.Layers))
	for _, l := range ase.Layers {
		if l.flags&1 != 0 && len(l.Text) > 0 {
			ofs := len(userdata)
			userdata = append(userdata, l.Text...)
			ld = append(ld, userdata[ofs:])
		}
	}
	return ld
}

func (ase *ase) buildFrames(framesr []image.Rectangle, userdata []byte) ([]Frame, []byte) {
	frames := make([]Frame, len(ase.frames))

	for i, fr := range ase.frames {
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

func (ase *ase) buildTags() []Tag {
	chunks := ase.frames[0].chunks
	for i, chunk := range chunks {
		if chunk.typ == 0x2018 {
			raw := chunk.raw
			ntags := int(binary.LittleEndian.Uint16(raw))
			tags := make([]Tag, ntags)

			ptr := raw[10:]
			for j := range ntags {
				ptr = ase.parseTagsChunk0x2018(&tags[j], ptr)
			}

			tagIdx := 0
			for j := i + 1; j < len(chunks) && tagIdx < ntags; j++ {
				if chunks[j].typ == 0x2020 {
					data, col := ase.parseUserDataChunk0x2020(chunks[j].raw)
					tags[tagIdx].UserData.Text = string(data)
					tags[tagIdx].UserData.Color = col
					tagIdx++
				} else if chunks[j].typ == 0x2018 || chunks[j].typ == 0x2019 {
					break
				}
			}
			return tags
		}
	}
	return nil
}

// Slice Chunk (0x2022)
func (ase *ase) buildSlices() (slices []Slice) {
	chunks := ase.frames[0].chunks
	for i, chunk := range chunks {
		if chunk.typ == 0x2022 {
			ofs := len(slices)
			raw := chunk.raw

			nKeysForSlice := int(binary.LittleEndian.Uint32(raw))
			flags := binary.LittleEndian.Uint32(raw[4:])
			name := parseString(raw[12:])

			raw = raw[14+len(name):]

			var s Slice
			s.Name = name
			frameIndices := make([]int, 0, nKeysForSlice)

			// parse each slice
			for i := 0; len(raw) > 0 && i < nKeysForSlice; i++ {
				frameIdx := int(binary.LittleEndian.Uint32(raw))
				frameIndices = append(frameIndices, frameIdx)
				raw = ase.parseSliceChunk0x2022(&s, flags, raw)
			}

			slices = append(slices, s)

			// check for user data chunk (0x2020)
			if i < len(chunks)-1 {
				if ud := chunks[i+1]; ud.typ == 0x2020 {
					data, col := ase.parseUserDataChunk0x2020(ud.raw)
					data = append([]byte{}, data...) // copy
					for j := ofs; j < len(slices); j++ {
						slices[j].UserData.Text = string(data)
						slices[j].Color = col
					}
				}
			}
			expandSliceKey(&slices[len(slices)-1], len(ase.frames), frameIndices)
		}
	}

	return
}

// Expand sparse keys across all frames (first key fills backward, rest forward).
func expandSliceKey(slice *Slice, lenFrames int, frameIndices []int) {
	if len(slice.Frames) == lenFrames {
		return
	}
	expandedKeys := make([]SliceFrame, lenFrames)
	keyIdx := 0
	current := slice.Frames[0]
	for frameIdx := range expandedKeys {
		if keyIdx < len(slice.Frames) && frameIndices[keyIdx] == frameIdx {
			current = slice.Frames[keyIdx]
			keyIdx++
		}
		expandedKeys[frameIdx] = current
	}
	slice.Frames = expandedKeys
}

func skipString(raw []byte) []byte {
	n := binary.LittleEndian.Uint16(raw)
	return raw[2+n:]
}

func parseString(raw []byte) string {
	n := binary.LittleEndian.Uint16(raw)
	return string(raw[2 : 2+n])
}

func parseColor(raw []byte) color.Color {
	return color.NRGBA{
		R: raw[0],
		G: raw[1],
		B: raw[2],
		A: raw[3],
	}
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
