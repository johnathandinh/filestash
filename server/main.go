package main

import (
	"github.com/mickael-kerjean/mux"
	. "github.com/mickael-kerjean/nuage/server/common"
	. "github.com/mickael-kerjean/nuage/server/ctrl"
	. "github.com/mickael-kerjean/nuage/server/middleware"
	_ "github.com/mickael-kerjean/nuage/server/plugin"
	"net/http"
    "net/http/pprof"
	"os"
	"runtime/debug"
	"strconv"
)

func main() {
	app := App{}
	app.Config = NewConfig()
	Log.SetVisibility(app.Config.Get("log.level").String())
	Init(&app)
}

func Init(a *App) {
	r := mux.NewRouter()

	// Profiling - handy to indentify nasty leaks and/or bugs!
	if os.Getenv("DEBUG") == "true" {
		r.HandleFunc("/debug/pprof/", pprof.Index)
		r.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
		r.HandleFunc("/debug/pprof/profile", pprof.Profile)
		r.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
		r.HandleFunc("/debug/pprof/trace", pprof.Trace)
		r.Handle("/debug/pprof/goroutine", pprof.Handler("goroutine"))
		r.Handle("/debug/pprof/heap", pprof.Handler("heap"))
		r.Handle("/debug/pprof/threadcreate", pprof.Handler("threadcreate"))
		r.Handle("/debug/pprof/block", pprof.Handler("block"))
		r.HandleFunc("/debug/free", func(w http.ResponseWriter, r *http.Request) {
			debug.FreeOSMemory()
			w.Write([]byte("DONE"))
		})
	}

	// API
	session := r.PathPrefix("/api/session").Subrouter()
	session.HandleFunc("", APIHandler(SessionGet, *a)).Methods("GET")
	session.HandleFunc("", APIHandler(SessionAuthenticate, *a)).Methods("POST")
	session.HandleFunc("", CtxInjector(SessionLogout, *a)).Methods("DELETE")
	session.Handle("/auth/{service}", APIHandler(SessionOAuthBackend, *a)).Methods("GET")

	files := r.PathPrefix("/api/files").Subrouter()
	files.HandleFunc("/ls", APIHandler(LoggedInOnly(FileLs), *a)).Methods("GET")
	files.HandleFunc("/cat", APIHandler(LoggedInOnly(FileCat), *a)).Methods("GET")
	files.HandleFunc("/cat", APIHandler(LoggedInOnly(FileSave), *a)).Methods("POST")
	files.HandleFunc("/mv", APIHandler(LoggedInOnly(FileMv), *a)).Methods("GET")
	files.HandleFunc("/rm", APIHandler(LoggedInOnly(FileRm), *a)).Methods("GET")
	files.HandleFunc("/mkdir", APIHandler(LoggedInOnly(FileMkdir), *a)).Methods("GET")
	files.HandleFunc("/touch", APIHandler(LoggedInOnly(FileTouch), *a)).Methods("GET")

	share := r.PathPrefix("/api/share").Subrouter()
	share.HandleFunc("", APIHandler(ShareList, *a)).Methods("GET")
	share.HandleFunc("/{share}", APIHandler(ShareUpsert, *a)).Methods("POST")
	share.HandleFunc("/{share}", APIHandler(ShareDelete, *a)).Methods("DELETE")
	share.HandleFunc("/{share}/proof", APIHandler(ShareVerifyProof, *a)).Methods("POST")

	// WEBDAV
	r.PathPrefix("/s/{share}").Handler(CtxInjector(WebdavHandler, *a))

	// APP
	r.HandleFunc("/api/config", CtxInjector(ConfigHandler, *a)).Methods("GET")
	r.PathPrefix("/assets").Handler(StaticHandler(FILE_ASSETS, *a)).Methods("GET")
	r.PathPrefix("/about").Handler(AboutHandler(*a))
	r.PathPrefix("/").Handler(DefaultHandler(FILE_INDEX, *a)).Methods("GET")

	srv := &http.Server{
		Addr:    ":" + strconv.Itoa(a.Config.Get("general.port").Int()),
		Handler: r,
	}
	Log.Info("STARTING SERVER")
	if err := srv.ListenAndServe(); err != nil {
		Log.Error("Server start: %v", err)
		return
	}
}
