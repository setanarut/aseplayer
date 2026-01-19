package main

import (
	"fmt"

	"github.com/setanarut/aseplayer/aseparser"
)

var ase = aseparser.NewAsepriteFromFile("../assets/slice.ase")

func main() {

	for _, VisiblelayerUserData := range ase.LayerData {
		fmt.Println(string(VisiblelayerUserData))
	}
	for _, tag := range ase.Tags {
		fmt.Println(tag.UserData)
	}

}
