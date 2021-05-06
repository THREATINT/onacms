package main

import (
	"bytes"
	"fmt"
	"net/http"
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"text/template"
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

		staticOutputDir = kingpin.Arg("Output", "do not start webserver, instead output site to <Output>").String()
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

	c := core.NewCore(&fs, log)
	if len(c.Nodes) == 0 {
		log.Fatal().Msg("no nodes, exiting...")
		os.Exit(0xe0)
	}

	if *staticOutputDir != "" {
		for _, node := range c.Nodes {
			if node.Enabled() {
				p := filepath.Join(*staticOutputDir, string(node.Path()))

				m := fmt.Sprintf("%s...", p)

				d := filepath.Dir(p)

				if d != "." && d != ".." && d != string(os.PathSeparator) {
					if err := os.MkdirAll(d, os.ModePerm); err != nil {
						log.Error().Msg(fmt.Sprintf("%s Mkdir: %s", m, err.Error()))
						os.Exit(0xf2)
					}
				}

				f, err := os.OpenFile(p, os.O_CREATE|os.O_WRONLY, os.ModePerm)
				if err != nil {
					log.Error().Msg(fmt.Sprintf("%s%s", m, err.Error()))
					os.Exit(0xf1)
				}
				defer f.Close()

				t := c.Templates[node.Template()]
				if t == nil {
					log.Warn().Msg(fmt.Sprintf("%s%s", m, err.Error()))
				}

				context := core.Context{
					Content: node.Render(),
					Node:    node,
				}

				for {
					var buf bytes.Buffer
					gt := template.New(t.Name())
					gt, err = gt.Parse(t.Content())
					if err != nil {
						log.Error().Msg(fmt.Sprintf("%s%s", m, err.Error()))
						break
					}

					err = gt.Execute(&buf, &context)
					if err != nil {
						log.Error().Msg(fmt.Sprintf("%s%s", m, err.Error()))
						break
					}

					context.Content = buf.String()

					if t.Parent() == "" {
						break
					}
					t = c.Templates[t.Parent()]
				}

				f.WriteString(context.Content)

				log.Info().Msg(fmt.Sprintf("%s ok", m))
			}
		}

		os.Exit(0)
	}

	r := chi.NewRouter()

	r.Use(middleware.Timeout(time.Second * 10))

	r.Use(middleware.Compress(9))

	r.Use(helpers.Recoverer(&log))

	r.Get("/*", c.HTTP)

	log.Info().Msg(fmt.Sprintf("Running on port %v.", *port))
	server := &http.Server{Addr: fmt.Sprintf(":%v", *port), Handler: http.TimeoutHandler(r, 4*time.Second, ""), ReadTimeout: time.Second * 2, WriteTimeout: time.Second * 4}

	server.ListenAndServe()
}
