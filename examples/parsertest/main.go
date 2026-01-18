package main

import (
	"image/color"
	"log"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/setanarut/aseplayer/aseparser"
)

var ase = aseparser.NewAsepriteFromFile("../assets/slice.ase")

var img *ebiten.Image

func main() {
	ebiten.SetScreenClearedEveryFrame(false)
	// img = ebiten.NewImageFromImage(ase.GetFrameImage(4))
	img = ebiten.NewImageFromImage(ase.GetSliceImage("slice_test", 1))

	g := &Game{}
	g.Init()

	ebiten.SetWindowSize(int(g.w), int(g.h))
	if err := ebiten.RunGame(g); err != nil {
		log.Fatal(err)
	}
}

type Game struct {
	w, h float64
}

func (g *Game) Init() {
	g.w, g.h = 500, 500
}

func (g *Game) Update() error {
	return nil
}

func (g *Game) Draw(s *ebiten.Image) {
	s.Fill(color.Gray{128})
	s.DrawImage(img, nil)
}

func (g *Game) Layout(w, h int) (int, int) {
	return 500, 500
}

func (g *Game) LayoutF(w, h float64) (float64, float64) {
	return g.w, g.h
}
