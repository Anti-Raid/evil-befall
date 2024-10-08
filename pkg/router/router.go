// Interface for a page
package router

import (
	"errors"
	"fmt"

	"github.com/anti-raid/evil-befall/pkg/state"
)

var (
	ErrRouteNotFound = errors.New("route not found")
)

var routes = []Route{}

func AddRoute(r Route) {
	if r := GetRoute(r.Command()); r != nil {
		panic(fmt.Sprintf("route %s already exists", r.Command()))
	}

	routes = append(routes, r)
}

func Routes() []Route {
	return routes
}

func GetRoute(id string) Route {
	for _, r := range routes {
		if r.Command() == id {
			return r
		}
	}

	return nil
}

// Get current route
func GetCurrentRoute(state *state.State) Route {
	for _, route := range routes {
		if state.CurrentLoc.ID == route.Command() {
			return route
		}
	}

	return nil
}

// Goto page based on the states current location
func GotoCurrent(state *state.State, args map[string]string) (Route, error) {
	for _, route := range routes {
		if state.CurrentLoc.ID == route.Command() {
			return route, Goto(route.Command(), state, args)
		}
	}

	return nil, errors.New("route not found")
}

func Goto(id string, state *state.State, args map[string]string) error {
	// Persist state if persist mode is enabled
	err := state.PersistToDisk()

	if err != nil {
		return fmt.Errorf("failed to persist state to disk: %w", err)
	}

	// Update the state
	state.CurrentLoc.ID = id
	state.CurrentLoc.Data = args

	// Get current route
	currentRoute := GetCurrentRoute(state)

	// If the current route is not nil, destroy it
	if currentRoute != nil {
		if err := currentRoute.Destroy(state); err != nil {
			return err
		}
	}

	r := GetRoute(id)

	if r == nil {
		return ErrRouteNotFound
	}

	if err := r.Setup(state); err != nil {
		return err
	}

	err = r.Render(state, args)

	if err != nil {
		return err
	}

	// Persist state if persist mode is enabled
	err = state.PersistToDisk()

	if err != nil {
		return fmt.Errorf("failed to persist state to disk: %w", err)
	}

	return nil
}

type Route interface {
	// The command name of the route
	Command() string

	// The description of the route
	Description() string

	// The arguments the route can take
	// [][3]string // Map of argument to the description and default value
	Arguments() [][3]string

	// Given a current state, sets up all state for the route
	Setup(state *state.State) error

	// Called on destruction of the route
	Destroy(state *state.State) error

	// Renders the route
	Render(state *state.State, args map[string]string) error
}

type CompletableRoute interface {
	Completion(state *state.State, line string, args map[string]string) ([]string, error)
}
