# onacms

Onacms (aka: "Oh No! Another Content Management System!") is not a full featured content management system, but a content management engine written in Go (Golang). It is considered to be *really* fast for it deliveres all content from memory.

Although Onacms knows about outputing content, minification, and ETags, it heavily relies on a webserver like NGINX or Apache Httpd as a frontend for most other stuff, e.g. logging or TLS (transport layer security)


## Getting started

Onacms makes use of the following three concepts for a site:

- Nodes (aka: pages) (/nodes):

  This is where your content goes. Content can be plain HTML or Markdown.

- Templates (/templates):

  Templates take the content from nodes and generate the actual output, e.g. HTML pages for a website, sitemap.xml, etc. Templates can be written in the builtin Golang HTML templating engine.

- Static/public files (/public):

  These files are handled by onacms in the same way that you would expect from any other webserver. Use it e.g. for  static files like robots.txt.


## Building and dependencies

You can either run
```
go build
```
for development or
```
make
```
for a production build that requires UNIX ```make``` and
[UPX](https://upx.github.io/)
to be installed installed your local machine.


## Running

```
onacms [--dir=<directory>] [--port=<TCP port>]
```
_directory_ is the directory containing the actual site (/nodes /templates /public).

_TCP port_ is the TCP port the daemon listens on. It defaults to 10000.

Onacms does not log interactions with clients! Please use the frontend webserver to have information like Client IP address, bytes transferred, etc. logged.


### Docker

We provide a Dockerfile to use onacms in a Docker container. Please see [hub.docker.com/r/threatint/onacms](https://hub.docker.com/r/threatint/onacms):
```
docker pull threatint/onacms
```

To start the container and map a local directory read-only to /data and a local port to 10000/TCP, e.g.:
```
docker run --name=mysite -p 10000:10000 -v /home/user/dir/:/data:ro threatint/onacms
```


## License

Released under the [GNU Affero General Public License](http://www.gnu.org/licenses/agpl.HTML).

Kindly contact [Team@THREATINT.com](mailto:team@threatint.com) for information on licensing for OEM / ISV.