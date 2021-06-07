package main

import (
	"fmt"
	"log"
	"net"
	"strings"

	"github.com/lxn/walk"
	"github.com/lxn/walk/declarative"
	"github.com/lxn/win"
	"github.com/progrium/watcher"
	//. "github.com/lxn/walk/declarative"
)

var (
	startupCommand  = "batchfile"
	startupText     = "Generate starup batch file"
	startupFileName = "topframe.bat"
)

// TODO: Implement windows functionality
func runApp(dir string, addr *net.TCPAddr, fw *watcher.Watcher) {
	var mainWindow *walk.MainWindow
	var webView *walk.WebView
	serverURL := fmt.Sprintf("http://%s:%d/index.html", addr.IP, addr.Port)

	log.Print("Starting to create webview window")
	declarative.MainWindow{
		AssignTo: &mainWindow,
		Title:    "WebCmd Webview",
		MinSize:  declarative.Size{600, 400},
		Size:     declarative.Size{960, 720},
		Visible:  true,
		Layout:   declarative.VBox{},
		Children: []declarative.Widget{
			declarative.WebView{
				AssignTo: &webView,
				Name:     "wv",
				URL:      serverURL,
			},
		},
		Functions: map[string]func(args ...interface{}) (interface{}, error){
			"icon": func(args ...interface{}) (interface{}, error) {
				if strings.HasPrefix(args[0].(string), "https") {
					return "check", nil
				}

				return "stop", nil
			},
		},
	}.Create()

	log.Print("Create complete, initializing webView with URL ", serverURL)

	win.SetWindowLong(mainWindow.Handle(), win.GWL_EXSTYLE, win.GetWindowLong(mainWindow.Handle(), win.GWL_EXSTYLE)|win.WS_EX_LAYERED|win.WS_EX_NOACTIVATE)
	win.SetWindowPos(mainWindow.Handle(), win.HWND_TOPMOST, 0, 0, 0, 0, win.SWP_NOSIZE|win.SWP_NOMOVE|win.SWP_NOACTIVATE|win.SWP_SHOWWINDOW)
	SetLayeredWindowAttributes(mainWindow.Handle(), win.RGB(255, 255, 255), 0, LWA_COLORKEY)

	mainWindow.SetFullscreen(true)
	mainWindow.SetEnabled(false)
	go func() {
		for {
			select {
			case event := <-fw.Event:
				if event.IsDir() {
					continue
				}
				log.Print("Set URL: ", serverURL)
				webView.SetURL(serverURL)
			case <-fw.Closed:
				return
			}
		}
	}()
	mainWindow.Run()

}
