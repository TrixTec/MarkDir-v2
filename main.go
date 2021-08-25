package main

import (
	"errors"
	"flag"
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/russross/blackfriday"
)

var (
	bind = flag.String("bind", "127.0.0.1:19000", "port to run the server on")
	path = flag.String("path", ".", "path to list")
)

func main() {
	flag.Parse()

	httpdir := http.Dir(*path)
	handler := renderer{httpdir, http.FileServer(httpdir)}

	log.Println("Serving on http://" + *bind)
	log.Fatal(http.ListenAndServe(*bind, handler))
}

var outputTemplate = template.Must(template.New("base").Parse(`
<html>
  <head>
    <title>{{ .Path }}</title>
  </head>
  <body>
    {{ .Body }}
  </body>
</html>
`))

type renderer struct {
	d http.Dir
	h http.Handler
}

func (r renderer) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	if !strings.HasSuffix(req.URL.Path, ".md") {
		dir, _ := os.ReadDir(*path)
		mdLinkList := []string{}

		for _, x := range dir {
			if strings.HasSuffix(x.Name(), ".md") {
				fmt.Println(x.Name())
				l := "<a href=\"" + x.Name() + "\">" + x.Name() + "</a><br>"
				mdLinkList = append(mdLinkList, l)
			}
			//r.h.ServeHTTP(rw, req)
		}
		fmt.Fprint(rw, strings.Join(mdLinkList, "\n"))
		return
	}

	// net/http is already running a path.Clean on the req.URL.Path,
	// so this is not a directory traversal, at least by my testing
	var pathErr *os.PathError
	input, err := ioutil.ReadFile(*path + req.URL.Path)
	if errors.As(err, &pathErr) {
		http.Error(rw, http.StatusText(http.StatusNotFound)+": "+req.URL.Path, http.StatusNotFound)
		log.Printf("file not found: %s", err)
		return
	}

	if err != nil {
		http.Error(rw, "Internal Server Error: "+err.Error(), 500)
		log.Printf("Couldn't read path %s: %v (%T)", req.URL.Path, err, err)
		return
	}

	output := blackfriday.MarkdownCommon(input)

	rw.Header().Set("Content-Type", "text/html")

	outputTemplate.Execute(rw, struct {
		Path string
		Body template.HTML
	}{
		Path: req.URL.Path,
		Body: template.HTML(string(output)),
	})

}
