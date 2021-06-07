package main

import (
	"fmt"
	"log"
	"net"
	"os/exec"

	"github.com/D4v1dW3bb/topframe/errorHandling"
	"github.com/progrium/macdriver/cocoa"
	"github.com/progrium/macdriver/core"
	"github.com/progrium/macdriver/objc"
	"github.com/progrium/macdriver/webkit"
	"github.com/progrium/watcher"
)

var (
	startupCommand  = "plist"
	startupText     = "Generate launch agent plist"
	startupFileName = "agent.plist"
)

func RunApp(dir string, addr *net.TCPAddr, fw *watcher.Watcher) {
	cocoa.TerminateAfterWindowsClose = false

	config := webkit.WKWebViewConfiguration_New()
	config.Preferences().SetValueForKey(core.True, core.String("developerExtrasEnabled"))

	url := core.URL(fmt.Sprintf("http://%d:%d", addr.IP, addr.Port))
	req := core.NSURLRequest_Init(url)

	app := cocoa.NSApp_WithDidLaunch(func(_ objc.Object) {
		wv := webkit.WKWebView_Init(cocoa.NSScreen_Main().Frame(), config)
		wv.Retain()
		wv.SetOpaque(false)
		wv.SetBackgroundColor(cocoa.NSColor_Clear())
		wv.SetValueForKey(core.False, core.String("drawsBackground"))
		wv.LoadRequest(req)

		win := cocoa.NSWindow_Init(cocoa.NSScreen_Main().Frame(),
			cocoa.NSClosableWindowMask|cocoa.NSBorderlessWindowMask,
			cocoa.NSBackingStoreBuffered, false)
		win.SetContentView(wv)
		win.SetBackgroundColor(cocoa.NSColor_Clear())
		win.SetOpaque(false)
		win.SetTitleVisibility(cocoa.NSWindowTitleHidden)
		win.SetTitlebarAppearsTransparent(true)
		win.SetIgnoresMouseEvents(true)
		win.SetLevel(cocoa.NSMainMenuWindowLevel + 2)
		win.MakeKeyAndOrderFront(win)
		win.SetCollectionBehavior(cocoa.NSWindowCollectionBehaviorCanJoinAllSpaces)
		win.Send("setHasShadow:", false)

		statusBar := cocoa.NSStatusBar_System().StatusItemWithLength(cocoa.NSVariableStatusItemLength)
		statusBar.Retain()
		statusBar.Button().SetTitle("🔲")

		menuInteract := cocoa.NSMenuItem_New()
		menuInteract.Retain()
		menuInteract.SetTitle("Interactive")
		menuInteract.SetAction(objc.Sel("interact:"))
		cocoa.DefaultDelegateClass.AddMethod("interact:", func(_ objc.Object) {
			if win.IgnoresMouseEvents() {
				win.SetLevel(cocoa.NSMainMenuWindowLevel - 1)
				win.SetIgnoresMouseEvents(false)
				menuInteract.SetState(1)
			} else {
				win.SetIgnoresMouseEvents(true)
				win.SetLevel(cocoa.NSMainMenuWindowLevel + 2)
				menuInteract.SetState(0)
			}
		})

		menuEnabled := cocoa.NSMenuItem_New()
		menuEnabled.Retain()
		menuEnabled.SetTitle("Enabled")
		menuEnabled.SetState(1)
		menuEnabled.SetAction(objc.Sel("enabled:"))
		cocoa.DefaultDelegateClass.AddMethod("enabled:", func(_ objc.Object) {
			if win.IsVisible() {
				win.Send("orderOut:", win)
				menuInteract.SetEnabled(false)
				menuEnabled.SetState(0)
			} else {
				win.Send("orderFront:", win)
				menuInteract.SetEnabled(true)
				menuEnabled.SetState(1)
			}
		})

		menuSource := cocoa.NSMenuItem_New()
		menuSource.SetTitle("Show Source...")
		menuSource.SetAction(objc.Sel("source:"))
		cocoa.DefaultDelegateClass.AddMethod("source:", func(_ objc.Object) {
			go func() {
				errorHandling.Fatal(exec.Command("open", dir).Run())
			}()
		})

		menuQuit := cocoa.NSMenuItem_New()
		menuQuit.SetTitle("Quit")
		menuQuit.SetAction(objc.Sel("terminate:"))

		menu := cocoa.NSMenu_New()
		menu.SetAutoenablesItems(false)
		menu.AddItem(menuEnabled)
		menu.AddItem(menuInteract)
		menu.AddItem(cocoa.NSMenuItem_Separator())
		menu.AddItem(menuSource)
		menu.AddItem(cocoa.NSMenuItem_Separator())
		menu.AddItem(menuQuit)

		statusBar.SetMenu(menu)

		go func() {
			for {
				select {
				case event := <-fw.Event:
					if event.IsDir() {
						continue
					}
					wv.Reload(nil)
				case <-fw.Closed:
					return
				}
			}
		}()
	})

	log.Printf("topframe %s from progrium.com\n", Version)
	app.ActivateIgnoringOtherApps(true)
	app.Run()
}
