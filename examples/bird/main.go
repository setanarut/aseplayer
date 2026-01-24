package main

import (
	"image"
	"image/color"
	"log"
	"strconv"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"github.com/setanarut/aseplayer"
	"github.com/setanarut/v"
	"golang.org/x/image/colornames"
)

func main() {

	g := &Game{}
	g.Init()
	ebiten.SetScreenClearedEveryFrame(false)
	ebiten.SetWindowSize(int(g.w), int(g.h))
	if err := ebiten.RunGame(g); err != nil {
		log.Fatal(err)
	}

}

type Game struct {
	fly    aseplayer.AnimPlayer
	subfly aseplayer.AnimPlayer
	w, h   float64

	flyPos    v.Vec
	subflyPos v.Vec
	dio       ebiten.DrawImageOptions
}

func (g *Game) Init() {

	g.fly = *aseplayer.NewAnimPlayerFromAsepriteFile("../../testfiles/bird.ase", aseplayer.Trim)

	// See the testfiles/userdata_pivots.lua and testfiles/bird.ase files.
	ParseUserDataPivots(&g.fly)

	g.flyPos = v.Vec{300, 120}
	g.subflyPos = v.Vec{300, 400}

	// shallow copy (shared frames)
	g.subfly = g.fly

	// initalize tags
	g.fly.Play("fly")
	g.subfly.Play("sub_fly")

	g.w, g.h = 512, 512
	g.dio = ebiten.DrawImageOptions{}
}

func (g *Game) Update() error {
	g.fly.Update(aseplayer.Delta)
	g.subfly.Update(aseplayer.Delta)
	return nil
}

func (g *Game) Draw(s *ebiten.Image) {

	s.Fill(color.RGBA{36, 56, 116, 255})

	g.dio.GeoM.Reset()
	g.dio.GeoM.Translate(g.fly.CurrentFrame.Pivot.X, g.fly.CurrentFrame.Pivot.Y)
	g.dio.GeoM.Translate(g.flyPos.X, g.flyPos.Y)
	s.DrawImage(g.fly.CurrentFrame.Image, &g.dio)

	// draw pivot point
	vector.FillCircle(s, float32(g.flyPos.X), float32(g.flyPos.Y), 3, colornames.Red, true)
	// draw frame image bounds
	StrokeImageRectangle(g.dio.GeoM, g.fly.CurrentFrame.Bounds(), s)

	g.dio.GeoM.Reset()
	g.dio.GeoM.Translate(g.subfly.CurrentFrame.Pivot.X, g.subfly.CurrentFrame.Pivot.Y)
	g.dio.GeoM.Translate(g.subflyPos.X, g.subflyPos.Y)
	s.DrawImage(g.subfly.CurrentFrame.Image, &g.dio)

	// draw pivot point
	vector.FillCircle(s, float32(g.subflyPos.X), float32(g.subflyPos.Y), 3, colornames.Red, true)
	// draw frame image bounds
	StrokeImageRectangle(g.dio.GeoM, g.subfly.CurrentFrame.Bounds(), s)

	ebitenutil.DebugPrintAt(s, g.fly.String(), 10, 10)
	ebitenutil.DebugPrintAt(s, g.subfly.String(), 10, 300)
}

func (g *Game) Layout(w, h int) (int, int) {
	return 512, 512
}

func ParseUserDataPivots(ap *aseplayer.AnimPlayer) {
	for _, v := range ap.Animations {
		for i := range v.Frames {
			v.Frames[i].Pivot = v.Frames[i].Pivot.Sub(parseOffset(v.UserData))
		}
	}
}

func parseOffset(userData string) (pivot v.Vec) {
	parts := strings.Split(userData, ",")
	pivot.X, _ = strconv.ParseFloat(parts[0], 64)
	pivot.Y, _ = strconv.ParseFloat(parts[1], 64)
	return
}

func StrokeImageRectangle(geo ebiten.GeoM, r image.Rectangle, screen *ebiten.Image) {
	x, y := geo.Apply(float64(r.Min.X), float64(r.Min.Y))
	vector.StrokeRect(
		screen,
		float32(x),
		float32(y),
		float32(r.Dx()),
		float32(r.Dy()),
		1,
		colornames.Cyan,
		false,
	)
}
