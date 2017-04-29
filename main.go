package tea

import (
	"html/template"
	"net/http"
	"time"

	"appengine"
	"appengine/datastore"
)

type pageInfo struct {
	T      string
	Recent bool
	Box    string
}

const page = `
<html>
<head>
<title>Is there tea yet?</title>

<meta name="viewport" content="width=device-width, initial-scale=1">

<style>
footer {
	position: absolute;
	right: 0;
	bottom: 0;
	left: 0;
	padding: 1rem;
	text-align: center;
}
</style>

</head>

<body>
<header>
<h1>Is there tea yet?</h1>
{{if .Recent}}
<h2>Probably!<h2>
{{else}}
<h2>Probably not.<h2>
{{end}}
</header>
Tea was last reported on {{.T}}. To report tea yourself, click <a href="/{{.Box}}/set">here</a>.
<footer>App by Anschel Schaffer-Cohen. Tea emoji in the favicon provided by <a href="https://www.emojione.com/">EmojiOne</a>.</footer>
</body>
`

var pageT = template.Must(template.New("page").Parse(page))

var philly *time.Location

const longEnough = 2 * time.Hour

func init() {
	var err error
	if philly, err = time.LoadLocation("America/New_York"); err != nil {
		panic("What happened to New York?")
	}
	http.HandleFunc("/", getTime("last"))
	http.HandleFunc("/last", getTime("last"))
	http.HandleFunc("/last/set", setTime("last"))
	http.HandleFunc("/sandbox", getTime("sandbox"))
	http.HandleFunc("/sandbox/set", setTime("sandbox"))
}

func recent(t time.Time) bool {
	d := time.Since(t)
	return d < longEnough
}

type last struct {
	T time.Time
}

func setTime(box string) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		c := appengine.NewContext(r)
		k := datastore.NewKey(c, "time", box, 0, nil)
		if _, err := datastore.Put(c, k, &last{time.Now()}); err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		http.Redirect(w, r, "/"+box, http.StatusFound)
	}
}

func getTime(box string) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		c := appengine.NewContext(r)
		k := datastore.NewKey(c, "time", box, 0, nil)
		l := new(last)
		if err := datastore.Get(c, k, l); err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		pageT.Execute(w, &pageInfo{
			T:      l.T.In(philly).Format("January 2 (Monday) at 3:04 PM"),
			Recent: recent(l.T),
			Box:    box,
		})
	}
}
