[![GoDoc](https://godoc.org/github.com/setanarut/aseplayer?status.svg)](https://pkg.go.dev/github.com/setanarut/aseplayer)

# aseplayer

Aseprite animation player for Ebitengine. 

> [!NOTE]  
> Only the topmost visible layer is considered. Others are ignored..

## Parsing Aseprite file

```Go
fly = aseplayer.NewAnimPlayerFromAsepriteFile("bird.ase")
```


## Tags

Each Aseprite [Tag](https://www.aseprite.org/docs/tags) is imported as an `Animation{}` struct and is ready to play.

### Playing tags

To play multiple animation tags simultaneously, use a shallow copy of `AnimPlayer`. It will share the same animations. Update each `AnimPlayer` with `Update()` and draw it with `Draw()`.

```Go
bird1 = aseplayer.NewAnimPlayerFromAsepriteFile("bird.ase", aseplayer.Default)
bird2 = *bird1
bird3 = *bird1

bird1.Play("fly")
bird2.Play("fly")
bird3.Play("walk")
```

### Tag properties

#### Animation Directions

AsePlayer supports three Animation Directions: `Forward`, `Reverse`, and `Ping-pong`.


> [!NOTE]  
> For **Ping-Pong** and **Reverse** playback, the `[]Frame` is specifically manipulated. For **Ping-Pong**, the number of frames will be greater than the Aseprite range. `[0 1 2 3] -> [0 1 2 3 2 1]`. **Reverse** is an reversed `[]Frame`.

#### Repeat

**AsePlayer** supports the **Repeat** property; `Animation.Repeat = 0` means infinite loop.

```Go
// Override.
g.animPlayer.Animations["turn"].Repeat = 1
```

#### UserData

Text field of Aseprite Tag's User Data. It is useful for data transfer. It can be automated with Aseprite Lua scripting. https://www.aseprite.org/api/tag#tagdata

## Frame Durations

[Frame durations](https://www.aseprite.org/docs/frame-duration) are supported. The animation plays according to these durations. 

```Go
// Override frame's duration.
animPlayer.Animations["walk"].Frames[2].Duration = time.Millisecond * 100
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
	s.DrawImage(g.myAnimPlayer.CurrentFrame.Image, nil)
}
```
