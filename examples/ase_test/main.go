package main

import (
	"fmt"
	"log"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/setanarut/aseplayer"
)

func main() {

	g := &Game{}
	g.Init()

	ebiten.SetWindowSize(int(g.w), int(g.h))
	if err := ebiten.RunGame(g); err != nil {
		log.Fatal(err)
	}

}

type Game struct {
	animPlayer *aseplayer.AnimPlayer
	w, h       float64
}

func (g *Game) Init() {
	g.animPlayer = aseplayer.NewAnimPlayerFromAsepriteFile("test.ase")
	fmt.Println(g.animPlayer.CurrentFrame.Bounds())
	g.w, g.h = 200, 200
}

func (g *Game) Update() error {
	if inpututil.IsKeyJustPressed(ebiten.KeyQ) {
		g.animPlayer.SetAnim("pingpong")
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyE) {
		g.animPlayer.SetAnim("reverse")
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyW) {
		g.animPlayer.SetAnim("forward")
	}
	g.animPlayer.Update()
	return nil
}

func (g *Game) Draw(s *ebiten.Image) {
	s.DrawImage(g.animPlayer.CurrentFrame, nil)
}

func (g *Game) Layout(w, h int) (int, int) {
	return 200, 200
}

func (g *Game) LayoutF(w, h float64) (float64, float64) {
	return g.w, g.h
}
