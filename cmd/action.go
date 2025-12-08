package main

import (
	"errors"
	"strings"

	"github.com/konveyor/tackle2-addon-platform/cmd/cloudfoundry"
	"github.com/konveyor/tackle2-addon-platform/cmd/helm"
	"github.com/konveyor/tackle2-hub/api"
)

// Action provides and addon action.
type Action interface {
	Run(*Data) error
}

// NewAction returns and action based the data provided.
func NewAction(d *Data) (a Action, err error) {
	switch d.Action {
	case "fetch":
		a = &Fetch{}
	case "import":
		a = &Import{}
	case "generate":
		a = &Generate{}
	default:
		err = errors.New("action not supported")
	}
	return
}

// BaseAction provides base functionality.
type BaseAction struct {
	application api.Application
	platform    api.Platform
}

// setApplication fetches and sets `application` referenced by the task.
// The associated `platform` will be set when as appropriate.
func (r *BaseAction) setApplication() (err error) {
	defer func() {
		if err != nil {
			if errors.Is(err, &api.NotFound{}) {
				err = nil
			}
		}
	}()
	app, err := addon.Task.Application()
	if err == nil {
		r.application = *app
	} else {
		return
	}
	if app.Platform == nil {
		return
	}
	p, err := addon.Platform.Get(app.Platform.ID)
	if err == nil {
		r.platform = *p
	}
	return
}

// setPlatform fetches and sets `platform` referenced by the task.
func (r *BaseAction) setPlatform() (err error) {
	defer func() {
		if err != nil {
			if errors.Is(err, &api.NotFound{}) {
				err = nil
			}
		}
	}()
	p, err := addon.Task.Platform()
	if err == nil {
		r.platform = *p
	}
	return
}

// selectProvider returns a platform provider based on kind.
func (r *BaseAction) selectProvider(kind string) (p Provider, err error) {
	switch strings.ToLower(kind) {
	case "cloudfoundry":
		p = &cloudfoundry.Provider{
			URL: r.platform.URL,
		}
	default:
		err = errors.New("platform kind not supported")
		err = wrap(err)
		return
	}
	if r.platform.Identity == nil || r.platform.Identity.ID == 0 {
		return
	}
	id, err := addon.Identity.Get(r.platform.Identity.ID)
	if err != nil {
		return
	}
	p.Use(id)
	addon.Activity(
		"[Provider] Using credentials (id=%d): %s",
		id.ID,
		id.Name)
	return
}

// selectEngine returns a template engine based on kind.
func (r *BaseAction) selectEngine(kind string) (e Engine, err error) {
	switch strings.ToLower(kind) {
	case "helm":
		e = &helm.Engine{}
	default:
		err = errors.New("generator kind not supported")
		err = wrap(err)
		return
	}
	return
}

// Provider is platform provider.
type Provider interface {
	Use(identity *api.Identity)
	Fetch(application *api.Application) (m *api.Manifest, err error)
	Find(filter api.Map) (found []api.Application, err error)
	Tag() string
}

// Engine is a template rendering engine.
type Engine interface {
	Render(templateDir string, values api.Map) (files Files, err error)
}
