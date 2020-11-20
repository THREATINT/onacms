package main

import (
	"fmt"
	"net/http"
	"os"
	"os/user"
	"strings"
	"time"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/rs/zerolog"
	"github.com/spf13/afero"

	"gopkg.in/alecthomas/kingpin.v2"

	"github.com/THREATINT/onacms/core"
	"github.com/THREATINT/onacms/helpers"
)

func main() {
	var (
		dir  = kingpin.Flag("dir", "directory containing the site (/public /nodes /templates)").Default("/www").String()
		port = kingpin.Flag("port", "(optional) TCP port").Default("10000").Int16()

		logtimestamps = kingpin.Flag("log-timestamps", "include timestamps in logging , not required e.g. when using syslog)").Bool()
	)

	kingpin.Parse()

	output := zerolog.ConsoleWriter{Out: os.Stdout}
	if *logtimestamps {
		output = zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339}
	} else {
		output = zerolog.ConsoleWriter{Out: os.Stdout}
		output.FormatTimestamp = func(i interface{}) string {
			return ""
		}
	}
	output.FormatLevel = func(i interface{}) string {
		return strings.ToUpper(fmt.Sprintf("%-6s|", i))
	}

	log := zerolog.New(output).With().Timestamp().Logger()

	log.Info().Msg("onacms (C) THREATINT")

	u, err := user.Current()
	if err == nil && u.Username == "root" {
		log.Warn().Msg("please do not run as root")
	}

	fs := afero.NewBasePathFs(afero.NewOsFs(), *dir)

	core := core.NewCore(&fs, log)
	if len(core.Nodes) == 0 {
		log.Fatal().Msg("no nodes, exiting...")
		os.Exit(0xe0)
	}

	r := chi.NewRouter()

	r.Use(middleware.Timeout(time.Minute))
	r.Use(middleware.StripSlashes)
	r.Use(middleware.Compress(9, "gzip"))

	r.Use(helpers.Recoverer(&log))

	r.Get("/*", core.HTTP)

	log.Info().Msg(fmt.Sprintf("Running on port %v.", *port))
	err = http.ListenAndServe(fmt.Sprintf(":%v", *port), r)
	if err != nil {
		log.Fatal().Msg(fmt.Sprintf("%s - exiting", err.Error()))
		os.Exit(0xfc)
	}
}
