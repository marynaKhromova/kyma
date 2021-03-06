package addon

import (
	"fmt"
	"strings"

	"github.com/Masterminds/semver"
	"github.com/kyma-project/kyma/components/helm-broker/internal"
	"github.com/pkg/errors"
	"k8s.io/helm/pkg/proto/hapi/chart"
)

type form struct {
	Meta     *FormMeta
	DocsMeta *DocsMeta
	Plans    map[string]*formPlan
}

// FormMeta describes the metdata information about the addon.
type FormMeta struct {
	ID                  string            `yaml:"id"`
	Name                string            `yaml:"name"`
	Version             string            `yaml:"version"`
	Description         string            `yaml:"description"`
	DisplayName         string            `yaml:"displayName"`
	Tags                string            `yaml:"tags"`
	ProviderDisplayName string            `yaml:"providerDisplayName"`
	LongDescription     string            `yaml:"longDescription"`
	DocumentationURL    string            `yaml:"documentationURL"`
	SupportURL          string            `yaml:"supportURL"`
	ImageURL            string            `yaml:"imageURL"`
	Bindable            bool              `yaml:"bindable"`
	ProvisionOnlyOnce   bool              `yaml:"provisionOnlyOnce"`
	Labels              map[string]string `yaml:"labels"`
	Requires            []string          `yaml:"requires"`
	BindingsRetrievable bool              `yaml:"bindingsRetrievable"`
	PlanUpdatable       *bool             `yaml:"planUpdatable"`
}

// DocsMeta contains data about addon's docs fetched from docs/meta.yaml file
type DocsMeta struct {
	Docs []internal.AddonDocs `yaml:"docs"`
}

// MapLabelsToModel maps the FormMeta.Labels to the model internal.Labels
func (m *FormMeta) MapLabelsToModel() internal.Labels {
	mapped := internal.Labels{}
	for k, v := range m.Labels {
		mapped[k] = v
	}
	return mapped
}

// MapTagsToModel maps the FormMeta.Tags to the model internal.AddonTag slice
func (m *FormMeta) MapTagsToModel() []internal.AddonTag {
	splittedTags := strings.Split(m.Tags, ",")
	mapped := make([]internal.AddonTag, 0, len(splittedTags))
	for i := range splittedTags {
		mapped = append(mapped, internal.AddonTag(strings.TrimSpace(splittedTags[i])))
	}
	return mapped
}

// Validate checks the FormMeta if all required fields are set
func (m *FormMeta) Validate() error {
	var messages []string

	if m.ID == "" {
		messages = append(messages, "missing ID field")
	}
	if m.Name == "" {
		messages = append(messages, "missing Name field")
	}
	if m.Version == "" {
		messages = append(messages, "missing Version field")
	}
	if m.Description == "" {
		messages = append(messages, "missing Description field")
	}
	if m.DisplayName == "" {
		messages = append(messages, "missing displayName field")
	}

	if len(messages) > 0 {
		return errors.New(strings.Join(messages, ", "))
	}

	return nil
}

// Validate checks the DocsMeta
func (m *DocsMeta) Validate() error {
	var messages []string

	if len(m.Docs) != 1 {
		messages = append(messages, "docs array should have at most one entry")
	}

	if len(messages) > 0 {
		return errors.New(strings.Join(messages, ", "))
	}

	return nil
}

func (f *form) Validate() error {
	var messages []string

	if f.Meta == nil {
		messages = append(messages, fmt.Sprintf("missing metadata information about addon. Please check if addon contains %q file", addonMetaName))
	}
	if len(f.Plans) == 0 {
		messages = append(messages, "addon does not contains any plans")
	}
	for name, plan := range f.Plans {
		if err := plan.Validate(); err != nil {
			messages = append(messages, fmt.Sprintf("while validating %q plan: %s", name, err.Error()))
		}
	}

	if f.Meta != nil {
		if err := f.Meta.Validate(); err != nil {
			messages = append(messages, fmt.Sprintf("while validating addon meta: %s", err.Error()))
		}
	}

	if f.DocsMeta != nil {
		if err := f.DocsMeta.Validate(); err != nil {
			messages = append(messages, fmt.Sprintf("while validating addon docs meta: %s", err.Error()))
		}
	}

	if len(messages) > 0 {
		return errors.New(strings.Join(messages, ", "))
	}

	return nil
}

func (f *form) ToModel(c *chart.Chart) (internal.Addon, error) {
	ybVer, err := semver.NewVersion(f.Meta.Version)
	if err != nil {
		return internal.Addon{}, errors.Wrap(err, "while converting form string version to semver type")
	}

	mappedPlans := make(map[internal.AddonPlanID]internal.AddonPlan)
	for name, plan := range f.Plans {
		dm, err := plan.ToModel(c)
		if err != nil {
			return internal.Addon{}, errors.Wrapf(err, "while mapping to model %q plan", name)
		}
		mappedPlans[internal.AddonPlanID(plan.Meta.ID)] = dm
	}

	var addonDocs []internal.AddonDocs
	if f.DocsMeta != nil {
		addonDocs = f.DocsMeta.Docs
	}

	return internal.Addon{
		ID:          internal.AddonID(f.Meta.ID),
		Name:        internal.AddonName(f.Meta.Name),
		Description: f.Meta.Description,
		Bindable:    f.Meta.Bindable,
		Metadata: internal.AddonMetadata{
			DisplayName:         f.Meta.DisplayName,
			DocumentationURL:    f.Meta.DocumentationURL,
			ImageURL:            f.Meta.ImageURL,
			LongDescription:     f.Meta.LongDescription,
			ProviderDisplayName: f.Meta.ProviderDisplayName,
			SupportURL:          f.Meta.SupportURL,
			ProvisionOnlyOnce:   f.Meta.ProvisionOnlyOnce,
			Labels:              f.Meta.MapLabelsToModel(),
		},
		Tags:                f.Meta.MapTagsToModel(),
		Version:             *ybVer,
		Plans:               mappedPlans,
		Requires:            f.Meta.Requires,
		BindingsRetrievable: f.Meta.BindingsRetrievable,
		PlanUpdatable:       f.Meta.PlanUpdatable,
		Docs:                addonDocs,
	}, nil
}

type formPlan struct {
	Meta          *formPlanMeta
	SchemasCreate *internal.PlanSchema
	SchemasBind   *internal.PlanSchema
	SchemasUpdate *internal.PlanSchema
	Values        map[string]interface{}
	BindTemplate  []byte
}

func (p *formPlan) Validate() error {
	if p.Meta == nil {
		return fmt.Errorf("missing metadata information about plan. Please check if plan contains %q file", addonPlanMetaName)
	}

	if p.Meta.Bindable != nil && *p.Meta.Bindable == true && p.BindTemplate == nil {
		return fmt.Errorf("plans is marked as bindable but %s file was not found in plan %s", addonPlanBindTemplateFileName, p.Meta.Name)
	}

	if err := p.Meta.Validate(); err != nil {
		return errors.Wrap(err, "while validating plan meta")
	}

	return nil
}

func (p *formPlan) ToModel(c *chart.Chart) (internal.AddonPlan, error) {
	if c == nil {
		return internal.AddonPlan{}, errors.New("missing input param chart")
	}
	if c.Metadata == nil {
		return internal.AddonPlan{}, errors.New("missing Metadata field in input param chart")
	}

	cVer, err := semver.NewVersion(c.Metadata.Version)
	if err != nil {
		return internal.AddonPlan{}, errors.Wrap(err, "while converting chart string version to semver type")
	}

	cRef := internal.ChartRef{
		Name:    internal.ChartName(c.Metadata.Name),
		Version: *cVer,
	}

	mappedSchemas := make(map[internal.PlanSchemaType]internal.PlanSchema)

	if p.SchemasUpdate != nil {
		mappedSchemas[internal.SchemaTypeUpdate] = *p.SchemasUpdate
	}
	if p.SchemasCreate != nil {
		mappedSchemas[internal.SchemaTypeProvision] = *p.SchemasCreate
	}
	if p.SchemasBind != nil {
		mappedSchemas[internal.SchemaTypeBind] = *p.SchemasBind
	}

	return internal.AddonPlan{
		ID:          internal.AddonPlanID(p.Meta.ID),
		Name:        internal.AddonPlanName(p.Meta.Name),
		Description: p.Meta.Description,
		Metadata: internal.AddonPlanMetadata{
			DisplayName: p.Meta.DisplayName,
		},
		ChartValues:  internal.ChartValues(p.Values),
		Schemas:      mappedSchemas,
		ChartRef:     cRef,
		Bindable:     p.Meta.Bindable,
		BindTemplate: p.BindTemplate,
		Free:         p.Meta.Free,
	}, nil
}

type formPlanMeta struct {
	ID          string `yaml:"id"`
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
	DisplayName string `yaml:"displayName"`
	Bindable    *bool  `yaml:"bindable"`
	Free        *bool  `yaml:"free"`
}

func (f *formPlanMeta) Validate() error {
	var messages []string
	if f.ID == "" {
		messages = append(messages, "missing ID field")
	}
	if f.Name == "" {
		messages = append(messages, "missing Name field")
	}
	if f.Description == "" {
		messages = append(messages, "missing Description field")
	}
	if f.DisplayName == "" {
		messages = append(messages, "missing displayName field")
	}
	if len(messages) > 0 {
		return errors.New(strings.Join(messages, ", "))
	}
	return nil
}
