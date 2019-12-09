/* Apache v2 license
*  Copyright (C) <2019> Intel Corporation
*
*  SPDX-License-Identifier: Apache-2.0
*/

package webserver

import (
	"github.com/gorilla/mux"
	"github.impcloud.net/RSP-Inventory-Suite/loss-prevention-service/app/config"
	"github.impcloud.net/RSP-Inventory-Suite/loss-prevention-service/pkg/middlewares"
	"github.impcloud.net/RSP-Inventory-Suite/loss-prevention-service/pkg/web"
)

// Route struct holds attributes to declare routes
type Route struct {
	Name        string
	Method      string
	Pattern     string
	HandlerFunc web.Handler
}

// NewRouter creates the routes for GET and POST
func NewRouter() *mux.Router {

	handler := Handler{ServiceName: config.AppConfig.ServiceName}

	var routes = []Route{
		//swagger:operation GET / default Healthcheck
		//
		// Healthcheck Endpoint
		//
		// Endpoint that is used to determine if the application is ready to take web requests
		//
		// ---
		// consumes:
		// - application/json
		//
		// produces:
		// - application/json
		//
		// schemes:
		// - http
		//
		// responses:
		//   '200':
		//     description: OK
		//
		{
			"Index",
			"GET",
			"/",
			handler.Index,
		},
		{
			"ListRecordings",
			"GET",
			"/recordings",
			handler.ListRecordings,
		},
		{
			"DeleteAllRecordings",
			"DELETE",
			"/recordings",
			handler.DeleteAllRecordings,
		},
		{
			"OptionsDeleteAllRecordings",
			"OPTIONS",
			"/recordings",
			handler.Options,
		},
		{
			"DeleteRecording",
			"DELETE",
			"/recordings/{foldername}",
			handler.DeleteRecording,
		},
		{
			"OptionsDeleteRecording",
			"OPTIONS",
			"/recordings/{foldername}",
			handler.Options,
		},
	}

	router := mux.NewRouter().StrictSlash(true)
	for _, route := range routes {

		var handler = route.HandlerFunc
		handler = middlewares.Recover(handler)
		handler = middlewares.Logger(handler)
		handler = middlewares.Bodylimiter(handler)
		if config.AppConfig.EnableCORS {
			handler = middlewares.CORS(config.AppConfig.CORSOrigin, handler)
		}

		router.
			Methods(route.Method).
			Path(route.Pattern).
			Name(route.Name).
			Handler(handler)
	}

	return router
}
