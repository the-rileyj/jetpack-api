package main

import (
	"github.com/the-rileyj/jetpack-api/functionality"
)

func main() {
	functionality.GetJetpackRouter().Run(":80")
}
