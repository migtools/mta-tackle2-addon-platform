package cloudfoundry

import (
	"path/filepath"

	cf "github.com/cloudfoundry/go-cfclient/v3/config"
	cfp "github.com/konveyor/asset-generation/pkg/providers/discoverers/cloud_foundry"
	hub "github.com/konveyor/tackle2-hub/addon"
	"github.com/konveyor/tackle2-hub/api"
	"github.com/konveyor/tackle2-hub/api/jsd"
	"github.com/konveyor/tackle2-hub/migration/json"
)

var (
	addon = hub.Addon
)

// Provider is a cloudfoundry provider.
type Provider struct {
	URL      string
	Identity *api.Identity
}

// Use identity.
func (p *Provider) Use(identity *api.Identity) {
	p.Identity = identity
}

// Fetch the manifest for the application.
func (p *Provider) Fetch(application *api.Application) (m *api.Manifest, err error) {
	if application.Coordinates == nil {
		err = &CoordinatesError{}
		return
	}
	coordinates := Coordinates{}
	err = application.Coordinates.As(&coordinates)
	if err != nil {
		return
	}
	ref := cfp.AppReference{
		OrgName:   coordinates.Organization,
		SpaceName: coordinates.Space,
		AppName:   coordinates.Name,
	}
	client, err := p.client(coordinates.Filter())
	if err != nil {
		return
	}
	manifest, err := client.Discover(ref)
	if err != nil {
		err = Wrap(err)
		return
	}
	m = &api.Manifest{}
	m.Content = manifest.Content
	m.Secret = manifest.Secret
	return
}

// Find applications on the platform.
func (p *Provider) Find(filter api.Map) (found []api.Application, err error) {
	f := Filter{}
	err = filter.As(&f)
	if err != nil {
		return
	}
	client, err := p.client(f)
	if err != nil {
		return
	}
	refs, err := client.ListApps()
	if err != nil {
		return
	}
	schema, err := addon.Schema.Find("platform", "cloudfoundry", "coordinates")
	if err != nil {
		return
	}
	for _, applications := range refs {
		for _, ref := range applications {
			appRef := ref.(cfp.AppReference)
			if !f.Match(appRef) {
				continue
			}
			r := api.Application{}
			r.Name = appRef.AppName
			r.Coordinates = &jsd.Document{
				Schema: schema.Name,
				Content: json.Map{
					"organization": appRef.OrgName,
					"space":        appRef.SpaceName,
					"name":         appRef.AppName,
				},
			}
			found = append(found, r)
		}
	}
	return
}

// Tag returns the platform= tag name.
func (p *Provider) Tag() string {
	return "Cloud Foundry"
}

// client returns a cloudfoundry client.
func (p *Provider) client(filter Filter) (client *cfp.CloudFoundryProvider, err error) {
	options := []cf.Option{
		cf.SkipTLSValidation(),
	}
	if p.Identity != nil {
		options = append(
			options,
			cf.UserPassword(
				p.Identity.User,
				p.Identity.Password))
	}
	cfConfig, err := cf.New(p.URL, options...)
	if err != nil {
		err = Wrap(err)
		return
	}
	pConfig := &cfp.Config{
		CloudFoundryConfig: cfConfig,
		OrgNames:           filter.Organizations,
		SpaceNames:         filter.Spaces,
	}
	client, err = cfp.New(pConfig, &addon.Log, true)
	if err != nil {
		err = Wrap(err)
		return
	}
	return
}

// Coordinates - platform coordinates.
type Coordinates struct {
	Organization string `json:"organization"`
	Space        string `json:"space"`
	Name         string `json:"name"`
}

// Filter returns a filter.
func (c *Coordinates) Filter() (f Filter) {
	f.Organizations = []string{c.Organization}
	f.Spaces = []string{c.Space}
	f.Names = []string{c.Name}
	return
}

// Filter applications.
type Filter struct {
	Organizations []string `json:"organizations"`
	Spaces        []string `json:"spaces"`
	Names         []string `json:"names"`
}

// Match returns true when the ref matches the filter.
func (f *Filter) Match(ref cfp.AppReference) (match bool) {
	match = f.MatchOrganization(ref.OrgName) &&
		f.MatchSpace(ref.SpaceName) &&
		f.MatchName(ref.AppName)
	return
}

// MatchOrganization returns true when the organization name matches the filter.
func (f *Filter) MatchOrganization(name string) (match bool) {
	if len(f.Organizations) == 0 {
		match = true
		return
	}
	for _, s := range f.Organizations {
		match = s == name
		if match {
			break
		}
	}
	return
}

// MatchSpace returns true when the space name matches the filter.
func (f *Filter) MatchSpace(name string) (match bool) {
	if len(f.Spaces) == 0 {
		match = true
		return
	}
	for _, s := range f.Spaces {
		match = s == name
		if match {
			break
		}
	}
	return
}

// MatchName returns true when the name matches the filter.
// The name may be a glob.
func (f *Filter) MatchName(name string) (match bool) {
	var err error
	if len(f.Names) == 0 {
		match = true
		return
	}
	for _, pattern := range f.Names {
		match, err = filepath.Match(pattern, name)
		if err != nil {
			addon.Log.Error(err, "Invalid glob pattern", "pattern", pattern)
			continue
		}
		if match {
			break
		}
	}
	return
}
