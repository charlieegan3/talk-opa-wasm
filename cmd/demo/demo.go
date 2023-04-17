package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"embed"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/alecthomas/chroma/quick"
	"github.com/gorilla/handlers"
	"github.com/open-policy-agent/opa/ast"
	"github.com/open-policy-agent/opa/bundle"
	"github.com/open-policy-agent/opa/compile"

	"github.com/gorilla/mux"
)

type policyNote struct {
	Title string
	Body  string
	HTML  template.HTML
}

func (p *policyNote) Render() {
	var err error

	var notesFormatted bytes.Buffer
	err = quick.Highlight(&notesFormatted, p.Body, "ruby", "html", "monokailight")
	if err != nil {
		p.HTML = template.HTML(p.Body)
		return
	}

	p.HTML = template.HTML(notesFormatted.String())
}

//go:embed templates/*
var templates embed.FS

const entrypoint = "application/form/deny"

var policyData = map[string]string{
	"application/form": `
package application.form

import future.keywords.in

`,
}

// https://play.openpolicyagent.org/p/DmpSU2s3VO
var policyNotes = map[string][]policyNote{
	"application/form": {
		{
			Title: "Validate Email",
			Body: ` 
# validate email
email_pattern := ` + "`" + `^\S+@\S+\.\S+$` + "`" + `
deny[reason] {
	not regex.match(email_pattern, input.email)
	reason := {
		"field": "email",	
		"message": sprintf("Email must match pattern %v.", [email_pattern]),
	}
}
deny[reason] {
	endswith(input.email, "@example.com")
	reason := {
		"field": "email",	
		"message": "@example.com emails not permitted.",
	}
}
`,
		},
		{
			Title: "Validate Route",
			Body: ` 

# validate route
deny[reason] {
	input.departure_station == input.destination_station 
	reason := {
		"field": "route",
		"message": "Departure and destination stations must be different."
	}
}
deny[reason] {
	"none" in { input.departure_station, input.destination_station } 
	reason := {
		"field": "route",
		"message": "Departure and destination stations must be set."
	}
}
station_areas := {
	"amsterdam": "europe",
	"paris": "europe",
	"osaka": "japan",
	"tokyo": "japan",
}
deny[reason] {
	station_areas[input.departure_station] != station_areas[input.destination_station]
	reason := {
		"field": "route",
		"message": sprintf("Route must be within the same area. %s and %s are in different areas.", [input.departure_station, input.destination_station]),
	}
}
`,
		},
		{
			Title: "Validate Passenger Count",
			Body: ` 

# validate passenger_count
deny[reason] {
	input.passenger_count < 1
	reason := {
		"field": "passenger_count",
		"message": "Must have at least one passenger.",
	}
}
`,
		},
		{
			Title: "Validate Seat Selection",
			Body: ` 

# validate seat
deny[reason] {
	count(input.seats) != input.passenger_count
	reason := {
		"field": "seat",
		"message": sprintf("Selected number of seats %d does not match passenger count %d.", [count(input.seats), input.passenger_count]),
	}
}
seat_adjacencies := {
	"seat1": ["seat2", "seat3", "seat4"],
	"seat2": ["seat1", "seat3", "seat4"],
	"seat3": ["seat1", "seat2", "seat4"],
	"seat4": ["seat1", "seat2", "seat3"],

	"seat5": ["seat6"],
	"seat6": ["seat5"],

	"seat7": ["seat8", "seat9", "seat10"],
	"seat8": ["seat7", "seat9", "seat10"],
	"seat9": ["seat7", "seat8", "seat10"],
	"seat10": ["seat7", "seat8", "seat9"],

	"seat11": ["seat12"],
	"seat12": ["seat11"],	
}
deny[reason] {
	count(input.seats) > 1

	# parsed JSON is an array rather than a set
	seats_set := {s | s := input.seats[_]}

	some i
	seat := seats_set[i]

	# validate that one of the other selected seats is adjacent
	# find the set of reachable seats from the current seat
	reachable := graph.reachable(seat_adjacencies, {seat}) - {seat}
	# check that the set of reachable seats is a subset of the set of selected seats
	reachable & seats_set == set()

	reason := {
		"field": "seat",
		"message": sprintf("Seat %s is not adjacent to other selected seats.", [seat]),
	}
}
`,
		},
	},
}

func main() {
	r := mux.NewRouter()
	r.Handle("/config", http.HandlerFunc(configIndexHandler))
	r.Handle("/config/{site}/{bundle}", http.HandlerFunc(configShowHandler))
	r.Handle("/bundles/{site}/{bundle}.wasm", http.HandlerFunc(bundleHandler))

	addr := "localhost:8080"
	server := &http.Server{
		Handler: handlers.LoggingHandler(os.Stdout, r),
		Addr:    addr,
	}

	fmt.Println("Listening on", addr)
	err := server.ListenAndServe()
	if err != nil {
		panic(err)
	}
}

func bundleHandler(w http.ResponseWriter, r *http.Request) {
	var err error

	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET")
	w.Header().Set("Access-Control-Allow-Headers", "*")
	w.Header().Set("Content-Type", "application/wasm")

	site := mux.Vars(r)["site"]
	bundleID := mux.Vars(r)["bundle"]

	if site == "" || bundleID == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	key := fmt.Sprintf("%s/%s", site, bundleID)

	rawPolicyCode, ok := policyData[key]
	if !ok {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	module, err := ast.ParseModuleWithOpts("policy.rego", rawPolicyCode, ast.ParserOptions{})
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Println(err)
		return
	}

	b := bundle.Bundle{
		Modules: []bundle.ModuleFile{
			{
				URL:    "policy.rego",
				Raw:    []byte(policyData["application/form"]),
				Parsed: module,
			},
		},
	}

	buf := bytes.NewBuffer(nil)

	compiler := compile.New().
		WithCapabilities(ast.CapabilitiesForThisVersion()).
		WithTarget("wasm").
		WithOutput(buf).
		WithOptimizationLevel(0).
		WithEntrypoints(entrypoint).
		WithBundle(&b)

	err = compiler.Build(r.Context())
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Println(err)
		return
	}

	gzReader, err := gzip.NewReader(buf)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Println(err)
		return
	}

	tarReader := tar.NewReader(gzReader)
	for true {
		header, err := tarReader.Next()

		if err == io.EOF {
			break
		}

		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Println(err)
			return
		}

		switch header.Typeflag {
		case tar.TypeReg:
			if strings.HasSuffix(header.Name, ".wasm") {
				_, err := io.Copy(w, tarReader)
				if err != nil {
					w.WriteHeader(http.StatusInternalServerError)
					fmt.Println(err)
					return
				}
			}
		}
	}
}

func configIndexHandler(w http.ResponseWriter, r *http.Request) {
	bs, err := templates.ReadFile("templates/index.html")
	if err != nil {
		fmt.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	t := template.New("page")
	ct, err := t.Parse(string(bs))
	if err != nil {
		fmt.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	err = ct.Execute(w, struct {
		Policies map[string]string
	}{
		Policies: policyData,
	})
	if err != nil {
		fmt.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

func configShowHandler(w http.ResponseWriter, r *http.Request) {
	site := mux.Vars(r)["site"]
	bundleID := mux.Vars(r)["bundle"]

	key := fmt.Sprintf("%s/%s", site, bundleID)

	if r.Method == http.MethodPost {
		err := r.ParseForm()
		if err != nil {
			fmt.Println(err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		value := r.Form.Get("value")
		policyData[key] = value
	}

	bs, err := templates.ReadFile("templates/config.html")
	if err != nil {
		fmt.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	t := template.New("page")
	ct, err := t.Parse(string(bs))
	if err != nil {
		fmt.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	current, _ := policyData[key]

	var currentFormatted bytes.Buffer
	err = quick.Highlight(&currentFormatted, current, "ruby", "html", "monokailight")
	if err != nil {
		fmt.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	notes := policyNotes[key]
	for i := range notes {
		notes[i].Render()
	}

	err = ct.Execute(w, struct {
		Key         string
		Current     string
		Notes       []policyNote
		CurrentHTML template.HTML
	}{
		Key:         key,
		Current:     current,
		Notes:       notes,
		CurrentHTML: template.HTML(currentFormatted.String()),
	})
	if err != nil {
		fmt.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}
