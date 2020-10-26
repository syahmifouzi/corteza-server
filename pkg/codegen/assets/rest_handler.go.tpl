package handlers

// This file is auto-generated.
//
// Changes to this file may cause incorrect behavior and will be lost if
// the code is regenerated.
//
// Definitions file that controls how this file is generated:
// {{ .Source }}

import (
	"context"
	"net/http"
	"github.com/go-chi/chi"
	"github.com/titpetric/factory/resputil"

	"github.com/cortezaproject/corteza-server/{{ .App }}/rest/request"
	"github.com/cortezaproject/corteza-server/pkg/logger"
)

type (
    // Internal API interface
    {{ export $.Endpoint.Entrypoint }}API interface {
    {{- range $a := $.Endpoint.Apis }}
        {{ export $a.Name }}(context.Context, *request.{{ export $.Endpoint.Entrypoint $a.Name }}) (interface{}, error)
    {{- end }}
    }

    // HTTP API interface
    {{ export .Endpoint.Entrypoint }} struct {
    {{- range $a := .Endpoint.Apis }}
        {{ export $a.Name }} func(http.ResponseWriter, *http.Request)
    {{- end }}
    }
)


func {{ export "New" $.Endpoint.Entrypoint }}(h {{ export $.Endpoint.Entrypoint }}API) *{{ export $.Endpoint.Entrypoint }} {
	return &{{ export $.Endpoint.Entrypoint }}{
    {{- range $a := .Endpoint.Apis }}
		{{ export $a.Name }}: func(w http.ResponseWriter, r *http.Request) {
			defer r.Body.Close()
			params := request.New{{ export $.Endpoint.Entrypoint $a.Name }}()
			if err := params.Fill(r); err != nil {
				logger.LogParamError("{{ export $.Endpoint.Entrypoint }}.{{ export $a.Name }}", r, err)
				resputil.JSON(w, err)
				return
			}

			value, err := h.{{ export $a.Name }}(r.Context(), params)
			if err != nil {
				logger.LogControllerError("{{ export $.Endpoint.Entrypoint }}.{{ export $a.Name }}", r, err, params.Auditable())
				resputil.JSON(w, err)
				return
			}
			logger.LogControllerCall("{{ export $.Endpoint.Entrypoint }}.{{ export $a.Name }}", r, params.Auditable())
			if !serveHTTP(value, w, r) {
				resputil.JSON(w, value)
			}
		},
    {{- end }}
	}
}

func (h {{ export $.Endpoint.Entrypoint }}) MountRoutes(r chi.Router, middlewares ...func(http.Handler) http.Handler) {
	r.Group(func(r chi.Router) {
		r.Use(middlewares...)

		{{- range $a := .Endpoint.Apis }}
		r.{{ export ( toLower $a.Method ) }}("{{ $.Endpoint.Path }}{{ $a.Path }}", h.{{ export $a.Name }})
		{{- end }}
	})
}