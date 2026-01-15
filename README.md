[![GoDoc](https://godoc.org/github.com/setanarut/aseplayer?status.svg)](https://pkg.go.dev/github.com/setanarut/aseplayer)

# aseplayer

Aseprite animation player for Ebitengine. 

There are two methods available to read Aseprite files.

```Go
animPlayer := aseplayer.NewAnimPlayerFromAsepriteFile("player.ase")
animPlayer := aseplayer.NewAnimPlayerFromAsepriteFileSystem(fsys, "player.ase")
```

> [!NOTE]  
> Layers are flattened, blending modes are applied, and frames are arranged on a single texture atlas. Invisible and reference layers are ignored.

## Tags

Each Aseprite [Tag](https://www.aseprite.org/docs/tags) is imported as an `Animation{}` struct and is ready to play. Each Tag's frames are stored as a []*ebiten.Image. 

<img width="736" height="172" alt="tags" src="https://github.com/user-attachments/assets/416fb4dc-133c-4e7c-a62e-35d93cab9c86" />


## Tag properties

### Animation Directions

AsePlayer supports three Animation Directions: `Forward`, `Reverse`, and `Ping-pong`.

<img width="336" height="288" alt="tag-properties" src="https://github.com/user-attachments/assets/1d568d23-a745-4526-b152-0d7ec62f8414" />

> [!NOTE]  
> For **Ping-Pong** and **Reverse** playback, the `Frames []*ebiten.Image` slice is specifically manipulated. For **Ping-Pong**, the number of frames will be greater than the Aseprite range. `[0 1 2 3] -> [0 1 2 3 2 1]`. **Reverse** is an reversed `[]*ebiten.Image`.

### Repeat

**AsePlayer** supports the **Repeat** property; `Animation.Repeat = 0` means infinite loop.

```Go
// Override.
g.animPlayer.Animations["turn"].Repeat = 1
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