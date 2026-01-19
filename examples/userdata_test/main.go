package main

import (
	"fmt"

	"github.com/setanarut/aseplayer/aseparser"
)

var ase = aseparser.NewAsepriteFromFile("../assets/slice.ase")

func main() {

	for _, f := range ase.Frames {
		fmt.Println("Bounds", f.Bounds)
		fmt.Println("Cel bounds", f.CelBounds)
	}

	for i, frame1LayerCelUserData := range ase.Frames[0].Layers {
		fmt.Println(i, frame1LayerCelUserData)
	}

	for _, VisiblelayerUserData := range ase.LayerData {
		fmt.Println(string(VisiblelayerUserData))
	}
	for _, tag := range ase.Tags {
		fmt.Println(tag.UserData)
	}

}
