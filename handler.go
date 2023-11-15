package main

import (
	_ "embed"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"strings"
)

//go:embed static/index.html
var indexHTML string

var tpl = template.Must(template.New("index").Parse(indexHTML))

func handler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" && !strings.HasSuffix(r.URL.Path, ".svg") {
		http.NotFound(w, r)
		return
	}

	logf("----------------------")
	logf(" => handling request:  %v", r.URL)
	logf("----------------------")

	// set up cmdline default for analysis
	Analysis.OptsSetup()

	// .. and allow overriding by HTTP params
	Analysis.OverrideByHTTP(r)

	var img string
	if img = Analysis.FindCachedImg(); img != "" {
		log.Println("serving file:", img)
		serveFile(w, r, img)
		return
	}

	// Convert list-style args to []string
	if e := Analysis.ProcessListArgs(); e != nil {
		http.Error(w, "invalid parameters", http.StatusBadRequest)
		return
	}

	output, err := Analysis.Render()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if r.Form.Get("format") == "dot" {
		log.Println("writing dot output..")
		fmt.Fprint(w, string(output))
		return
	}

	log.Printf("converting dot to %s..\n", *outputFormat)

	img, err = dotToImage("", *outputFormat, output)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	err = Analysis.CacheImg(img)
	if err != nil {
		http.Error(w, "cache img error: "+err.Error(), http.StatusBadRequest)
		return
	}

	log.Println("serving file:", img)
	serveFile(w, r, img)
}

type VariablesToRender struct {
	SvgData template.HTML
	// other fields...
}

func serveFile(w http.ResponseWriter, r *http.Request, imgPath string) {
	// read the data in the file `imgPath` and store it in the byte array `data`
	data, err := os.ReadFile(imgPath)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	myvar := VariablesToRender{
		SvgData: template.HTML(data),
		// set other fields...
	}

	outputHTML(w, myvar)
}

func outputHTML(w http.ResponseWriter, data interface{}) {
	if err := tpl.Execute(w, data); err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
}
