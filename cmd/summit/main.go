package main

import (
	"github.com/RhykerWells/Summit/common/run"
)

func main() {
	run.Init()
	initPlugins()
	run.Run()
}
