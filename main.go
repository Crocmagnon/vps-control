package main

import (
	"context"
	"embed"
	"fmt"
	"github.com/ovh/go-ovh/ovh"
	"html/template"
	"log/slog"
	"net/http"
	"os"
	"time"
)

//go:embed templates
var templates embed.FS

func main() {
	ctx := context.Background()
	if err := run(ctx, os.Getenv); err != nil {
		slog.Error(err.Error())
		os.Exit(1)
	}
	slog.Info("exiting...")
}

func run(ctx context.Context, getenv func(key string) string) error {
	client, err := ovh.NewClient(
		getenv("OVH_ENDPOINT"),
		getenv("OVH_APPLICATION_KEY"),
		getenv("OVH_APPLICATION_SECRET"),
		getenv("OVH_CONSUMER_KEY"),
	)
	if err != nil {
		return fmt.Errorf("creating ovh client: %w", err)
	}

	tmpls, err := template.New("").
		ParseFS(templates, "templates/*.tmpl")
	if err != nil {
		return fmt.Errorf("parsing templates: %w", err)
	}

	http.HandleFunc("GET /{$}", rootHandler(client, tmpls))
	http.HandleFunc("GET /vps/{$}", vpsHandler(client, tmpls))
	http.HandleFunc("POST /vps/reboot/{$}", rebootHandler(client, tmpls))
	http.HandleFunc("GET /vps/task/{$}", taskHandler(client, tmpls))

	listenAddr := getenv("LISTEN_ADDR")

	slog.Info("listening", "addr", listenAddr)

	if err := http.ListenAndServe(listenAddr, nil); err != nil {
		return fmt.Errorf("serving http: %w", err)
	}

	return nil
}

func rootHandler(client *ovh.Client, tmpls *template.Template) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		var vpsList []string

		if err := client.GetWithContext(r.Context(), "/vps", &vpsList); err != nil {
			slog.Error(err.Error())
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		slog.Info("vps list", "vps", vpsList)
		templateResp(w, tmpls, "vps-list.tmpl", vpsList)
	}
}

func vpsHandler(_ *ovh.Client, tmpls *template.Template) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		vps := r.URL.Query().Get("id")
		slog.Info("vps", "vps", vps)

		templateResp(w, tmpls, "vps-item.tmpl", vps)
	}
}

func rebootHandler(client *ovh.Client, tmpls *template.Template) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		vps := r.FormValue("id")
		slog.Info("reboot vps", "vps", vps)

		var resp RebootResponse

		if err := client.PostWithContext(r.Context(), fmt.Sprintf("/vps/%s/reboot", vps), nil, &resp); err != nil {
			slog.Error(err.Error())
			w.WriteHeader(http.StatusInternalServerError)
			templateResp(w, tmpls, "error.tmpl", err.Error())
			return
		}

		w.Header().Set("Location", fmt.Sprintf("/vps/task?vpsId=%v&taskId=%v", vps, resp.Id))
		w.WriteHeader(http.StatusSeeOther)
	}
}

func taskHandler(client *ovh.Client, tmpls *template.Template) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		vps := r.URL.Query().Get("vpsId")
		task := r.URL.Query().Get("taskId")
		slog.Info("task vps", "vps", vps, "task", task)

		var resp RebootResponse

		if err := client.GetWithContext(r.Context(), fmt.Sprintf("/vps/%s/tasks/%s", vps, task), &resp); err != nil {
			slog.Error(err.Error())
			w.WriteHeader(http.StatusInternalServerError)
			templateResp(w, tmpls, "error.tmpl", err.Error())
			return
		}

		templateResp(w, tmpls, "vps-task.tmpl", resp)
	}
}

func templateResp(w http.ResponseWriter, tmpls *template.Template, name string, data any) {
	err := tmpls.ExecuteTemplate(w, name, data)
	if err != nil {
		slog.Error(err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

type RebootResponse struct {
	Date     time.Time `json:"date"`
	Id       int       `json:"id"`
	Progress int       `json:"progress"`
	State    string    `json:"state"`
	Type     string    `json:"type"`
}
