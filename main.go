package main

import (
	"context"
	"flag"
	"github.com/hneemann/session"
	"github.com/hneemann/session/fileSys"
	"log"
	"net/http"
	"os"
	"os/signal"
	"scan/data"
	"scan/server"
	"strconv"
	"syscall"
	"time"
)

type persist struct{}

func (p persist) Load(fs fileSys.FileSystem) (*data.UserData, error) {
	return data.Load(fs)
}

func (p persist) Save(fs fileSys.FileSystem, data *data.UserData) error {
	return data.Save(fs)
}

func main() {
	dataFolder := flag.String("folder", "", "data folder")
	cert := flag.String("cert", "", "certificate")
	key := flag.String("key", "", "certificate")
	port := flag.Int("port", 8080, "port")
	debug := flag.Bool("debug", false, "debug mode. In this mode, the server does not enable browser caching. Also, user 'admin' with password 'admin' is created with a fixed session token. This does not work if OIDC is used!")
	flag.Parse()

	log.Println("Folder:", *dataFolder)

	mux := http.NewServeMux()

	sc := session.NewSessionCache[data.UserData](
		session.NewDataManager[data.UserData](
			session.NewFileSystemFactory(*dataFolder),
			persist{}),
		4*time.Hour, time.Hour)
	if *debug {
		err := sc.CreateDebugSession("admin", "admin", "debugTokenForAdmin")
		if err != nil {
			log.Fatal(err)
		} /**/
	}
	defer sc.Close()

	mux.HandleFunc("/login", sc.LoginHandler(server.Templates.Lookup("login.html")))
	mux.HandleFunc("/register", sc.RegisterHandler(server.Templates.Lookup("register.html")))

	mux.Handle("/assets/", Cache(http.FileServer(http.FS(server.Assets)), 180, !*debug))
	mux.HandleFunc("/", sc.CheckSessionFunc(server.Main))
	mux.HandleFunc("/store/", sc.CheckSessionFunc(server.Store))
	mux.HandleFunc("/create/", sc.CheckSessionFunc(server.Create))
	mux.HandleFunc("/documents/", sc.CheckSessionFunc(server.Documents))

	serv := &http.Server{Addr: ":" + strconv.Itoa(*port), Handler: mux}

	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-c
		log.Print("terminated by signal ", sig.String())

		err := serv.Shutdown(context.Background())
		if err != nil {
			log.Println(err)
		}
		for {
			<-c
		}
	}()

	var err error
	if *cert != "" && *key != "" {
		log.Println("Starting server with TLS")
		err = serv.ListenAndServeTLS(*cert, *key)
	} else {
		log.Println("Starting server without TLS")
		err = serv.ListenAndServe()
	}
	if err != nil {
		log.Println(err)
	}

}

func Cache(parent http.Handler, minutes int, enableCache bool) http.Handler {
	if enableCache {
		return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			writer.Header().Add("Cache-Control", "public, max-age="+strconv.Itoa(minutes*60))
			parent.ServeHTTP(writer, request)
		})
	} else {
		log.Println("browser caching disabled")
		return parent
	}
}
