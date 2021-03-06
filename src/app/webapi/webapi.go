// Package webapi Web API
//
// This is the API for the application.
//
// Swagger 2.0 Spec - generated by [go-swagger](https://github.com/go-swagger/go-swagger)
//
// Schemes: http
// Host: localhost:8080
// BasePath: /
// Version: 2.0
//
// Consumes:
// - application/x-www-form-urlencoded
//
// Produces:
// - application/json
//
// SecurityDefinitions:
// token:
//   type: apiKey
//   name: Authorization
//   in: header
//   description: "The following syntax must be used in the Authorization header: Bearer TOKEN"
//
// swagger:meta
package webapi

import (
	"encoding/json"
	"net/http"
	"os"

	"app/webapi/component"
	"app/webapi/component/auth"
	"app/webapi/component/root"
	"app/webapi/component/user"
	"app/webapi/internal/bind"
	"app/webapi/internal/response"
	"app/webapi/middleware"
	"app/webapi/pkg/database"
	"app/webapi/pkg/logger"
	"app/webapi/pkg/query"
	"app/webapi/pkg/router"
	"app/webapi/pkg/server"
	"app/webapi/pkg/webtoken"
)

// *****************************************************************************
// Application Settings
// *****************************************************************************

// AppConfig contains the application settings with JSON tags.
type AppConfig struct {
	Database database.Connection    `json:"Database"`
	Server   server.Config          `json:"Server"`
	JWT      webtoken.Configuration `json:"JWT"`
}

// ParseJSON unmarshals the JSON bytes to the struct.
func (c *AppConfig) ParseJSON(b []byte) error {
	return json.Unmarshal(b, &c)
}

// *****************************************************************************
// Application Logic
// *****************************************************************************

// Routes will set up the components and return the router.
func Routes(config *AppConfig, appLogger logger.ILog) (*router.Mux,
	*http.Server, *http.Server) {
	// Set up the dependencies.
	l := logger.New(appLogger)
	db := Database(config.Database, l)
	q := query.New(db)
	b := bind.New()
	resp := response.New()
	t := webtoken.New(config.JWT.Secret)

	// Create the component core.
	core := component.NewCore(l, db, q, b, resp, t)

	// Set up the routes.
	r := router.New()
	root.New(core).Routes(r)
	auth.New(core).Routes(r)
	user.New(core).Routes(r)

	// Set up the 404 page.
	r.Instance().NotFound = router.Handler(
		func(w http.ResponseWriter, r *http.Request) (int, error) {
			return http.StatusNotFound, nil
		})

	// Set the handling of all responses.
	router.ServeHTTP = func(w http.ResponseWriter, r *http.Request, status int, err error) {
		// Handle only errors.
		if status >= 400 {
			resp := new(response.GenericResponse)
			resp.Body.Status = http.StatusText(status)
			if err != nil {
				resp.Body.Message = err.Error()
			}

			// Write the content.
			w.WriteHeader(status)
			w.Header().Set("Content-Type", "application/json")
			err := json.NewEncoder(w).Encode(resp.Body)
			if err != nil {
				w.Write([]byte(`{"status":"Internal Server Error","message":"problem encoding JSON"}`))
				return
			}
		}

		// Display server errors.
		if status >= 500 {
			if err != nil {
				l.Printf("%v", err)
			}
		}
	}

	// Set up the HTTP listener.
	httpServer := new(http.Server)
	httpServer.Addr = config.Server.HTTPAddress()

	// Determine if HTTP should redirect to HTTPS.
	if config.Server.ForceHTTPSRedirect {
		httpServer.Handler = http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			http.Redirect(w, req, "https://"+req.Host, http.StatusMovedPermanently)
		})
	} else {
		httpServer.Handler = middleware.Wrap(r, appLogger, config.JWT.Secret)
	}

	// Set up the HTTPS listener.
	httpsServer := new(http.Server)
	httpsServer.Addr = config.Server.HTTPSAddress()
	httpsServer.Handler = middleware.Wrap(r, appLogger, config.JWT.Secret)

	return r, httpServer, httpsServer
}

// Database returns the database connection.
func Database(dbc database.Connection, l logger.ILog) *database.DBW {
	// Set the database password from an environment variable.
	pwd := os.Getenv("DB_PASSWORD")
	if len(pwd) > 0 {
		dbc.Password = pwd
	}

	connection, err := dbc.Connect(true)
	if err != nil {
		// Don't fail here, just show an error message.
		l.Printf("DB Error: %v", err)
	}
	// Wrap the DB connection.
	db := database.New(connection)

	return db
}
