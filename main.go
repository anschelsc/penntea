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
}

const page = `
<html>
<head>
<title>Is there tea yet?</title>
</head>

<body>
{{if .Recent}}
Probably!
{{else}}
Probably not.
{{end}}
Tea was last reported on {{.T}}. To report tea yourself, click <a href="/set">here</a>.
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
	http.HandleFunc("/", getTime)
	http.HandleFunc("/set", setTime)
}

func recent(t time.Time) bool {
	d := time.Since(t)
	return d < longEnough
}

type last struct {
	T time.Time
}

func setTime(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)
	k := datastore.NewKey(c, "time", "last", 0, nil)
	if _, err := datastore.Put(c, k, &last{time.Now()}); err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	http.Redirect(w, r, "/", http.StatusFound)
}

func getTime(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)
	k := datastore.NewKey(c, "time", "last", 0, nil)
	l := new(last)
	if err := datastore.Get(c, k, l); err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	pageT.Execute(w, &pageInfo{
		T:      l.T.In(philly).Format("January 2 (Monday) at 3:04 PM"),
		Recent: recent(l.T),
	})
}
