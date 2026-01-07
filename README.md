# aseplayer

Aseprite animation player for Ebitengine

1. AsePlayer only supports a single layer. Do not use multiple layers. Use a separate aseprite file for each character.
2. Each [Tag](https://www.aseprite.org/docs/tags) is imported as an `Animation{}` struct and is ready to play.
  - <img width="736" height="172" alt="tags" src="https://github.com/user-attachments/assets/416fb4dc-133c-4e7c-a62e-35d93cab9c86" />  
3. AsePlayer supports three Animation Directions: `Forward`, `Ping-pong`, and `Reverse`.
  - <img width="336" height="288" alt="tag-properties" src="https://github.com/user-attachments/assets/1d568d23-a745-4526-b152-0d7ec62f8414" />
4. [Frame durations](https://www.aseprite.org/docs/frame-duration) are supported. The animation plays according to these durations.

---

There are two methods available to read the file.

```Go
func NewAnimPlayerFromAsepriteFileSystem(fs fs.FS, asePath string) *AnimPlayer
func NewAnimPlayerFromAsepriteFile(asePath string) *AnimPlayer
```

## Usage

A pseudo-code

```Go
func (g *Game) Update() error {
	if inpututil.IsKeyJustPressed(ebiten.KeyRight) {
		g.myAnimPlayer.SetAnim("walk")
	}
	if inpututil.IsKeyJustPressed(ebiten.KeySpace) {
		g.myAnimPlayer.SetAnim("jump")
	}
	g.myAnimPlayer.Update()
	return nil
}

func (g *Game) Draw(s *ebiten.Image) {
	s.DrawImage(g.myAnimPlayer.CurrentFrame, nil)
}
```