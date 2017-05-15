package tea

import (
	"html/template"
	"net/http"
	"time"

	"appengine"
	"appengine/datastore"
)

type pageInfo struct {
	When   string // Time tea was last reported
	Exists bool   // Does tea exist?
	Un     bool   // Was tea unreported?
	Box    string // Are we in the sandbox?
}

const page = `
<html>
<head>
<title>Is there tea yet?</title>

<meta name="viewport" content="width=device-width, initial-scale=1">

<style>
footer {
     font-size: 70%;
     font-family: arial, sans-serif;
     position: absolute;
	right: 0;
	bottom: 0;
	left: 0;
	padding: 1rem;
	text-align: center;
}
td {
     text-align: center;
     background-color: #668800;
     color: white;
     font-family: arial, sans-serif;
     padding: 0 0 0 0;
}
tr {
     padding: 0 0 0 0;
     border: none;
}
table {
     border: none;
     border-collapse: collapse;
}
.bigtype {
     font-size: 150%;
}
.smallwidth {
     width: 30%;
}
.centered {
     margin: auto;
}

</style>

</head>

<body>

<table class="centered">
  <tr>
    <td colspan="3" class="bigtype">Tea Status</td>
  </tr>
  <tr>
    <td>&nbsp;</td>
    <td><img src="static/{{if not .Exists}}n{{end}}exists.png"></td>
    <td>&nbsp;</td>
  </tr>
  <tr>
    <td colspan="3">Tea was last {{if .Un}}un{{end}}reported {{.When}}</td>
  </tr>
  <tr>
    <td colspan="3"><a href="{{.Box}}/{{if .Exists}}un{{end}}set"><img class="smallwidth" src="static/button.png"></a></td>
  </tr>
</table>



<footer class="centered">App by Anschel Schaffer-Cohen. Design by Cullen Schaffer. Tea emoji in the favicon provided by <a href="https://www.emojione.com/">EmojiOne</a>.</footer>
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
	http.HandleFunc("/last/set", setTime("last", true))
	http.HandleFunc("/last/unset", setTime("last", false))
	http.HandleFunc("/sandbox", getTime("sandbox"))
	http.HandleFunc("/sandbox/set", setTime("sandbox", true))
	http.HandleFunc("/sandbox/unset", setTime("sandbox", false))
}

func recent(t time.Time) bool {
	d := time.Since(t)
	return d < longEnough
}

func format(t time.Time) string {
	d := time.Since(t)
	if d < 24*time.Hour {
		return t.In(philly).Format("at 3:04 PM")
	}
	return "more than 24 hours ago"
}

type last struct {
	T      time.Time
	Exists bool
}

// These two handler functions can take any box, not just /last and /sandbox.
// But that would open up to an attack that makes database size grow linearly
// with the number of requests, which is probably bad. Thus the handlers have
// those two possible boxes hardcoded.

// For each box, the datastore contains just one entity, whose key is the name
// of the box and whose values are the time of the last report and whether tea
// existed at that time.  setTime() stores the current time, and getTime()
// fetches the stored time.  Note that time is stored without any timezone
// changes--presumably Google is using UTC internally--and only converted to
// local time on output. That means (for instance) that recent() will give
// correct answers during the DST changeover.

func setTime(box string, exists bool) func(http.ResponseWriter, *http.Request) {
	// http.HandleFunc() wants to be given a function with ResponseWriter
	// and *Request arguments, but we want that function to know which box
	// and exists value to use; so we use a closure like this.
	return func(w http.ResponseWriter, r *http.Request) {
		c := appengine.NewContext(r)
		k := datastore.NewKey(c, "time", box, 0, nil)
		if _, err := datastore.Put(c, k, &last{T: time.Now(), Exists: exists}); err != nil {
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
			When:   format(l.T),
			Exists: l.Exists && recent(l.T),
			Un:     !l.Exists,
			Box:    box,
		})
	}
}
