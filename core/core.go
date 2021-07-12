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

// NewCore Initialiser for new onacms core engine
func NewCore(fs *afero.Fs, logger zerolog.Logger) *Core {
	var (
		err     error
		ftindex bleve.Index
	)

	log = logger

	c := new(Core)

	c.fs = fs

	indexMapping := bleve.NewIndexMapping()
	if ftindex, err = bleve.NewMemOnly(indexMapping); err != nil {
		log.Error().Msg(err.Error())
	} else {
		c.ftindex = ftindex
	}

	c.HTTPHeaders = &HTTPHeaders{}

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
	c.populateHeaders("http-headers.xml")
	log.Info().Msg(fmt.Sprintf("%d HTTP header(s)", len(c.HTTPHeaders.URI)))

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
	if dc, err := c.ftindex.DocCount(); err == nil {
		log.Info().Msg(fmt.Sprintf("%d node(s) in index", dc))
	} else {
		log.Error().Msg(err.Error())
	}

	return c
}

// Core struct for onacms core engine
type Core struct {
	Nodes       []*Node
	PublicFiles map[string]*PublicFile
	Templates   map[string]*Template
	HTTPHeaders *HTTPHeaders
	fs          *afero.Fs
	minifier    *minify.M
	ftindex     bleve.Index
}

// HTTP ...
func (core *Core) HTTP(w http.ResponseWriter, r *http.Request) {
	var (
		err error

		urlpath    = strings.ToLower(r.URL.String())
		newurlpath string

		content []byte
	)

	// we do not understand HTTP Range requests -> ignore
	// see https://tools.ietf.org/html/rfc7233#section-1.1
	// - no action needed -

	// Allow only HTTP GET and HEAD
	if strings.ToUpper(r.Method) != "GET" && strings.ToUpper(r.Method) != "HEAD" {
		// neither HTTP GET nor HEAD? -> return 405 ("Method Not Allowed")
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	if _, err = url.Parse(urlpath); err != nil {
		// error parsing the URL? -> HTTP 400 ("Bad Request")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// normalise + sanitise URL
	if newurlpath, err = url.PathUnescape(urlpath); err != nil {
		log.Error().Msg(err.Error())
		http.Error(w, "", http.StatusInternalServerError)
		return
	}

	newurlpath = bluemonday.StrictPolicy().Sanitize(newurlpath)

	// some browsers allow \ or %5c in URLs. BlueMonday encodes %5c back to \, but:
	// for we a using relative paths for forwards, a request with path /\www.evil-host.com
	// would redirect to www.evil-host.com because some browsers treat \ as /, so /\ becomes //
	// which is treated as an absolute path!
	// quick fix: replace all /\ with /
	newurlpath = strings.ReplaceAll(newurlpath, "/\\", "/")

	// if suffix '/' is present, remove to avoid "duplicate content" problem with search engines
	for strings.HasSuffix(newurlpath, "/") && len(newurlpath) > 1 {
		newurlpath = strings.TrimSuffix(newurlpath, "/")
	}

	// path is different from original path after cleanup + sanitising -> redirect to new (clean) path
	if newurlpath != urlpath {
		log.Warn().Msg(fmt.Sprintf("'%s', sanitised to '%s'", urlpath, newurlpath))
		http.Redirect(w, r, newurlpath, http.StatusSeeOther)
		return
	}

	// remove leading slash ("/")
	urlpath = strings.TrimPrefix(urlpath, "/")

	// we start by searching the static content:
	if f := core.PublicFiles[urlpath]; f != nil {
		etag := crypto.RIPEMD160(string(f.Content[:]))

		// send ETag, no matter if 200 or 304 (see https://tools.ietf.org/html/rfc7232#section-4.1)
		w.Header().Set("Etag", etag)

		inm := r.Header.Get("If-None-Match")
		if inm != "" && strings.Contains(inm, etag) {
			// no change here
			w.WriteHeader(http.StatusNotModified)
			return
		}

		w.Header().Set("Content-Type", f.MimeType)
		content = f.Content
	} else {
		var (
			node *Node
		)

		// we have not found matching static content, so we start searching our nodes list:
		if node = FindNode(urlpath, core.Nodes); node == nil {
			if node = FindApplicationEndpointNode(urlpath, core.Nodes); node == nil {
				if node = FindFallbackNode(urlpath, core.Nodes); node != nil {
					http.Redirect(w, r, string(node.Path()), http.StatusSeeOther)
					return
				}

				var acceptLang = TIhttp.ParseAcceptLanguage(r.Header.Get("Accept-Language"))

				// We have not found a matching node yet.
				//
				// Fallback is to get the best match (based on language) from the root nodes
				for _, l := range acceptLang {
					for _, n := range RootNodes(core.Nodes) {
						if n.Language() == l.Lang && n.Enabled() {
							http.Redirect(w, r, string(n.Path()), http.StatusSeeOther)
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
							http.Redirect(w, r, string(n.Path()), http.StatusSeeOther)
							return
						}
					}
				}

				// we tried almost everything ... last resort:
				// we redirect to the first node that is available (aka: enabled)
				for _, n := range RootNodes(core.Nodes) {
					if n.Enabled() {
						http.Redirect(w, r, string(n.Path()), http.StatusSeeOther)
						return
					}
				}

				// you guessed it: we give up!
				// Nothing to be found here!
				// We are done.
				http.Error(w, "not found", http.StatusNotFound)
				return
			}
		}

		if node.RedirectTo() != "" {
			http.Redirect(w, r, strings.TrimSpace(string(node.RedirectTo())), http.StatusSeeOther)
			return
		}

		context := Context{
			HTTPRequest:   r,
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
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			err = gt.Execute(&buf, &context)
			if err != nil {
				log.Error().Msg(fmt.Sprintf("%s: %s", lr.String(), err.Error()))
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			context.Content = buf.String()

			if t.Parent() == "" {
				break
			}

			t = core.Templates[t.Parent()]
		}

		// Minify the content
		var (
			page string
		)
		if page, err = core.minifier.String(t.MimeType(), string(context.Content)); err != nil {
			// If minifying goes wrong for any reason, we leave the original content untouched and continue
			page = string(context.Content)
			log.Warn().Msg(err.Error())
		}

		etag := crypto.RIPEMD160(page)

		// send ETag, no matter if 200 or 304 (see https://tools.ietf.org/html/rfc7232#section-4.1)
		w.Header().Set("Etag", etag)

		// Etag in request matches our Etag? -> content has not chanced
		inm := r.Header.Get("If-None-Match")
		if inm != "" && strings.Contains(inm, etag) {
			w.WriteHeader(http.StatusNotModified)
			return
		}

		w.Header().Set("Content-Type", t.MimeType()+"; charset=UTF-8")

		// only send body if HTTP Method is GET, HEAD does not expect a body
		if strings.ToUpper(r.Method) == "GET" {
			content = []byte(page)
		}
	}

	// set HTTP headers based on URI
	for _, h := range core.HTTPHeaders.Match(urlpath) {
		r := strings.SplitN(h, ":", 2)
		w.Header().Add(strings.TrimSpace(r[0]), strings.TrimSpace(r[1]))
	}

	// write content to response
	if _, err = w.Write(content); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
	}
}

func (core *Core) populateHeaders(filename string) {
	var (
		err  error
		s    bytes.Buffer
		file []byte
	)

	if file, err = afero.ReadFile(*core.fs, filename); err != nil {
		s.WriteString(" - ")
		s.WriteString(err.Error())
		log.Warn().Msg(s.String())

		return
	}

	if err = core.HTTPHeaders.Read(file); err != nil {
		s.WriteString(" - ")
		s.WriteString(err.Error())
		log.Warn().Msg(s.String())
	}

	for _, uri := range core.HTTPHeaders.URI {
		uri.Expression = strings.ToLower(uri.Expression)
	}
}

func (core *Core) populatePublicFiles(dir string) {
	var (
		s    bytes.Buffer
		file []byte
	)

	afero.Walk(*core.fs, dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			log.Error().Msg(err.Error())
			return nil
		}

		path = filepath.Clean(path)
		p := strings.TrimPrefix(path, dir)
		p = strings.TrimPrefix(p, "/")
		p = strings.ToLower(p)

		if !info.IsDir() {
			s.Reset()
			s.WriteString("--")
			s.WriteString(path)

			if file, err = afero.ReadFile(*core.fs, path); err != nil {
				s.WriteString(" - ")
				s.WriteString(err.Error())
				log.Error().Msg(s.String())
				return nil
			}

			core.PublicFiles[p] = &PublicFile{
				Content:  file,
				MimeType: TIhttp.MimeTypeByExtension(filepath.Ext(path)),
			}
		}

		return nil
	})
}

func (core *Core) populateTemplates(dir string) {
	var (
		s    bytes.Buffer
		file []byte
	)

	afero.Walk(*core.fs, dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			log.Error().Msg(err.Error())
			return nil
		}

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

				if file, err = afero.ReadFile(*core.fs, path); err != nil {
					s.WriteString(" - ")
					s.WriteString(err.Error())
					log.Error().Msg(s.String())
					return nil
				}

				var templ Template
				if err = templ.Read(file); err != nil {
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
					}

					s.WriteString(" (using ")
					s.WriteString(templ.ContentFile())
					s.WriteString(") ")

					templ.SetContent(string(file))
				}

				gt := template.New(p)
				if _, err = gt.Parse(templ.Content()); err != nil {
					s.WriteString(" - ")
					s.WriteString(err.Error())
					log.Error().Msg(s.String())
					return nil
				}

				core.Templates[p] = &templ
			}
		}

		return nil
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

		return nodes
	}

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

				nodes = append(nodes, &node)
				core.Nodes = append(core.Nodes, &node)
				sort.Sort(NodeSorter(nodes))
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
