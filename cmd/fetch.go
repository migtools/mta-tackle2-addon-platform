package main

import "github.com/konveyor/tackle2-hub/api"

const (
	TagSource   = "platform-discovery"
	TagCategory = "Platform"
)

// Fetch application manifest action.
type Fetch struct {
	BaseAction
}

// Run executes the action.
func (a *Fetch) Run(d *Data) (err error) {
	err = a.setApplication()
	if err != nil {
		return
	}
	addon.Activity(
		"[Fetch] Fetch manifest for application (id=%d): %s",
		a.application.ID,
		a.application.Name)
	addon.Activity(
		"[Fetch] Using platform (id=%d): %s",
		a.platform.ID,
		a.platform.Name)
	provider, err := a.selectProvider(a.platform.Kind)
	if err != nil {
		return
	}
	err = a.fetch(provider, &a.application)
	return
}

// fetch manifest.
func (a *Fetch) fetch(p Provider, app *api.Application) (err error) {
	manifest, err := p.Fetch(app)
	if err != nil {
		return
	}
	manifest.Application.ID = app.ID
	err = addon.Manifest.Create(manifest)
	if err == nil {
		addon.Activity(
			"Manifest (id=%d) created.",
			manifest.ID)
	}
	return
}
