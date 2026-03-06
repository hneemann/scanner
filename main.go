package main

import (
	"context"
	"errors"
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

func (p persist) Init(fs fileSys.FileSystem, d *data.UserData) error {
	d.SetFileSystem(fs)
	return nil
}

func (p persist) Load(fs fileSys.FileSystem) (*data.UserData, error) {
	return data.Load(fs)
}

func (p persist) Save(fs fileSys.FileSystem, data *data.UserData) error {
	return data.Save(fs)
}

type anonymousDataManager struct {
	parent session.Manager[data.UserData]
}

func (a anonymousDataManager) DeleteOldUsers(maxAge time.Duration) error {
	return a.parent.DeleteOldUsers(maxAge)
}

func (a anonymousDataManager) DoesUserExist(user string) bool {
	return a.parent.DoesUserExist(user)
}

func (a anonymousDataManager) CreateUser(user, pass string) (*data.UserData, error) {
	return a.parent.CreateUser(user, pass)
}

func (a anonymousDataManager) CheckPassword(user, pass string) bool {
	if user == "anonymous" && pass == "" {
		return true
	}
	return a.parent.CheckPassword(user, pass)
}

func (a anonymousDataManager) ChangePassword(user, oldPass, newPass string) error {
	if user == "anonymous" && oldPass == "" {
		return errors.New("password for anonymous user cannot be changed")
	}
	return a.parent.ChangePassword(user, oldPass, newPass)
}

func (a anonymousDataManager) CreatePersist(user, pass string) (session.Persist[data.UserData], error) {
	return a.parent.CreatePersist(user, pass)
}

func main() {
	dataFolder := flag.String("folder", "", "data folder")
	cert := flag.String("cert", "", "certificate")
	key := flag.String("key", "", "certificate")
	port := flag.Int("port", 8080, "port")
	debug := flag.Bool("debug", false, "debug mode. In this mode, the server does not enable browser caching. Also, user 'admin' with password 'admin' is created with a fixed session token. This does not work if OIDC is used!")
	anonymous := flag.Bool("anon", false, "allows to log in anonymously to user anonymous!")
	flag.Parse()

	log.Println("Folder:", *dataFolder)

	mux := http.NewServeMux()

	var manager session.Manager[data.UserData]
	manager = session.NewFileManager[data.UserData](
		session.NewFileSystemFactory(*dataFolder),
		persist{})

	if *anonymous {
		log.Println("anonymous login enabled")
		manager = anonymousDataManager{manager}
	}

	sc := session.NewSessionCache[data.UserData](
		manager,
		3*time.Hour, time.Hour)
	if *debug {
		err := sc.CreateDebugSession("admin", "admin", "debugTokenForAdmin")
		if err != nil {
			log.Fatal(err)
		} /**/
	}
	defer sc.Close()

	mux.HandleFunc("/login", sc.LoginHandler(server.Templates.Lookup("login.html")))
	mux.HandleFunc("/register", sc.RegisterHandler(server.Templates.Lookup("register.html")))
	mux.HandleFunc("/logout", sc.LogoutHandler(server.Templates.Lookup("logout.html")))

	mux.Handle("/assets/", Cache(http.FileServer(http.FS(server.Assets)), 180, !*debug))
	mux.HandleFunc("/", sc.CheckSessionFunc(server.Main))
	mux.HandleFunc("/store/", sc.CheckSessionRestFunc(server.Store))
	mux.HandleFunc("/create/", sc.CheckSessionRestFunc(server.Create))
	mux.HandleFunc("/documents/", sc.CheckSessionFunc(server.Documents))
	if *anonymous {
		mux.HandleFunc("/anonymous/", server.LoginAnonymous(sc))
	}

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
