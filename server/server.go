package server

import (
	"bytes"
	"fmt"
	"log"
	"net/http"
	"path/filepath"
	"time"

	"github.com/gorilla/mux"

	"senan.xyz/g/gonic/assets"
	"senan.xyz/g/gonic/db"
	"senan.xyz/g/gonic/scanner"
	"senan.xyz/g/gonic/server/ctrladmin"
	"senan.xyz/g/gonic/server/ctrlbase"
	"senan.xyz/g/gonic/server/ctrlsubsonic"
)

type ServerOptions struct {
	DB           *db.DB
	MusicPath    string
	ListenAddr   string
	ScanInterval time.Duration
}

type Server struct {
	*http.Server
	router   *mux.Router
	ctrlBase *ctrlbase.Controller
	opts     ServerOptions
}

func New(opts ServerOptions) *Server {
	opts.MusicPath = filepath.Clean(opts.MusicPath)
	ctrlBase := &ctrlbase.Controller{
		DB:        opts.DB,
		MusicPath: opts.MusicPath,
		Scanner:   scanner.New(opts.DB, opts.MusicPath),
	}
	router := mux.NewRouter()
	router.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// make the admin page the default
		http.Redirect(w, r, "/admin/home", http.StatusMovedPermanently)
	})
	router.HandleFunc("/musicFolderSettings.view", func(w http.ResponseWriter, r *http.Request) {
		// jamstash seems to call "musicFolderSettings.view" to start a scan. notice
		// that there is no "/rest/" prefix, so i doesn't fit in with the nice router,
		// custom handler, middleware. etc setup that we've got in `SetupSubsonic()`.
		// instead lets redirect to down there and use the scan endpoint
		redirectTo := fmt.Sprintf("/rest/startScan.view?%s", r.URL.Query().Encode())
		http.Redirect(w, r, redirectTo, http.StatusMovedPermanently)
	})
	// common middleware for admin and subsonic routes
	router.Use(ctrlBase.WithLogging)
	router.Use(ctrlBase.WithCORS)
	server := &http.Server{
		Addr:         opts.ListenAddr,
		Handler:      router,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  15 * time.Second,
	}
	return &Server{
		Server:   server,
		router:   router,
		ctrlBase: ctrlBase,
		opts:     opts,
	}
}

func (s *Server) SetupAdmin() error {
	ctrl := ctrladmin.New(s.ctrlBase)
	//
	// begin public routes (creates session)
	routPublic := s.router.PathPrefix("/admin").Subrouter()
	routPublic.Use(ctrl.WithSession)
	routPublic.NotFoundHandler = ctrl.H(ctrl.ServeNotFound)
	routPublic.Handle("/login", ctrl.H(ctrl.ServeLogin))
	routPublic.HandleFunc("/login_do", ctrl.ServeLoginDo) // "raw" handler, updates session
	assets.PrefixDo("static", func(path string, asset *assets.EmbeddedAsset) {
		_, name := filepath.Split(path)
		route := filepath.Join("/static", name)
		reader := bytes.NewReader(asset.Bytes)
		routPublic.HandleFunc(route, func(w http.ResponseWriter, r *http.Request) {
			http.ServeContent(w, r, name, asset.ModTime, reader)
		})
	})
	//
	// begin user routes (if session is valid)
	routUser := routPublic.NewRoute().Subrouter()
	routUser.Use(ctrl.WithUserSession)
	routUser.HandleFunc("/logout", ctrl.ServeLogout) // "raw" handler, updates session
	routUser.Handle("/home", ctrl.H(ctrl.ServeHome))
	routUser.Handle("/change_own_password", ctrl.H(ctrl.ServeChangeOwnPassword))
	routUser.Handle("/change_own_password_do", ctrl.H(ctrl.ServeChangeOwnPasswordDo))
	routUser.Handle("/link_lastfm_do", ctrl.H(ctrl.ServeLinkLastFMDo))
	routUser.Handle("/unlink_lastfm_do", ctrl.H(ctrl.ServeUnlinkLastFMDo))
	routUser.Handle("/link_funk", ctrl.H(ctrl.ServeLinkFunk))
	routUser.Handle("/link_funk_do", ctrl.H(ctrl.ServeLinkFunkDo))
	routUser.Handle("/unlink_funk_do", ctrl.H(ctrl.ServeUnlinkFunkDo))
	routUser.Handle("/upload_playlist_do", ctrl.H(ctrl.ServeUploadPlaylistDo))
	//
	// begin admin routes (if session is valid, and is admin)
	routAdmin := routUser.NewRoute().Subrouter()
	routAdmin.Use(ctrl.WithAdminSession)
	routAdmin.Handle("/change_password", ctrl.H(ctrl.ServeChangePassword))
	routAdmin.Handle("/change_password_do", ctrl.H(ctrl.ServeChangePasswordDo))
	routAdmin.Handle("/delete_user", ctrl.H(ctrl.ServeDeleteUser))
	routAdmin.Handle("/delete_user_do", ctrl.H(ctrl.ServeDeleteUserDo))
	routAdmin.Handle("/create_user", ctrl.H(ctrl.ServeCreateUser))
	routAdmin.Handle("/create_user_do", ctrl.H(ctrl.ServeCreateUserDo))
	routAdmin.Handle("/update_lastfm_api_key", ctrl.H(ctrl.ServeUpdateLastFMAPIKey))
	routAdmin.Handle("/update_lastfm_api_key_do", ctrl.H(ctrl.ServeUpdateLastFMAPIKeyDo))
	routAdmin.Handle("/update_funk_node", ctrl.H(ctrl.ServeUpdateFunkNode))
	routAdmin.Handle("/update_funk_node_do", ctrl.H(ctrl.ServeUpdateFunkNodeDo))
	routAdmin.Handle("/start_scan_do", ctrl.H(ctrl.ServeStartScanDo))
	return nil
}

func (s *Server) SetupSubsonic() error {
	ctrl := ctrlsubsonic.New(s.ctrlBase)
	routPublic := s.router.PathPrefix("/rest").Subrouter()
	routPublic.Handle("/getCoverArt{_:(?:\\.view)?}", ctrl.HR(ctrl.ServeGetCoverArt))
	rout := routPublic.NewRoute().Subrouter()
	rout.Use(ctrl.WithValidSubsonicArgs)
	rout.NotFoundHandler = ctrl.H(ctrl.ServeNotFound)
	//
	// begin common
	rout.Handle("/getLicense{_:(?:\\.view)?}", ctrl.H(ctrl.ServeGetLicence))
	rout.Handle("/getMusicFolders{_:(?:\\.view)?}", ctrl.H(ctrl.ServeGetMusicFolders))
	rout.Handle("/getScanStatus{_:(?:\\.view)?}", ctrl.H(ctrl.ServeGetScanStatus))
	rout.Handle("/ping{_:(?:\\.view)?}", ctrl.H(ctrl.ServePing))
	rout.Handle("/scrobble{_:(?:\\.view)?}", ctrl.H(ctrl.ServeScrobble))
	rout.Handle("/startScan{_:(?:\\.view)?}", ctrl.H(ctrl.ServeStartScan))
	rout.Handle("/getUser{_:(?:\\.view)?}", ctrl.H(ctrl.ServeGetUser))
	rout.Handle("/getPlaylists{_:(?:\\.view)?}", ctrl.H(ctrl.ServeGetPlaylists))
	rout.Handle("/getPlaylist{_:(?:\\.view)?}", ctrl.H(ctrl.ServeGetPlaylist))
	rout.Handle("/createPlaylist{_:(?:\\.view)?}", ctrl.H(ctrl.ServeUpdatePlaylist))
	rout.Handle("/updatePlaylist{_:(?:\\.view)?}", ctrl.H(ctrl.ServeUpdatePlaylist))
	rout.Handle("/deletePlaylist{_:(?:\\.view)?}", ctrl.H(ctrl.ServeDeletePlaylist))
	//
	// begin raw
	rout.Handle("/download{_:(?:\\.view)?}", ctrl.HR(ctrl.ServeStream))
	rout.Handle("/stream{_:(?:\\.view)?}", ctrl.HR(ctrl.ServeStream))
	//
	// begin browse by tag
	rout.Handle("/getAlbum{_:(?:\\.view)?}", ctrl.H(ctrl.ServeGetAlbum))
	rout.Handle("/getAlbumList2{_:(?:\\.view)?}", ctrl.H(ctrl.ServeGetAlbumListTwo))
	rout.Handle("/getArtist{_:(?:\\.view)?}", ctrl.H(ctrl.ServeGetArtist))
	rout.Handle("/getArtists{_:(?:\\.view)?}", ctrl.H(ctrl.ServeGetArtists))
	rout.Handle("/search3{_:(?:\\.view)?}", ctrl.H(ctrl.ServeSearchThree))
	//
	// begin browse by folder
	rout.Handle("/getIndexes{_:(?:\\.view)?}", ctrl.H(ctrl.ServeGetIndexes))
	rout.Handle("/getMusicDirectory{_:(?:\\.view)?}", ctrl.H(ctrl.ServeGetMusicDirectory))
	rout.Handle("/getAlbumList{_:(?:\\.view)?}", ctrl.H(ctrl.ServeGetAlbumList))
	rout.Handle("/search2{_:(?:\\.view)?}", ctrl.H(ctrl.ServeSearchTwo))
	return nil
}

func (s *Server) scanTick() {
	ticker := time.NewTicker(s.opts.ScanInterval)
	for range ticker.C {
		if err := s.ctrlBase.Scanner.Start(); err != nil {
			log.Printf("error while scanner: %v", err)
		}
	}
}

func (s *Server) Start() error {
	if s.opts.ScanInterval > 0 {
		log.Printf("will be scanning at intervals of %s", s.opts.ScanInterval)
		go s.scanTick()
	}
	return s.ListenAndServe()
}
