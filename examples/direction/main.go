package main

import (
	"fmt"
	"log"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
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
	g.animPlayer = aseplayer.NewAnimPlayerFromAsepriteFile("../assets/dir.ase", false)

	fmt.Println(g.animPlayer.CurrentAnimation)
	g.w, g.h = 512, 512
}

func (g *Game) Update() error {

	if inpututil.IsKeyJustPressed(ebiten.Key1) {
		g.animPlayer.Play("forward")
	}
	if inpututil.IsKeyJustPressed(ebiten.Key2) {
		g.animPlayer.Play("reverse")
	}
	if inpututil.IsKeyJustPressed(ebiten.Key3) {
		g.animPlayer.Play("repeat_2")
	}
	if inpututil.IsKeyJustPressed(ebiten.Key4) {
		g.animPlayer.Play("ping_pong")
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyR) {
		g.animPlayer.Rewind()
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
	ebitenutil.DebugPrintAt(s, "Play tags\nKey1 = forward\nkey2 = reverse\nKey3 = repeat_2\nKey4 = ping_pong\n", 10, 10)
	ebitenutil.DebugPrintAt(s, g.animPlayer.String(), 10, 140)

	d := ebiten.DrawImageOptions{}
	d.GeoM.Translate(192, 192)
	s.DrawImage(g.animPlayer.CurrentFrame.Image, &d)
}

func (g *Game) Layout(w, h int) (int, int) {
	return 512, 512
}
