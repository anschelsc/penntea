package tea

import (
	"html/template"
	"net/http"
	"time"

	"appengine"
	"appengine/datastore"
)

type pageInfo struct {
	T      string // Time tea was last reported
	Recent bool   // Recent enough that tea is probably still there?
	Box    string // Are we in the sandbox?
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

var philly *time.Location // For timezone correctness. Note we only convert on output.

const longEnough = 2 * time.Hour // Change this to make recent() more or less tolerant.

func init() {
	var err error
	// NB edit this if Philadelphia stops being in the same TZ as New York
	if philly, err = time.LoadLocation("America/New_York"); err != nil {
		panic("What happened to New York?")
	}
	http.HandleFunc("/", getTime("last")) // Alias for /last
	http.HandleFunc("/last", getTime("last"))
	http.HandleFunc("/last/set", setTime("last"))
	http.HandleFunc("/sandbox", getTime("sandbox"))
	http.HandleFunc("/sandbox/set", setTime("sandbox"))
}

func recent(t time.Time) bool {
	d := time.Since(t)
	return d < longEnough
}

// The datastore demands we give it structs, even if there's only one value to care about.
// This doesn't waste any space, but it's annoying to type around.
type last struct {
	T time.Time
}

// These two handler functions can take any box, not just /last and /sandbox.
// But that would open up to an attack that makes database size grow linearly
// with the number of requests, which is probably bad. Thus the handlers have
// those two possible boxes hardcoded.

// For each box, the datastore contains just one entity, whose key is the name
// of the box and whose sole value "T" is the time tea was last reported.
// setTime() stores the current time, and getTime() fetches the stored time.
// Note that time is stored without any timezone changes--presumably Google is
// using UTC internally--and only converted to local time on output. That means
// (for instance) that recent() will give correct answers during the DST
// changeover.

func setTime(box string) func(http.ResponseWriter, *http.Request) {
	// http.HandleFunc() wants to be given a function with ResponseWriter and
	// *Request arguments, but we want that function to know which box to use;
	// so we use a closure like this.
	return func(w http.ResponseWriter, r *http.Request) {
		c := appengine.NewContext(r)
		k := datastore.NewKey(c, "time", box, 0, nil)
		if _, err := datastore.Put(c, k, &last{time.Now()}); err != nil {
			// This can only fail if Google screws up
			http.Error(w, err.Error(), 500)
			return
		}
		http.Redirect(w, r, "/"+box, http.StatusFound)
	}
}

func getTime(box string) func(http.ResponseWriter, *http.Request) {
	// Same closure thing here as in setTime()
	return func(w http.ResponseWriter, r *http.Request) {
		c := appengine.NewContext(r)
		k := datastore.NewKey(c, "time", box, 0, nil)
		l := new(last)
		if err := datastore.Get(c, k, l); err != nil {
			// This error actually can happen, if tea has never been reported;
			// the first time you make a new sandbox, you should report tea
			// once yourself.
			http.Error(w, err.Error(), 500)
			return
		}
		// Write the template out to the browser, substituting reported time,
		// recentness, and box in the appropriate slots. Note the time
		// conversion.
		pageT.Execute(w, &pageInfo{
			T:      l.T.In(philly).Format("January 2 (Monday) at 3:04 PM"),
			Recent: recent(l.T),
			Box:    box,
		})
	}
}
