package core

import (
	"bytes"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"text/template"

	"github.com/THREATINT/go-crypto"
	TIhttp "github.com/THREATINT/go-http"
	"github.com/blevesearch/bleve"
	"github.com/microcosm-cc/bluemonday"
	"github.com/rs/zerolog"
	"github.com/spf13/afero"
	"github.com/tdewolff/minify/v2"
	"github.com/tdewolff/minify/v2/css"
	"github.com/tdewolff/minify/v2/html"
	"github.com/tdewolff/minify/v2/js"
	"github.com/tdewolff/minify/v2/json"
	"github.com/tdewolff/minify/v2/svg"
	"github.com/tdewolff/minify/v2/xml"
)

var log zerolog.Logger

func NewCore(fs *afero.Fs, logger zerolog.Logger) *Core {

	log = logger

	c := new(Core)

	c.fs = fs

	indexMapping := bleve.NewIndexMapping()
	ftindex, err := bleve.NewMemOnly(indexMapping)
	if err != nil {
		log.Error().Msg(err.Error())
	} else {
		c.ftindex = ftindex
	}

	c.HttpHeaders = &HTTPHeaders{}

	c.PublicFiles = make(map[string]*PublicFile)

	c.Templates = make(map[string]*Template)

	c.minifier = minify.New()
	c.minifier.AddFunc("text/plain", TextMinify.Minify)
	c.minifier.AddFunc("text/css", css.Minify)
	c.minifier.AddFunc("text/html", html.Minify)
	c.minifier.AddFunc("image/svg+xml", svg.Minify)
	c.minifier.AddFuncRegexp(regexp.MustCompile("^(application|text)/(x-)?(java|ecma)script$"), js.Minify)
	c.minifier.AddFuncRegexp(regexp.MustCompile("[/+]json$"), json.Minify)
	c.minifier.AddFuncRegexp(regexp.MustCompile("[/+]xml$"), xml.Minify)

	log.Info().Msg("reading HTTP headers...")
	c.populateHeaders("httpheaders.xml")
	log.Info().Msg(fmt.Sprintf("%d HTTP header(s)", len(c.HttpHeaders.Uri)))

	log.Info().Msg("reading public files...")
	c.populatePublicFiles("public")
	log.Info().Msg(fmt.Sprintf("%d public file(s)", len(c.PublicFiles)))

	log.Info().Msg("reading templates")
	c.populateTemplates("templates")
	log.Info().Msg(fmt.Sprintf("%d template(s)", len(c.Templates)))

	log.Info().Msg("reading nodes...")
	c.populateNodes("nodes")
	log.Info().Msg(fmt.Sprintf("%d node(s)", len(c.Nodes)))

	log.Info().Msg("building search index...")
	c.populateFTIndex()
	dc, err := c.ftindex.DocCount()
	log.Info().Msg(fmt.Sprintf("%d node(s) in index", dc))

	return c
}

type Core struct {
	Nodes       []*Node
	PublicFiles map[string]*PublicFile
	Templates   map[string]*Template
	HttpHeaders *HTTPHeaders
	fs          *afero.Fs
	minifier    *minify.M
	ftindex     bleve.Index
}

func (core *Core) Http(w http.ResponseWriter, r *http.Request) {

	// we do not understand HTTP Range requests -> ignore
	// see https://tools.ietf.org/html/rfc7233#section-1.1
	// - no action needed -

	// Allow only HTTP GET and HEAD
	if strings.ToUpper(r.Method) != "GET" && strings.ToUpper(r.Method) != "HEAD" {
		// neither HTTP GET nor HEAD? -> return 405 ("Method Not Allowed")
		w.WriteHeader(405)
		return
	}

	u, err := url.Parse(strings.ToLower(r.URL.String()))
	if err != nil {
		// error parsing the URL? -> HTTP 400 ("Bad Request")
		w.WriteHeader(400)
		return
	}

	// normalise + sanitise URL
	origurlpath := strings.ToLower(r.URL.String())
	newurlpath, err := url.PathUnescape(origurlpath)
	if err != nil {
		log.Error().Msg(err.Error())
		http.Error(w, "", 500)
		return
	}

	bm := bluemonday.StrictPolicy()

	newurlpath = bm.Sanitize(newurlpath)

	if newurlpath != origurlpath {
		log.Warn().Msg(fmt.Sprintf("Possible XSS: '%s' (sansitised to '%s')", origurlpath, newurlpath))
		http.Redirect(w, r, string(newurlpath), 303)
		return
	}

	// further normalisation:
	// if suffix '/' is present, redirect to url without suffix
	// to avoid "duplicate content" problem with search engines
	urlpath := strings.TrimSuffix(u.Path, "/")
	if u.Path != urlpath && urlpath != "" {
		http.Redirect(w, r, string(urlpath), 303)
		return
	}

	// remove leading slash ("/")
	urlpath = strings.TrimPrefix(urlpath, "/")

	var content []byte

	// we start by searching the static content:
	f := core.PublicFiles[urlpath]
	if f != nil {
		etag := crypto.SHA256(string(f.Content[:]))

		// send ETag, no matter if 200 or 304 (see https://tools.ietf.org/html/rfc7232#section-4.1)
		w.Header().Set("Etag", etag)

		inm := r.Header.Get("If-None-Match")
		if inm != "" && strings.Contains(inm, etag) {
			// no change here
			w.WriteHeader(304)
			return
		}

		w.Header().Set("Content-Type", f.MimeType)

		content = f.Content
	} else {
		// we have not found matching static content, so we start searching our nodes list:
		node := FindNode(urlpath, core.Nodes)
		if node == nil {
			node = FindApplicationEndpointNode(urlpath, core.Nodes)
			if node == nil {
				node = FindFallbackNode(urlpath, core.Nodes)

				if node != nil {
					http.Redirect(w, r, string(node.Path()), 303)
					return
				}

				var acceptLang = TIhttp.ParseAcceptLanguage(r.Header.Get("Accept-Language"))

				// We have not found a matching node yet.
				//
				// Fallback is to get the best match (based on language) from the root nodes
				for _, l := range acceptLang {
					for _, n := range RootNodes(core.Nodes) {
						if n.Language() == l.Lang && n.Enabled() {
							http.Redirect(w, r, string(n.Path()), 303)
							return
						}
					}
				}

				// Since we have reached this line, we still have not found a matching language.
				//
				// Maybe the user does only accept e.g. "en-US", but our site is configured to use "en",
				// let's try to ignore the country part of the locale requested:
				for _, l := range acceptLang {
					for _, n := range RootNodes(core.Nodes) {
						if n.Language() == strings.Split(l.Lang, "-")[0] && n.Enabled() {
							http.Redirect(w, r, string(n.Path()), 303)
							return
						}
					}
				}

				// we tried almost everything ... last resort:
				// we redirect to the first node that is available (aka: enabled)
				for _, n := range RootNodes(core.Nodes) {
					if n.Enabled() {
						http.Redirect(w, r, string(n.Path()), 303)
						return
					}
				}

				// you guessed it: we give up!
				// Nothing to be found here!
				// We are done.
				http.Error(w, "not found", 404)
				return
			}
		}

		if node.RedirectTo() != "" {
			http.Redirect(w, r, strings.TrimSpace(string(node.RedirectTo())), 303)
			return
		}

		context := Context{
			HttpRequest:   r,
			Node:          node,
			Content:       node.Render(),
			AllNodes:      core.Nodes,
			PublicFiles:   core.PublicFiles,
			FulltextIndex: core.ftindex,
		}

		var lr bytes.Buffer
		lr.WriteString(r.RemoteAddr)
		lr.WriteString(" ")
		lr.WriteString(r.Method)
		lr.WriteString(" ")
		lr.WriteString(r.Host)
		lr.WriteString(r.RequestURI)
		lr.WriteString(" ")
		lr.WriteString(r.URL.Port())
		lr.WriteString(" - ")

		t := core.Templates[node.Template()]
		if t == nil {
			log.Error().Msg(lr.String())
			return
		}

		for {
			var buf bytes.Buffer
			gt := template.New(t.Name())
			gt, err = gt.Parse(t.Content())

			if err != nil {
				log.Error().Msg(fmt.Sprintf("%s: %s", lr.String(), err.Error()))
				w.WriteHeader(500)
				return
			}

			err = gt.Execute(&buf, &context)
			if err != nil {
				log.Error().Msg(fmt.Sprintf("%s: %s", lr.String(), err.Error()))
				w.WriteHeader(500)
				return
			}

			context.Content = buf.String()
			if t.Parent() == "" {
				break
			}

			t = core.Templates[t.Parent()]
		}

		// Minify the content
		page, err := core.minifier.String(t.MimeType(), string(context.Content))
		if err != nil {
			// If it goes wrong for any reason, we leave the original content untouched and continue
			page = string(context.Content)
			log.Warn().Msg(err.Error())
		}

		etag := crypto.RIPEMD160(page)

		// send ETag, no matter if 200 or 304 (see https://tools.ietf.org/html/rfc7232#section-4.1)
		w.Header().Set("Etag", etag)

		// Etag in request matches our Etag? -> content has not chanced
		inm := r.Header.Get("If-None-Match")
		if inm != "" && strings.Contains(inm, etag) {
			w.WriteHeader(304)
			return
		}

		w.Header().Set("Content-Type", t.MimeType()+"; charset=UTF-8")

		if strings.ToUpper(r.Method) != "HEAD" {
			// don't send body if HTTP Method is HEAD
			content = []byte(page)
		}
	}

	// set HTTP headers based on URI
	for _, h := range core.HttpHeaders.Match(urlpath) {
		r := strings.SplitN(h, ":", 2)
		w.Header().Add(strings.TrimSpace(r[0]), strings.TrimSpace(r[1]))
	}

	// write content to response
	if _, err = w.Write(content); err != nil {
		w.WriteHeader(500)
	}
}

func (core *Core) populateHeaders(filename string) {
	var s bytes.Buffer

	file, err := afero.ReadFile(*core.fs, filename)
	if err != nil {
		s.WriteString(" - ")
		s.WriteString(err.Error())
		log.Error().Msg(s.String())

		return
	}

	err = core.HttpHeaders.Read(file)
	if err != nil {
		s.WriteString(" - ")
		s.WriteString(err.Error())
		log.Error().Msg(s.String())
	}

	for _, uri := range core.HttpHeaders.Uri {
		uri.Expression = strings.ToLower(uri.Expression)
	}
}

func (core *Core) populatePublicFiles(dir string) {
	var s bytes.Buffer

	afero.Walk(*core.fs, dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			log.Error().Msg(err.Error())
			return nil
		} else {
			path = filepath.Clean(path)
			p := strings.TrimPrefix(path, dir)
			p = strings.TrimPrefix(p, "/")
			p = strings.ToLower(p)

			if !info.IsDir() {
				s.Reset()
				s.WriteString("--")
				s.WriteString(path)

				file, err := afero.ReadFile(*core.fs, path)
				if err != nil {
					s.WriteString(" - ")
					s.WriteString(err.Error())
					log.Error().Msg(s.String())
					return nil
				}

				core.PublicFiles[p] = &PublicFile{
					Content:  file,
					MimeType: TIhttp.MimeTypeByExtension(filepath.Ext(path)),
				}

				//log.Debug().Msg(fmt.Sprintf("reading file %s (%s)", path, core.PublicFiles[p].MimeType))
			}

			return nil
		}
	})
}

func (core *Core) populateTemplates(dir string) {
	var s bytes.Buffer

	afero.Walk(*core.fs, dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			log.Error().Msg(err.Error())
			return nil
		} else {
			path = filepath.Clean(path)
			p := strings.TrimPrefix(path, dir)
			p = strings.TrimPrefix(p, "/")
			p = strings.ToLower(p)

			if strings.HasSuffix(p, ".xml") {
				p = strings.TrimSuffix(p, ".xml")

				if !info.IsDir() {
					s.Reset()
					s.WriteString("--")
					s.WriteString(path)

					file, err := afero.ReadFile(*core.fs, path)
					if err != nil {
						s.WriteString(" - ")
						s.WriteString(err.Error())
						log.Error().Msg(s.String())
						return nil
					}

					var templ Template
					err = templ.Read(file)
					if err != nil {
						s.WriteString(" - ")
						s.WriteString(err.Error())
						log.Error().Msg(s.String())
						return nil
					}
					templ.name = p

					if templ.ContentFile() != "" {
						file, err = afero.ReadFile(*core.fs, filepath.Join(filepath.Dir(path), templ.ContentFile()))
						if err != nil {
							s.WriteString(" - ")
							s.WriteString(err.Error())
							log.Error().Msg(s.String())
							return nil
						} else {
							s.WriteString(" (using ")
							s.WriteString(templ.ContentFile())
							s.WriteString(") ")

							templ.SetContent(string(file))
						}
					}

					gt := template.New(p)
					gt, err = gt.Parse(templ.Content())
					if err != nil {
						s.WriteString(" - ")
						s.WriteString(err.Error())
						log.Error().Msg(s.String())
						return nil
					}

					//log.Debug().Msg(fmt.Sprintf("reading template %s", templ.Name))

					core.Templates[p] = &templ
				}
			}

			return nil
		}
	})
}

func (core *Core) populateNodes(dir string) {
	log.Info().Msg("reading Nodes")
	core.nodesFromDir(dir)
}

func (core *Core) nodesFromDir(dir string) []*Node {
	var nodes []*Node
	var s bytes.Buffer

	fis, err := afero.ReadDir(*core.fs, dir)
	if err != nil {
		s.Reset()
		s.WriteString(dir)
		s.WriteString("--")
		s.WriteString(err.Error())
		log.Error().Msg(s.String())
	} else {
		for _, fi := range fis {
			if !fi.IsDir() {
				p := path.Join(dir, fi.Name())

				s.Reset()
				s.WriteString("--")
				s.WriteString(p)

				file, err := afero.ReadFile(*core.fs, p)
				if err != nil {
					s.WriteString(" - ")
					s.WriteString(err.Error())
					log.Error().Msg(s.String())
				} else {
					var node Node

					p := strings.TrimPrefix(p, dir)
					p = strings.TrimPrefix(p, "/")
					p = strings.TrimSuffix(p, ".xml")

					err = node.Read(file, p)
					if err != nil {
						s.WriteString(" - ")
						s.WriteString(err.Error())
						log.Error().Msg(s.String())
						return nil
					}

					//log.Debug().Msg(fmt.Sprintf("reading node %s", node.Path()))

					nodes = append(nodes, &node)
					core.Nodes = append(core.Nodes, &node)
					sort.Sort(NodeSorter(nodes))
				}
			}
		}
	}

	for _, node := range nodes {
		p := path.Join(dir, node.Name())
		isDir, err := afero.DirExists(*core.fs, p)
		if err == nil && isDir {
			node.children = core.nodesFromDir(p)
			for _, n := range node.children {
				n.SetParent(node)
			}
		}
	}

	return nodes
}

func (core *Core) populateFTIndex() {
	log.Info().Msg("indexing Nodes ... ")

	for _, node := range core.Nodes {
		if node.Enabled() {
			ns := NewNodeSearchable(node)

			//log.Debug().Msg(fmt.Sprintf("indexing node %s", node.Path()))

			core.ftindex.Index(string(node.Path()), ns)
		}
	}
}
