package main

import (
	"fmt"
	"log"
	"net"
	"os/exec"
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
		Title:    "Topframe",
		MinSize:  declarative.Size{Height: 600, Width: 400},
		Size:     declarative.Size{Height: 960, Width: 720},
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

	win.SetWindowLong(mainWindow.Handle(), win.GWL_EXSTYLE, win.GetWindowLong(mainWindow.Handle(), win.GWL_EXSTYLE)|win.WS_EX_LAYERED|win.WS_EX_TRANSPARENT)
	win.SetWindowPos(mainWindow.Handle(), win.HWND_TOPMOST, 0, 0, 0, 0, win.SWP_NOSIZE|win.SWP_NOMOVE|win.SWP_SHOWWINDOW)
	SetLayeredWindowAttributes(mainWindow.Handle(), win.RGB(255, 255, 255), 0, LWA_COLORKEY)

	mainWindow.SetFullscreen(true)
	mainWindow.SetEnabled(false)
	// We load our icon from a file.
	icon, err := walk.Resources.Icon("./data/stop.ico")
	if err != nil {
		log.Fatal(err)
	}

	// Create the notify icon and make sure we clean it up on exit.
	ni, err := walk.NewNotifyIcon(mainWindow)
	if err != nil {
		log.Fatal(err)
	}
	defer ni.Dispose()

	// Set the icon and a tool tip text.
	if err := ni.SetIcon(icon); err != nil {
		log.Fatal(err)
	}
	if err := ni.SetToolTip("Click for info or use the context menu to exit."); err != nil {
		log.Fatal(err)
	}

	// When the left mouse button is pressed, bring up our balloon.
	ni.MouseDown().Attach(func(x, y int, button walk.MouseButton) {
		if button != walk.LeftButton {
			return
		}

		if err := ni.ShowCustom(
			"Walk NotifyIcon Example",
			"There are multiple ShowX methods sporting different icons.",
			icon); err != nil {

			log.Fatal(err)
		}
	})

	// Set Source action in the context menu. This action opens explorer in the .topframe directory
	menuSource := walk.NewAction()
	if err := menuSource.SetText("Source"); err != nil {
		log.Fatal(err)
	}

	menuSource.Triggered().Attach(func() { exec.Command(`explorer`, dir).Run() })
	if err := ni.ContextMenu().Actions().Add(menuSource); err != nil {
		log.Fatal(err)
	}
	// Set a separator in the context menu
	if err := ni.ContextMenu().Actions().Add(walk.NewSeparatorAction()); err != nil {
		log.Fatal(err)
	}
	// Set Interact action in the context menu.
	menuInteract := walk.NewAction()
	if err := menuInteract.SetText("Interact"); err != nil {
		log.Fatal(err)
	}
	menuInteract.SetChecked(false)
	menuInteract.Triggered().Attach(func() {
		if mainWindow.Enabled() {
			props := win.GetWindowLong(mainWindow.Handle(), win.GWL_EXSTYLE) &^ win.WS_OVERLAPPEDWINDOW
			win.SetWindowLong(mainWindow.Handle(), win.GWL_EXSTYLE, props|win.WS_EX_TRANSPARENT)
			mainWindow.SetEnabled(false)
			menuInteract.SetChecked(false)
		} else {
			props := win.GetWindowLong(mainWindow.Handle(), win.GWL_EXSTYLE) &^ win.WS_EX_TRANSPARENT
			win.SetWindowLong(mainWindow.Handle(), win.GWL_EXSTYLE, props|win.WS_OVERLAPPEDWINDOW)
			mainWindow.SetEnabled(true)
			menuInteract.SetChecked(true)
		}
	})

	if err := ni.ContextMenu().Actions().Add(menuInteract); err != nil {
		log.Fatal(err)
	}

	// Set Enable action in the context menu.
	menuEnable := walk.NewAction()
	if err := menuEnable.SetText("Enabled"); err != nil {
		log.Fatal(err)
	}
	menuEnable.SetChecked(true)
	menuEnable.Triggered().Attach(func() {
		if mainWindow.Visible() {
			mainWindow.SetVisible(false)
			menuEnable.SetChecked(false)
		} else {
			mainWindow.SetVisible(true)
			menuEnable.SetChecked(true)
		}
	})

	if err := ni.ContextMenu().Actions().Add(menuEnable); err != nil {
		log.Fatal(err)
	}

	// Set a separator in the context menu.
	if err := ni.ContextMenu().Actions().Add(walk.NewSeparatorAction()); err != nil {
		log.Fatal(err)
	}

	// Set Quit action in the context menu.
	menuQuit := walk.NewAction()
	if err := menuQuit.SetText("Quit"); err != nil {
		log.Fatal(err)
	}
	menuQuit.Triggered().Attach(func() { walk.App().Exit(0) })
	if err := ni.ContextMenu().Actions().Add(menuQuit); err != nil {
		log.Fatal(err)
	}

	// The notify icon is hidden initially, so we have to make it visible.
	if err := ni.SetVisible(true); err != nil {
		log.Fatal(err)
	}

	// Now that the icon is visible, we can bring up an info balloon.
	if err := ni.ShowInfo("Walk NotifyIcon Example", "Click the icon to show again."); err != nil {
		log.Fatal(err)
	}

	go func() {
		for {
			select {
			case event := <-fw.Event:
				if event.IsDir() {
					continue
				}
				log.Print(mainWindow.Size())
				webView.SetURL(serverURL)
			case <-fw.Closed:
				return
			}
		}
	}()
	mainWindow.Run()

}
