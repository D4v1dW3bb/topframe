package main

import (
	"net"

	"github.com/progrium/watcher"
)

var (
	startupCommand  = "batchfile"
	startupText     = "Generate starup batch file"
	startupFileName = "topframe.bat"
)

// TODO: Implement windows functionality
func runApp(dir string, addr *net.TCPAddr, fw *watcher.Watcher) {

}
