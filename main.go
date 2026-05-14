package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sync"

	"github.com/fsnotify/fsnotify"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type BrowserClient struct {
	conn *websocket.Conn
	send chan []byte
}

type WebsServer struct {
	clients    map[*BrowserClient]bool
	broadcast  chan []byte
	register   chan *BrowserClient
	unregister chan *BrowserClient
	mu         sync.RWMutex
	watcher    *fsnotify.Watcher
	port       int
	dir        string
	watchExts  []string
}

func NewWebsServer(port int, dir string, watchExts []string) *WebsServer {
	return &WebsServer{
		clients:    make(map[*BrowserClient]bool),
		broadcast:  make(chan []byte),
		register:   make(chan *BrowserClient),
		unregister: make(chan *BrowserClient),
		port:       port,
		dir:        dir,
		watchExts:  watchExts,
	}
}

func (s *WebsServer) run() {
	for {
		select {
		case client := <-s.register:
			s.mu.Lock()
			s.clients[client] = true
			s.mu.Unlock()
			log.Printf("Client connected. Total clients: %d", len(s.clients))

		case client := <-s.unregister:
			s.mu.Lock()
			if _, ok := s.clients[client]; ok {
				delete(s.clients, client)
				close(client.send)
			}
			s.mu.Unlock()
			log.Printf("Client disconnected. Total clients: %d", len(s.clients))

		case message := <-s.broadcast:
			s.mu.RLock()
			for client := range s.clients {
				select {
				case client.send <- message:
				default:
					close(client.send)
					delete(s.clients, client)
				}
			}
			s.mu.RUnlock()
		}
	}
}

func (s *WebsServer) broadcastReload() {
	message := []byte(`{"command":"reload","path":"/"}`)
	s.broadcast <- message
	log.Println("Broadcasting reload signal to all clients")
}

func (s *WebsServer) shouldReload(name string) bool {
	if len(s.watchExts) == 0 {
		return true
	}
	ext := filepath.Ext(name)
	for _, e := range s.watchExts {
		if ext == "."+e || ext == e {
			return true
		}
	}
	return false
}

func (s *WebsServer) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}

	client := &BrowserClient{
		conn: conn,
		send: make(chan []byte, 256),
	}

	s.register <- client

	go client.writePump()
	go client.readPump(s)
}

func (c *BrowserClient) writePump() {
	defer func() {
		c.conn.Close()
	}()

	for message := range c.send {
		if err := c.conn.WriteMessage(websocket.TextMessage, message); err != nil {
			return
		}
	}
}

func (c *BrowserClient) readPump(s *WebsServer) {
	defer func() {
		s.unregister <- c
		c.conn.Close()
	}()

	for {
		_, _, err := c.conn.ReadMessage()
		if err != nil {
			break
		}
	}
}

func (s *WebsServer) startFileWatcher() error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	s.watcher = watcher

	dir := s.dir
	if dir == "" {
		dir = "."
	}

	if err := watcher.Add(dir); err != nil {
		return err
	}

	go func() {
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				if event.Op&(fsnotify.Write|fsnotify.Create|fsnotify.Remove|fsnotify.Rename) != 0 {
					if s.shouldReload(event.Name) {
						log.Printf("File changed: %s", event.Name)
						s.broadcastReload()
					}
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				log.Printf("Watcher error: %v", err)
			}
		}
	}()

	return nil
}

func (s *WebsServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/ws" {
		s.handleWebSocket(w, r)
		return
	}

	if r.URL.Path == "/favicon.ico" {
		w.Header().Set("Content-Type", "image/x-icon")
		w.Write([]byte{0, 0, 1, 0, 1, 0, 16, 16, 0, 0, 1, 0, 32, 0, 104, 4, 0, 0, 22, 0, 0, 0, 40, 0, 0, 0, 16, 0, 0, 0, 32, 0, 0, 0, 1, 0, 32, 0, 0, 0, 0, 0, 0, 4, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0})
		return
	}

	staticDir := s.dir
	if staticDir == "" {
		staticDir = "."
	}

	filePath := filepath.Join(staticDir, r.URL.Path)

	if r.URL.Path == "/" || r.URL.Path == "" {
		indexPath := filepath.Join(staticDir, "index.html")
		if _, err := os.Stat(indexPath); err == nil {
			http.ServeFile(w, r, indexPath)
			return
		}
	}

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		http.NotFound(w, r)
		return
	}

	http.ServeFile(w, r, filePath)
}

func openBrowser(url string) {
	var err error
	switch runtime.GOOS {
	case "windows":
		err = exec.Command("cmd", "/c", "start", url).Run()
	case "darwin":
		err = exec.Command("open", url).Run()
	default:
		err = exec.Command("xdg-open", url).Run()
	}
	if err != nil {
		log.Printf("Failed to open browser: %v", err)
	}
}

func getAvailablePort(preferred int) int {
	if ln, err := net.Listen("tcp", fmt.Sprintf(":%d", preferred)); err == nil {
		ln.Close()
		return preferred
	}
	for i := 0; i < 100; i++ {
		port := 49152 + i
		if ln, err := net.Listen("tcp", fmt.Sprintf(":%d", port)); err == nil {
			ln.Close()
			return port
		}
	}
	return 0
}

func main() {
	cfg := loadConfig()

	port := flag.Int("port", cfg.Port, "Server port")
	dir := flag.String("dir", cfg.Dir, "Directory to serve")
	flag.Parse()

	actualPort := getAvailablePort(*port)
	if actualPort != *port {
		log.Printf("Port %d is in use, using random port: %d", *port, actualPort)
	}
	if actualPort == 0 {
		log.Fatal("Could not find available port")
	}

	server := NewWebsServer(actualPort, *dir, cfg.WatchExts)

	go server.run()

	addr := fmt.Sprintf(":%d", actualPort)
	log.Printf("Webs Server starting on http://localhost:%d", actualPort)
	log.Printf("Serving directory: %s", *dir)
	log.Println("Press Ctrl+C to stop")

	url := fmt.Sprintf("http://localhost:%d", actualPort)
	go openBrowser(url)

	go func() {
		if err := server.startFileWatcher(); err != nil {
			log.Printf("Warning: Could not start file watcher: %v", err)
		} else {
			log.Println("File watcher started")
		}
	}()

	if err := http.ListenAndServe(addr, server); err != nil {
		log.Fatal(err)
	}
}
