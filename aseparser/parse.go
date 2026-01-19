package aseparser

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"errors"
	"image"
	"image/color"
	"io"
)

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

func parseUserData(raw []byte) (data []byte, col color.Color) {
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

func (f *file) parseChunk2019(raw []byte) {
	entries := binary.LittleEndian.Uint32(raw[0:])
	lo := binary.LittleEndian.Uint32(raw[4:])

	raw = raw[20:]

	for i := range entries {
		flags := binary.LittleEndian.Uint16(raw)
		f.palette[lo+i] = parseColor(raw[2:])
		raw = raw[6:]

		if flags&1 != 0 {
			raw = skipString(raw)
		}
	}
}

// https://github.com/aseprite/aseprite/blob/main/docs/ase-file-specs.md#old-palette-chunk-0x0011
func (f *file) parseChunk0011(raw []byte) {
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

		for j := 0; j < n && currentIndex < len(f.palette); j++ {
			f.palette[currentIndex] = color.NRGBA{
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

// https://github.com/aseprite/aseprite/blob/main/docs/ase-file-specs.md#old-palette-chunk-0x0004
func (f *file) parseChunk0004(raw []byte) {
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

		for j := 0; j < n && currentIndex < len(f.palette); j++ {
			f.palette[currentIndex] = color.NRGBA{
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

func (f *file) initPalette() {
	var chunk0004 []byte
	var chunk0011 []byte
	found2019 := false

	for _, ch := range f.frames[0].chunks {
		if ch.typ == 0x2019 {
			f.parseChunk2019(ch.raw)
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
			f.parseChunk0004(chunk0004)
		} else if chunk0011 != nil {
			f.parseChunk0011(chunk0011)
		}
	}

	if f.flags&1 != 0 {
		f.palette[f.transparent] = color.Transparent
	}
}

func (f *file) initLayers() error {
	chunks := f.frames[0].chunks
	for i, ch := range chunks {
		if ch.typ == 0x2004 {
			var l Layer
			if err := l.Parse(ch.raw); err != nil {
				return err
			}

			if i < len(chunks)-1 {
				if ch2 := chunks[i+1]; ch2.typ == 0x2020 {
					data, col := parseUserData(ch2.raw)
					l.Text = string(data)
					l.Color = col
				}
			}

			f.Layers = append(f.Layers, l)
		}
	}

	nlayers := len(f.Layers)
	for i := range f.frames {
		f.frames[i].cels = make([]cel, nlayers)
	}

	return nil
}

func (f *file) parseChunk2005(frame int, raw []byte) (*cel, error) {
	layer := binary.LittleEndian.Uint16(raw)
	xpos := int(int16(binary.LittleEndian.Uint16(raw[2:])))
	ypos := int(int16(binary.LittleEndian.Uint16(raw[4:])))
	opacity := raw[6]
	celtype := binary.LittleEndian.Uint16(raw[7:])

	if f.Layers[layer].flags&1 == 0 || f.Layers[layer].flags&64 != 0 {
		return nil, nil
	}

	raw = raw[16:]
	opacity = byte((int(opacity) * int(f.Layers[layer].opacity)) / 255)

	var pix []byte

	switch celtype {
	case 0: // uncompressed
		pix = raw[4:]
	case 1: // linked
		srcFrame := int(binary.LittleEndian.Uint16(raw))
		f.frames[frame].cels[layer] = f.frames[srcFrame].cels[layer]
		return &f.frames[frame].cels[layer], nil
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

	f.frames[frame].cels[layer] = f.makeCel(f, bounds, opacity, pix)
	return &f.frames[frame].cels[layer], nil
}

func (f *file) initCels() error {
	for i := range f.frames {
		chunks := f.frames[i].chunks
		for j, ch := range chunks {
			if ch.typ == 0x2005 {
				cel, err := f.parseChunk2005(i, ch.raw)
				if err != nil {
					return err
				} else if cel != nil && j < (len(chunks)-1) {
					// user data chunk
					if ch2 := chunks[j+1]; ch2.typ == 0x2020 {
						data, col := parseUserData(ch2.raw)
						cel.Text = string(data)
						cel.Color = col
					}
				}
			}
		}
	}

	return nil
}

func parseTag(t *Tag, raw []byte) []byte {
	t.Lo = binary.LittleEndian.Uint16(raw)
	t.Hi = binary.LittleEndian.Uint16(raw[2:])
	t.LoopDirection = LoopDirection(raw[4])
	t.Repeat = binary.LittleEndian.Uint16(raw[5:])
	t.Name = parseString(raw[17:])
	return raw[19+len(t.Name):]
}

func (f *file) buildTags() []Tag {
	chunks := f.frames[0].chunks
	for i, chunk := range chunks {
		if chunk.typ == 0x2018 {
			raw := chunk.raw
			ntags := int(binary.LittleEndian.Uint16(raw))
			tags := make([]Tag, ntags)

			ptr := raw[10:]
			for j := 0; j < ntags; j++ {
				ptr = parseTag(&tags[j], ptr)
			}

			tagIdx := 0
			for j := i + 1; j < len(chunks) && tagIdx < ntags; j++ {
				if chunks[j].typ == 0x2020 {
					data, col := parseUserData(chunks[j].raw)
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

func parseSlice(s *Slice, flags uint32, raw []byte) []byte {
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

func (f *file) buildSlices() (slices []Slice) {
	chunks := f.frames[0].chunks
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
				raw = parseSlice(&s, flags, raw)
			}

			slices = append(slices, s)

			// check for user data chunk
			if i < len(chunks)-1 {
				if ud := chunks[i+1]; ud.typ == 0x2020 {
					data, col := parseUserData(ud.raw)
					data = append([]byte{}, data...) // copy
					for j := ofs; j < len(slices); j++ {
						slices[j].UserData.Text = string(data)
						slices[j].Color = col
					}
				}
			}
			expandSliceKey(&slices[len(slices)-1], len(f.frames), frameIndices)
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
