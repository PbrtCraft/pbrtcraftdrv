package main

import (
	"github.com/PbrtCraft/pbrtcraftdrv/parsepy"
	"fmt"
)

func main() {
	cs, _ := parsepy.GetClasses("test.py")
	for _, c := range cs {
		fmt.Println("----------")
		fmt.Println(c.Name)
		fmt.Println(c.InitFunc.Def)
		fmt.Println(c.InitFunc.Doc)
		for _, param := range c.InitFunc.Params {
			fmt.Println(param)
		}
	}
}
