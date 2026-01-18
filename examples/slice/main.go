package main

import (
	"image/color"
	"log"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"github.com/setanarut/aseplayer"
	"golang.org/x/image/colornames"
)

func main() {

	g := &Game{}
	g.Init()

	ebiten.SetWindowSize(int(g.w), int(g.h))
	ebiten.SetWindowTitle("Slice bounds test")
	if err := ebiten.RunGame(g); err != nil {
		log.Fatal(err)
	}

}

type Game struct {
	animPlayer *aseplayer.AnimPlayer
	w, h       float64
}

func (g *Game) Init() {
	g.animPlayer = aseplayer.NewAnimPlayerFromAsepriteFile("slice.ase", true)
	g.w, g.h = 512, 512
}

func (g *Game) Update() error {

	if inpututil.IsKeyJustPressed(ebiten.Key1) {
		g.animPlayer.Play("slice_test")
	}
	if inpututil.IsKeyJustPressed(ebiten.Key2) {
		g.animPlayer.Play("no_slice")
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyC) {
		g.animPlayer.Animations["forward"].Repeat = 1
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyP) {
		g.animPlayer.Paused = !g.animPlayer.Paused
	}

	g.animPlayer.Update(aseplayer.Delta)

	return nil
}

func (g *Game) Draw(s *ebiten.Image) {
	s.Fill(color.Gray{50})
	ebitenutil.DebugPrint(s, "Play tags\nKey1, Key2")
	ebitenutil.DebugPrintAt(s, g.animPlayer.String(), 192, 0)

	// draw animPlayer
	d := ebiten.DrawImageOptions{}
	d.GeoM.Translate(-g.animPlayer.CurrentFrame.Pivot.X, -g.animPlayer.CurrentFrame.Pivot.Y)
	d.GeoM.Translate(256, 256)
	s.DrawImage(g.animPlayer.CurrentFrame.Image, &d)

	// draw animation bounds
	r := g.animPlayer.CurrentFrame.Image.Bounds()
	x, y := d.GeoM.Apply(float64(r.Min.X), float64(r.Min.Y))
	vector.StrokeRect(s, float32(x), float32(y), float32(r.Dx()), float32(r.Dy()), 1, colornames.Yellow, false)

	// draw animation pivot
	vector.FillCircle(s,
		float32(256),
		float32(256),
		3, colornames.White, false,
	)
}

func (g *Game) Layout(w, h int) (int, int) {
	return 512, 512
}
