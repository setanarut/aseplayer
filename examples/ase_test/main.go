package main

import (
	"image/color"
	"log"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/setanarut/aseplayer"
)

var (
	ani *aseplayer.AnimPlayer
)

func main() {

	ani = aseplayer.NewAnimPlayerFromAsepriteFile("test.ase")

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
	ebiten.SetScreenClearedEveryFrame(false)

	g.w, g.h = 500, 500
}

func (g *Game) Update() error {
	if inpututil.IsKeyJustPressed(ebiten.KeyQ) {
		ani.SetAnim("pingpong")
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyE) {
		ani.SetAnim("reverse")
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyW) {
		ani.SetAnim("forward")
	}
	ani.Update()
	return nil
}

func (g *Game) Draw(s *ebiten.Image) {
	s.Fill(color.Gray{100})
	s.DrawImage(ani.CurrentFrame, nil)
}

func (g *Game) Layout(w, h int) (int, int) {
	return 500, 500
}

func (g *Game) LayoutF(w, h float64) (float64, float64) {
	return g.w, g.h
}
