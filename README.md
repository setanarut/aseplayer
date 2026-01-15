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


```Go
type Animation struct {
	// The animation tag name is identical to the Aseprite file
	Tag string
	// Animation frames
	Frames []*ebiten.Image
	// Frame durations retrieved from the Aseprite file
	Durations []time.Duration
}
```

## Animation Direction

AsePlayer supports three Animation Directions: `Forward`, `Ping-pong`, and `Reverse`.

<img width="336" height="288" alt="tag-properties" src="https://github.com/user-attachments/assets/1d568d23-a745-4526-b152-0d7ec62f8414" />

> [!NOTE]  
> For **Ping-Pong** and **Reverse** playback, the `Frames []*ebiten.Image` slice is specifically manipulated. For **Ping-Pong**, the number of frames will be greater than the Aseprite range. `[0 1 2] -> [0 1 2 3 2 1]`. **Reverse** is an reversed `[]*ebiten.Image`.

```go
case aseprite.PingPong:
	for i := len(frames) - 2; i > 0; i-- {
		frames = append(frames, frames[i])
		durations = append(durations, durations[i])
	}
```

## Frame Durations

[Frame durations](https://www.aseprite.org/docs/frame-duration) are supported. The animation plays according to these durations.

```Go
// Example: Override the third "walk" frame's duration.
g.animPlayer.Animations["walk"].Durations[2] = time.Millisecond * 100
```

## Usage and Tips

### SetAnim()

A pseudo-code for basic usage with `SetAnim()`

```Go
func (g *Game) Update() error {
	if inpututil.IsKeyJustPressed(ebiten.KeyRight) {
		g.AnimPlayer.SetAnim("walk")
	}
	if inpututil.IsKeyJustPressed(ebiten.KeySpace) {
		g.AnimPlayer.SetAnim("jump")
	}
	// Update AnimPlayer
	g.myAnimPlayer.Update()
	return nil
}

func (g *Game) Draw(s *ebiten.Image) {
	// Draw AnimPlayer
	s.DrawImage(g.myAnimPlayer.CurrentFrame, nil)
}
```

### IsJustEnded()

The currently playing animation can trigger another state once it finishes playing a single time.

```Go
func (s *walkTurnState) Enter(p *Player) {
	p.animPlayer.SetAnim("walk_turn")
}

func (s *walkTurnState) Update(p *Player) {
	// "walk_turn" plays only once
	if p.animPlayer.IsJustEnded() {
		fsm.ChangeState(p.walkState)
	}
}
```

### CheckAndSetAnim()

Animation states can be handled inside each Update() call.

```Go
func (s *run) Update(p *Player) {
	if p.firePressed {
		switch p.Direction8 {
		case UpRight, UpLeft:
			p.animPlayer.CheckAndSetAnim("run_shoot_diag_up")
		default:
			p.animPlayer.CheckAndSetAnim("run_shoot")
		}
	} else {
		p.animPlayer.CheckAndSetAnim("run")
	}
```