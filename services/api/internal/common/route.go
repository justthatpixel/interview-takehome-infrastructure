package common

import "net/http"

// Route is one endpoint a service exposes over HTTP. Services declare their
// own; the server mounts them without naming handler methods.
type Route struct {
	Path    string
	Label   string // metrics label; defaults to Path when empty
	Handler http.HandlerFunc
	Metered bool
}

// Endpoints is implemented by a service's transport package.
type Endpoints interface {
	Routes() []Route
}

// RouteLabel is the label a route reports under.
func (r Route) RouteLabel() string {
	if r.Label != "" {
		return r.Label
	}
	return r.Path
}
