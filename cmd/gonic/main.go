package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	_ "github.com/jinzhu/gorm/dialects/sqlite"
	"github.com/peterbourgon/ff"

	"github.com/sentriz/gonic/db"
	"github.com/sentriz/gonic/server"
	"github.com/sentriz/gonic/version"
)

const (
	programName = "gonic"
	programVar  = "GONIC"
)

func main() {
	set := flag.NewFlagSet(programName, flag.ExitOnError)
	listenAddr := set.String("listen-addr", "0.0.0.0:4747", "listen address (optional)")
	musicPath := set.String("music-path", "", "path to music")
	dbPath := set.String("db-path", "gonic.db", "path to database (optional)")
	scanInterval := set.Int("scan-interval", 0, "interval (in minutes) to automatically scan music (optional)")
	_ = set.String("config-path", "", "path to config (optional)")
	showVersion := set.Bool("version", false, "show gonic version")
	if err := ff.Parse(set, os.Args[1:],
		ff.WithConfigFileFlag("config-path"),
		ff.WithConfigFileParser(ff.PlainParser),
		ff.WithEnvVarPrefix(programVar),
	); err != nil {
		log.Fatalf("error parsing args: %v\n", err)
	}
	if *showVersion {
		fmt.Println(version.VERSION)
		os.Exit(0)
	}
	if _, err := os.Stat(*musicPath); os.IsNotExist(err) {
		log.Fatal("please provide a valid music directory")
	}
	db, err := db.New(*dbPath)
	if err != nil {
		log.Fatalf("error opening database: %v\n", err)
	}
	defer db.Close()
	serverOptions := server.ServerOptions{
		DB:           db,
		MusicPath:    *musicPath,
		ListenAddr:   *listenAddr,
		ScanInterval: time.Duration(*scanInterval) * time.Minute,
	}
	log.Printf("using opts %+v\n", serverOptions)
	s := server.New(serverOptions)
	if err = s.SetupAdmin(); err != nil {
		log.Fatalf("error setting up admin routes: %v\n", err)
	}
	if err = s.SetupSubsonic(); err != nil {
		log.Fatalf("error setting up subsonic routes: %v\n", err)
	}
	log.Printf("starting server at %s", *listenAddr)
	if err := s.Start(); err != nil {
		log.Fatalf("error starting server: %v\n", err)
	}
}
