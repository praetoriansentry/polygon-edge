package main

import (
	_ "embed"
	"github.com/0xPolygon/polygon-edge/command/root"
	"github.com/0xPolygon/polygon-edge/licenses"
	"net/http"
	_ "net/http/pprof"
)

var (
	//go:embed LICENSE
	license string
)

func main() {
	licenses.SetLicense(license)
	go func() {
		http.ListenAndServe("localhost:6060", nil)
	}()
	root.NewRootCommand().Execute()
}
