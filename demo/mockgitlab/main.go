// mockgitlab: a tiny fake GitLab API with made-up projects, used to produce README screenshots
// without touching a real instance.
//
//	go run ./demo/mockgitlab            # listens on :8080
//	go run . --host=http://localhost:8080 --group-id=acme --token=demo
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type job struct {
	name, stage, status string
	secs                float64
	allowFailure        bool
}

type project struct {
	id       int
	name     string
	ref      string
	status   string
	age      time.Duration
	sha      string
	title    string
	author   string
	username string
	jobs     []job
}

var projects = []project{
	{1, "acme-api", "main", "running", 95 * time.Second, "a1b2c3d", "Add rate limiting to /v2/orders", "Sam Rivera", "srivera", []job{
		{"lint", "build", "success", 18, false}, {"compile", "build", "success", 74, false},
		{"unit-tests", "test", "success", 141, false}, {"integration-tests", "test", "running", 62, false}, {"coverage", "test", "pending", 0, false},
		{"docker-image", "package", "created", 0, false},
		{"deploy-staging", "deploy", "created", 0, false}, {"deploy-prod", "deploy", "manual", 0, false},
	}},
	{2, "acme-web", "refs/merge-requests/412/head", "failed", 14 * time.Minute, "9f8e7d6", "Fix cart total rounding on checkout", "Jane Doe", "jdoe", []job{
		{"lint", "build", "success", 22, false}, {"bundle", "build", "success", 98, false},
		{"unit-tests", "test", "success", 87, false}, {"e2e-chrome", "test", "failed", 213, false}, {"e2e-firefox", "test", "canceled", 40, false},
		{"visual-regression", "test", "failed", 55, true},
		{"deploy-preview", "deploy", "skipped", 0, false},
	}},
	{3, "acme-worker", "main", "success", 3 * time.Hour, "5c4b3a2", "Retry failed jobs with exponential backoff", "Priya Nair", "pnair", []job{
		{"lint", "build", "success", 15, false}, {"compile", "build", "success", 61, false},
		{"unit-tests", "test", "success", 120, false}, {"integration-tests", "test", "success", 188, false},
		{"docker-image", "package", "success", 93, false},
		{"deploy-staging", "deploy", "success", 47, false}, {"deploy-prod", "deploy", "success", 52, false},
	}},
	{4, "acme-infra", "main", "success", 26 * time.Hour, "d00dfee", "Bump terraform AWS provider to 5.x", "Lee Chen", "lchen", []job{
		{"fmt", "validate", "success", 6, false}, {"validate", "validate", "success", 11, false},
		{"tflint", "validate", "failed", 9, true},
		{"plan", "plan", "success", 84, false},
		{"apply", "apply", "success", 305, false},
	}},
	{5, "acme-docs", "docs/new-quickstart", "pending", 12 * time.Second, "7e6d5c4", "Rewrite quickstart guide", "Jane Doe", "jdoe", []job{
		{"build-site", "build", "pending", 0, false}, {"link-check", "test", "created", 0, false}, {"publish", "deploy", "created", 0, false},
	}},
	{6, "acme-mobile", "release/3.2", "success", 5 * 24 * time.Hour, "b3a1c9e", "Release 3.2.0", "Sam Rivera", "srivera", []job{
		{"lint", "build", "success", 31, false}, {"build-android", "build", "success", 412, false}, {"build-ios", "build", "success", 538, false},
		{"unit-tests", "test", "success", 160, false},
		{"upload-testflight", "deploy", "success", 77, false},
	}},
}

func main() {
	addr := flag.String("addr", ":8080", "listen address")
	flag.Parse()
	start := time.Now()
	ts := func(t time.Time) string { return t.UTC().Format(time.RFC3339) }

	http.HandleFunc("/api/v4/groups/", func(w http.ResponseWriter, r *http.Request) {
		var out []map[string]any
		for _, p := range projects {
			out = append(out, map[string]any{"id": p.id, "name": p.name, "web_url": "https://git.example.com/acme/" + p.name})
		}
		json.NewEncoder(w).Encode(out)
	})
	http.HandleFunc("/api/v4/projects/", func(w http.ResponseWriter, r *http.Request) {
		parts := strings.Split(strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/v4/projects/"), "/"), "/")
		id, _ := strconv.Atoi(parts[0])
		if id < 1 || id > len(projects) {
			http.NotFound(w, r)
			return
		}
		p := projects[id-1]
		created := start.Add(-p.age)
		updated := created.Add(time.Duration(sumSecs(p)) * time.Second)
		if p.status == "running" || p.status == "pending" {
			updated = start
		}
		switch {
		case len(parts) == 2: // /projects/:id/pipelines
			json.NewEncoder(w).Encode([]map[string]any{{
				"id": 1000 + id, "iid": 100 + id, "status": p.status, "ref": p.ref, "sha": p.sha,
				"web_url":    fmt.Sprintf("https://git.example.com/acme/%s/-/pipelines/%d", p.name, 1000+id),
				"created_at": ts(created), "updated_at": ts(updated),
			}})
		case len(parts) == 4: // /projects/:id/pipelines/:pid/jobs
			var out []map[string]any
			t := created
			for i, j := range p.jobs {
				m := map[string]any{
					"id": id*100 + i, "name": j.name, "stage": j.stage, "status": j.status, "allow_failure": j.allowFailure,
					"web_url": fmt.Sprintf("https://git.example.com/acme/%s/-/jobs/%d", p.name, id*100+i),
					"commit":  map[string]any{"short_id": p.sha, "title": p.title, "author_name": p.author},
					"user":    map[string]any{"username": p.username},
				}
				if j.secs > 0 {
					m["duration"] = j.secs
					m["started_at"] = ts(t)
					t = t.Add(time.Duration(j.secs) * time.Second)
				}
				if j.status == "running" {
					m["duration"] = nil
					m["started_at"] = ts(start.Add(-time.Duration(j.secs) * time.Second))
				}
				out = append(out, m)
			}
			json.NewEncoder(w).Encode(out)
		}
	})
	log.Printf("mock GitLab listening on %s", *addr)
	log.Fatal(http.ListenAndServe(*addr, nil))
}

func sumSecs(p project) float64 {
	var s float64
	for _, j := range p.jobs {
		s += j.secs
	}
	return s
}
