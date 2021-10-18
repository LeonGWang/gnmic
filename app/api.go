package app

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/karimra/gnmic/config"
	"github.com/karimra/gnmic/types"
	"github.com/karimra/gnmic/utils"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func (a *App) newAPIServer() (*http.Server, error) {
	a.routes()
	tlscfg, err := utils.NewTLSConfig(a.Config.APIServer.CaFile, a.Config.APIServer.CertFile, a.Config.APIServer.KeyFile, a.Config.APIServer.SkipVerify)
	if err != nil {
		return nil, err
	}
	if a.Config.APIServer.EnableMetrics {
		a.router.Handle("/metrics", promhttp.HandlerFor(a.reg, promhttp.HandlerOpts{}))
		a.reg.MustRegister(prometheus.NewGoCollector())
		a.reg.MustRegister(prometheus.NewProcessCollector(prometheus.ProcessCollectorOpts{}))
	}
	s := &http.Server{
		Addr:         a.Config.APIServer.Address,
		Handler:      a.router,
		ReadTimeout:  a.Config.APIServer.Timeout / 2,
		WriteTimeout: a.Config.APIServer.Timeout / 2,
	}

	if tlscfg != nil {
		s.TLSConfig = tlscfg
	}

	return s, nil
}

type APIErrors struct {
	Errors []string `json:"errors,omitempty"`
}

func (a *App) handleConfigTargetsGet(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
	targets, err := a.Config.GetTargets()
	if err == config.ErrNoTargetsFound {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(APIErrors{Errors: []string{err.Error()}})
		return
	}
	if err != nil && err != config.ErrNoTargetsFound {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(APIErrors{Errors: []string{err.Error()}})
		return
	}

	if id == "" {
		err = json.NewEncoder(w).Encode(targets)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(APIErrors{Errors: []string{err.Error()}})
		}
		return
	}
	if t, ok := targets[id]; ok {
		err = json.NewEncoder(w).Encode(t)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(APIErrors{Errors: []string{err.Error()}})
		}
		return
	}
	w.WriteHeader(http.StatusNotFound)
	json.NewEncoder(w).Encode(APIErrors{Errors: []string{fmt.Sprintf("target %q not found", id)}})
}

func (a *App) handleConfigTargetsPost(w http.ResponseWriter, r *http.Request) {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(APIErrors{Errors: []string{err.Error()}})
		return
	}
	defer r.Body.Close()
	tc := new(types.TargetConfig)
	err = json.Unmarshal(body, tc)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(APIErrors{Errors: []string{err.Error()}})
		return
	}
	// if _, ok := a.Config.Targets[tc.Name]; ok {
	// 	w.WriteHeader(http.StatusBadRequest)
	// 	json.NewEncoder(w).Encode(APIErrors{Errors: []string{"target config already exists"}})
	// 	return
	// }
	a.Config.Targets[tc.Name] = tc
	err = a.collector.AddTarget(tc)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(APIErrors{Errors: []string{err.Error()}})
		return
	}
}

func (a *App) handleConfigTargetsDelete(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
	a.collector.DeleteTarget(a.ctx, id)
	delete(a.Config.Targets, id)
}

func (a *App) handleConfigSubscriptions(w http.ResponseWriter, r *http.Request) {
	subsc, err := a.Config.GetSubscriptions(nil)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(APIErrors{Errors: []string{err.Error()}})
		return
	}
	err = json.NewEncoder(w).Encode(subsc)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(APIErrors{Errors: []string{err.Error()}})
		return
	}
}

func (a *App) handleConfigOutputs(w http.ResponseWriter, r *http.Request) {
	outputs, err := a.Config.GetOutputs()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(APIErrors{Errors: []string{err.Error()}})
		return
	}
	err = json.NewEncoder(w).Encode(outputs)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(APIErrors{Errors: []string{err.Error()}})
		return
	}
}

func (a *App) handleConfigClustering(w http.ResponseWriter, r *http.Request) {
	err := a.Config.GetClustering()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(APIErrors{Errors: []string{err.Error()}})
		return
	}
	err = json.NewEncoder(w).Encode(a.Config.Clustering)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(APIErrors{Errors: []string{err.Error()}})
		return
	}
}

func (a *App) handleConfigAPIServer(w http.ResponseWriter, r *http.Request) {
	err := a.Config.GetAPIServer()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(APIErrors{Errors: []string{err.Error()}})
		return
	}
	err = json.NewEncoder(w).Encode(a.Config.APIServer)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(APIErrors{Errors: []string{err.Error()}})
		return
	}
}

func (a *App) handleConfigGNMIServer(w http.ResponseWriter, r *http.Request) {
	err := a.Config.GetGNMIServer()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(APIErrors{Errors: []string{err.Error()}})
		return
	}
	err = json.NewEncoder(w).Encode(a.Config.GnmiServer)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(APIErrors{Errors: []string{err.Error()}})
		return
	}
}

func (a *App) handleConfigInputs(w http.ResponseWriter, r *http.Request) {
	inputs, err := a.Config.GetInputs()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(APIErrors{Errors: []string{err.Error()}})
		return
	}
	err = json.NewEncoder(w).Encode(inputs)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(APIErrors{Errors: []string{err.Error()}})
		return
	}
}

func (a *App) handleConfigProcessors(w http.ResponseWriter, r *http.Request) {
	evps, err := a.Config.GetEventProcessors()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(APIErrors{Errors: []string{err.Error()}})
		return
	}
	err = json.NewEncoder(w).Encode(evps)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(APIErrors{Errors: []string{err.Error()}})
		return
	}
}

func (a *App) handleConfig(w http.ResponseWriter, r *http.Request) {
	err := json.NewEncoder(w).Encode(a.Config)
	if err != nil {
		a.Logger.Print(err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(APIErrors{Errors: []string{err.Error()}})
		return
	}
}

func (a *App) handleTargetsGet(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
	if id == "" {
		err := json.NewEncoder(w).Encode(a.collector.Targets)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(APIErrors{Errors: []string{err.Error()}})
			return
		}
		return
	}
	if t, ok := a.collector.Targets[id]; ok {
		err := json.NewEncoder(w).Encode(t)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(APIErrors{Errors: []string{err.Error()}})
			return
		}
		return
	}
	w.WriteHeader(http.StatusNotFound)
	json.NewEncoder(w).Encode(APIErrors{Errors: []string{"no targets found"}})
}

func (a *App) handleTargetsPost(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
	if id == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	if tc, ok := a.Config.Targets[id]; ok {
		err := a.collector.AddTarget(tc)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(APIErrors{Errors: []string{err.Error()}})
			return
		}
	} else {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(APIErrors{Errors: []string{fmt.Sprintf("target %q not found", id)}})
		return
	}
	go a.collector.TargetSubscribeStream(a.ctx, id)
}

func (a *App) handleTargetsDelete(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
	if id == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	if _, ok := a.collector.Targets[id]; !ok {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(APIErrors{Errors: []string{fmt.Sprintf("target %q not found", id)}})
		return
	}
	err := a.collector.DeleteTarget(a.ctx, id)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(APIErrors{Errors: []string{err.Error()}})
		return
	}
}

func headersMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Content-Type", "application/json")
		next.ServeHTTP(w, r)
	})
}

func (a *App) loggingMiddleware(next http.Handler) http.Handler {
	next = handlers.LoggingHandler(a.Logger.Writer(), next)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(w, r)
	})
}
