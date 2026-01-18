package main

import (
	"image/color"
	"log"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/setanarut/aseplayer"
)

func main() {

	g := &Game{}
	g.Init()

	ebiten.SetWindowSize(int(g.w), int(g.h))
	ebiten.SetWindowTitle("500 ms test")
	if err := ebiten.RunGame(g); err != nil {
		log.Fatal(err)
	}

}

type Game struct {
	animPlayer *aseplayer.AnimPlayer
	w, h       float64
}

func (g *Game) Init() {
	ebiten.SetScreenClearedEveryFrame(false)
	g.animPlayer = aseplayer.NewAnimPlayerFromAsepriteFile("../assets/blend.ase", false)
	g.w, g.h = 512, 512
}

func (g *Game) Update() error {
	g.animPlayer.Update(aseplayer.Delta)
	return nil
}

func (g *Game) Draw(s *ebiten.Image) {
	s.Fill(color.Gray{128})
	ebitenutil.DebugPrintAt(s, g.animPlayer.String(), 192, 0)
	d := ebiten.DrawImageOptions{}
	d.GeoM.Translate(192, 192)

	s.DrawImage(g.animPlayer.CurrentFrame.Image, &d)
}

func (g *Game) Layout(w, h int) (int, int) {
	return 512, 512
}
