package aseplayer

import (
	"os"
	"testing"
	"time"
)

const (
	tagFly    = "fly"
	tagSubFly = "sub_fly"
	testAse   = "testdata/bird.ase"
)

var ase *AnimPlayer

func TestMain(m *testing.M) {
	ase = NewAnimPlayerFromAsepriteFile("testfiles/bird.ase", Trim)
	exitCode := m.Run()
	os.Exit(exitCode)
}

func TestAnimationWorkflow(t *testing.T) {
	// 1. tags
	t.Run("Check Tags", func(t *testing.T) {
		anim1 := ase.Animations[tagFly]
		anim2 := ase.Animations[tagSubFly]
		if anim1 == nil || anim2 == nil {
			t.Fatal("Tags missing")
		}
	})

	// 2. frame pointers
	t.Run("Frame Image Pointers", func(t *testing.T) {
		if ase.Animations[tagFly].Frames[2].Image != ase.Animations[tagSubFly].Frames[0].Image {
			t.Errorf("Sub-tag images are not equal!")
		}
		if ase.Animations[tagSubFly].Frames[1].Image != ase.Animations[tagSubFly].Frames[3].Image {
			t.Errorf("ping-pong images are not equal!")
		}
	})
	// 2. Durations
	t.Run("Durations", func(t *testing.T) {
		want := 100 * time.Millisecond
		dur := ase.Animations[tagSubFly].Frames[0].Duration
		if dur != want {
			t.Errorf("Duration is wrong! Got %v, want %v", dur, want)
		}

		dur = ase.Animations[tagFly].Frames[0].Duration
		want = 62 * time.Millisecond
		if dur != want {
			t.Errorf("Duration is wrong! Got %v, want %v", dur, want)
		}
	})
}
