package sentinels_app

import (
	"fmt"
  "log"
  "net/http"
  "html/template"
	"strconv"

	"sentinels"
)

var templates = template.Must(template.ParseFiles("form.html", "result.html"))

type result struct {
	PC int
	LP int
	RG int
	Promo bool
	Setup *sentinels.Setup
	Msg string
	Nump string
	Iterations int
}

var expansions = []string{"baseset", "miniexpansion", "rookcity", "infernalrelics", "shatteredtimelines", "vengeance", "promos"}

func handler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		templates.ExecuteTemplate(w, "form.html", "")
	case "POST":
		if m, err := formInts(r, "pc", "lp"); err != nil {
			log.Println(err)
		} else {
			exp := []sentinels.ExpansionType{}
			for i, v := range expansions {
				if r.FormValue(v) == "on" {
					exp = append(exp, sentinels.ExpansionType(i))
				}
			}
			r := &result{}
			if len(exp) == 0 {
				r.Msg = "No card set selected."
			}	else {
				r.PC = m["pc"]
				r.LP = m["lp"]
				r.Nump = fmt.Sprintf("%d heroes", m["pc"])
				var err error
				if r.Setup, r.Iterations, err = sentinels.FindSetup(r.PC, r.LP, 10, exp); err != nil {
					r.Msg = err.Error()
				}	
			}
			templates.ExecuteTemplate(w, "result.html", r)
		}
	default:
		log.Printf("Unhandled method: %s", r.Method)
	}
}

func formInts(r *http.Request, names... string) (map[string]int, error) {
	m := make(map[string]int)
	for _, n := range names {
		if i, err := strconv.Atoi(r.FormValue(n)); err != nil {
			return nil, err
		} else {
			m[n] = i
		}
	}
	return m, nil
}

func init() {
  http.HandleFunc("/", handler)
  http.ListenAndServe(":8080", nil)
}