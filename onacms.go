package main

import (
	"compress/gzip"
	"fmt"
	"net/http"
	"os"
	"os/user"
	"strings"
	"time"

	"github.com/NYTimes/gziphandler"
	"github.com/THREATINT/onacms/core"
	"github.com/rs/zerolog"
	"github.com/spf13/afero"
	"gopkg.in/alecthomas/kingpin.v2"
)

func main() {
	var (
		dir  = kingpin.Flag("dir", "directory containing the site (/public /nodes /templates)").Default("/www").String()
		port = kingpin.Flag("port", "(optional) TCP port").Default("10000").Int16()
	)

	kingpin.Parse()

	output := zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339}
	output.FormatLevel = func(i interface{}) string {
		return strings.ToUpper(fmt.Sprintf("| %-6s|", i))
	}
	log := zerolog.New(output).With().Timestamp().Logger()

	log.Info().Msg("onacms (C) THREATINT")

	u, err := user.Current()
	if err == nil && u.Username == "root" {
		log.Warn().Msg("please do not run as root!")
	}

	fs := afero.NewBasePathFs(afero.NewOsFs(), *dir)

	core := core.NewCore(&fs, log)

	if len(core.Nodes) == 0 {
		log.Fatal().Msg("no nodes, exiting...")
		os.Exit(0xe0)
	}

	log.Info().Msg(fmt.Sprintf("starting onacms port %d", *port))

	defer func() {
		if r := recover(); r != nil {
			log.Fatal().Msg(fmt.Sprintf("%+v - exiting", r))
			os.Exit(0xfe)
		}
	}()

	server := http.NewServeMux()

	gzh, err := gziphandler.GzipHandlerWithOpts(
		gziphandler.MinSize(100),
		gziphandler.CompressionLevel(gzip.BestCompression))

	server.Handle("/", gzh(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		core.Http(w, r)
	})))

	err = http.ListenAndServe(fmt.Sprintf(":%d", *port), server)
	if err != nil {
		log.Fatal().Msg(fmt.Sprintf("%s - exiting", err.Error()))
		os.Exit(0xfc)
	}
}
