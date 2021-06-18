package main

import (
	"bufio"
	"embed"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"runtime"
	"strings"
	"text/template"
	"time"

	"github.com/progrium/watcher"
)

var (
	Version = readVersion()

	docsURL = fmt.Sprintf("http://%s", getPackageName())
)

//go:embed data
var data embed.FS

func init() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	runtime.LockOSThread()
}

func main() {
	var (
		flagHelp          = flag.Bool("help", false, "show help")
		flagHelpShort     = flag.Bool("h", false, "show help")
		flagVersion       = flag.Bool("version", false, "show version")
		flagVersionShort  = flag.Bool("v", false, "show help")
		flagDocs          = flag.Bool("docs", false, "open documentation in browser")
		flagStartupScript = flag.Bool("startupscript", false, startupText)
	)
	flag.Parse()

	if *flagHelp || *flagHelpShort {
		printHelp()
		return
	}
	if *flagVersion || *flagVersionShort {
		fmt.Println(Version)
		return
	}
	if *flagDocs {
		switch runtime.GOOS {
		case "windows":
			fatal(exec.Command("rundll32", "url.dll,FileProtocolHandler", docsURL).Start())
		case "darwin":
			fatal(exec.Command("open", docsURL).Start())
		default:
			fatal(fmt.Errorf("unsupported platform"))
		}
		return
	}

	dir := ensureDir()

	if *flagStartupScript {
		generateStartupFile(dir)
		return
	}

	addr := startServer(dir)
	fw := startWatcher(dir)
	runApp(dir, addr, fw)

}

func ensureDir() (dir string) {
	usr, err := user.Current()
	fatal(err)

	if os.Getenv("TOPFRAME_DIR") != "" {
		dir = os.Getenv("TOPFRAME_DIR")
	} else {
		dir = filepath.Join(usr.HomeDir, ".topframe")
	}

	os.MkdirAll(dir, 0755)

	if _, err := os.Stat(filepath.Join(dir, "index.html")); os.IsNotExist(err) {
		ioutil.WriteFile(filepath.Join(dir, "index.html"), mustReadFile(data, "data/index.html"), 0644)
	}

	if _, err := os.Stat(filepath.Join(dir, "stocks")); os.IsNotExist(err) {
		ioutil.WriteFile(filepath.Join(dir, "stocks"), mustReadFile(data, "data/stocks"), 0644)
	}

	return dir
}

// TODO: Check functionality on windows with .bat file
func generateStartupFile(dir string) {
	tmpl, err := template.New(startupCommand).Parse(string(mustReadFile(data, fmt.Sprintf("data/%s", startupFileName))))
	fatal(err)

	p, err := exec.LookPath(os.Args[0])
	fatal(err)

	bin, _ := filepath.Abs(p)
	fatal(tmpl.Execute(os.Stdout, struct {
		Dir, Bin string
	}{
		Dir: dir,
		Bin: bin,
	}))
}

func startServer(dir string) *net.TCPAddr {
	srv := http.Server{
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			dirpath := filepath.Join(dir, r.URL.Path)
			if isExecScript(dirpath) && r.Header.Get("Accept") == "text/event-stream" {
				streamExecScript(w, dirpath, strings.Split(r.URL.RawQuery, "+"))
				return
			}
			if strings.HasPrefix(r.URL.Path, "/-/") {
				http.StripPrefix("/-/", http.FileServer(http.FS(data))).ServeHTTP(w, r)
				return
			}
			http.FileServer(http.Dir(dir)).ServeHTTP(w, r)
		}),
	}

	addr := "localhost:0"
	if os.Getenv("TOPFRAME_ADDR") != "" {
		addr = os.Getenv("TOPFRAME_ADDR")
	}
	ln, err := net.Listen("tcp", addr)
	fatal(err)

	go srv.Serve(ln)

	return ln.Addr().(*net.TCPAddr)
}

func startWatcher(dir string) *watcher.Watcher {
	fw := watcher.New()
	fatal(fw.AddRecursive(dir))

	go fw.Start(450 * time.Millisecond)
	return fw
}

func streamExecScript(w http.ResponseWriter, dirpath string, args []string) {
	flusher, ok := w.(http.Flusher)
	if !ok || !isExecScript(dirpath) {
		http.Error(w, "script unsupported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	cmd := exec.Command(dirpath, args...)
	cmd.Stderr = os.Stderr
	r, _ := cmd.StdoutPipe()
	scanner := bufio.NewScanner(r)

	finished := make(chan bool)
	go func() {
		for scanner.Scan() {
			_, err := io.WriteString(w, fmt.Sprintf("event: stdout\ndata: %s\n\n", scanner.Text()))
			if err != nil {
				log.Println("script:", err)
				return
			}
			flusher.Flush()
		}
		if err := scanner.Err(); err != nil {
			log.Println("script:", err)
		}
		finished <- true
	}()

	if err := cmd.Run(); err != nil {
		log.Println(err)
	}
	<-finished
}

func isExecScript(dirpath string) bool {
	fi, err := os.Stat(dirpath)
	if err != nil {
		return false
	}
	return fi.Mode()&0111 != 0
}

func mustReadFile(fs embed.FS, name string) []byte {
	b, err := fs.ReadFile(name)
	fatal(err)
	return b
}

func printHelp() {
	fmt.Printf("Usage: topframe <flags>\n")
	fmt.Printf("Topframe is a fullscreen webview overlay agent\n\n")
	fmt.Printf("Flags:\n")
	flag.VisitAll(func(f *flag.Flag) {
		if len(f.Name) > 1 {
			fmt.Printf("  -%-10s %s\n", f.Name, f.Usage)
		}
	})
}

func getPackageName() string {
	output, _ := exec.Command("go", "list", "-m").CombinedOutput()
	return strings.TrimSpace(string(output))
}

func readVersion() string {
	file, err := os.Open("version")
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		if err = file.Close(); err != nil {
			log.Fatal(err)
		}
	}()

	b, err := ioutil.ReadAll(file)

	return string(b)
}
