package main

import (
	"github.com/PbrtCraft/pbrtcraftdrv/filetree"
	"fmt"
)

func main() {
	ft, err := filetree.GetFolder("../")
	if err != nil {
		fmt.Println(err)
		return
	}
	ft.Print()
}
