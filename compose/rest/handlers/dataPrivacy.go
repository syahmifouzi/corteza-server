package handlers

// This file is auto-generated.
//
// Changes to this file may cause incorrect behavior and will be lost if
// the code is regenerated.
//
// Definitions file that controls how this file is generated:
//

import (
	"context"
	"github.com/cortezaproject/corteza-server/compose/rest/request"
	"github.com/cortezaproject/corteza-server/pkg/api"
	"github.com/go-chi/chi/v5"
	"net/http"
)

type (
	// Internal API interface
	DataPrivacyAPI interface {
		SensitiveDataList(context.Context, *request.DataPrivacySensitiveDataList) (interface{}, error)
	}

	// HTTP API interface
	DataPrivacy struct {
		SensitiveDataList func(http.ResponseWriter, *http.Request)
	}
)

func NewDataPrivacy(h DataPrivacyAPI) *DataPrivacy {
	return &DataPrivacy{
		SensitiveDataList: func(w http.ResponseWriter, r *http.Request) {
			defer r.Body.Close()
			params := request.NewDataPrivacySensitiveDataList()
			if err := params.Fill(r); err != nil {
				api.Send(w, r, err)
				return
			}

			value, err := h.SensitiveDataList(r.Context(), params)
			if err != nil {
				api.Send(w, r, err)
				return
			}

			api.Send(w, r, value)
		},
	}
}

func (h DataPrivacy) MountRoutes(r chi.Router, middlewares ...func(http.Handler) http.Handler) {
	r.Group(func(r chi.Router) {
		r.Use(middlewares...)
		r.Get("/data-privacy/sensitive-data", h.SensitiveDataList)
	})
}
