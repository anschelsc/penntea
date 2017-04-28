package hello

import (
	"fmt"
	"net/http"
	"time"

	"appengine"
	"appengine/datastore"
)

var philly *time.Location

func init() {
	var err error
	if philly, err = time.LoadLocation("America/New_York"); err != nil {
		panic("What happened to New York?")
	}
	http.HandleFunc("/", getTime)
	http.HandleFunc("/set", setTime)
}

type Last struct {
	T time.Time
}

func setTime(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)
	k := datastore.NewKey(c, "time", "last", 0, nil)
	if _, err := datastore.Put(c, k, &Last{time.Now()}); err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	http.Redirect(w, r, "/", http.StatusFound)
}

func getTime(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)
	k := datastore.NewKey(c, "time", "last", 0, nil)
	l := new(Last)
	if err := datastore.Get(c, k, l); err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	fmt.Fprint(w, l.T.In(philly).Format("January 2 (Monday) at 3:04 PM"))
}
