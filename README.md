[![GoDoc](https://godoc.org/github.com/setanarut/aseplayer?status.svg)](https://pkg.go.dev/github.com/setanarut/aseplayer)

# aseplayer

Aseprite animation player for Ebitengine. 

There are two methods available to read Aseprite files.

```Go
func NewAnimPlayerFromAsepriteFile(asePath string, smartSlice bool) *AnimPlayer
func NewAnimPlayerFromAsepriteFileSystem(fs fs.FS, asePath string, smartSlice bool) *AnimPlayer
```

> [!NOTE]  
> Layers are flattened, blending modes are applied, and frames are arranged on a single texture atlas. Invisible and reference layers are ignored.

## Tags

Each Aseprite [Tag](https://www.aseprite.org/docs/tags) is imported as an `Animation{}` struct and is ready to play. Each Tag's frames are stored as a []*ebiten.Image.

<img width="655" height="155" alt="Tags" src="https://github.com/user-attachments/assets/be21a4af-451f-4e02-b457-88d1d29123ab" />

## Tag properties

### Animation Directions

AsePlayer supports three Animation Directions: `Forward`, `Reverse`, and `Ping-pong`.

<img width="503" height="318" alt="dir" src="https://github.com/user-attachments/assets/167e82ec-e9e5-454c-989b-15a2712c9de9" />

> [!NOTE]  
> For **Ping-Pong** and **Reverse** playback, the `Frames []*ebiten.Image` slice is specifically manipulated. For **Ping-Pong**, the number of frames will be greater than the Aseprite range. `[0 1 2 3] -> [0 1 2 3 2 1]`. **Reverse** is an reversed `[]*ebiten.Image`.

### Repeat

**AsePlayer** supports the **Repeat** property; `Animation.Repeat = 0` means infinite loop.

<img width="434" height="107" alt="repeat" src="https://github.com/user-attachments/assets/f275762c-20db-426e-a840-85949f23eb3f" />

```Go
// Override.
g.animPlayer.Animations["turn"].Repeat = 1
```

## Slices

With the `smartSlice` argument, if there is a *Slice* with the same name as the *Tag*, the animation frames is trimmed accordingly. The pivot point of the slice is also taken as `Animation.PivotX` and `Animation.PivotY` See [./examples/slice](./examples/slice/)

```Go
func NewAnimPlayerFromAsepriteFile(asePath string, smartSlice bool) *AnimPlayer
```

## Frame Durations

[Frame durations](https://www.aseprite.org/docs/frame-duration) are supported. The animation plays according to these durations. 

```Go
// Override the third "walk" frame's duration.
animPlayer.Animations["walk"].Durations[2] = time.Millisecond * 100
```

## Usage

A pseudo-code for basic usage with `Play()`

```Go
func (g *Game) Update() error {
	if inpututil.IsKeyJustPressed(ebiten.KeyRight) {
		g.AnimPlayer.Play("walk")
	}
	if inpututil.IsKeyJustPressed(ebiten.KeySpace) {
		g.AnimPlayer.Play("jump")
	}
	// Update AnimPlayer
	g.myAnimPlayer.Update(aseplayer.Delta)
	return nil
}

func (g *Game) Draw(s *ebiten.Image) {
	// Draw AnimPlayer
	s.DrawImage(g.myAnimPlayer.CurrentFrame, nil)
}
```
