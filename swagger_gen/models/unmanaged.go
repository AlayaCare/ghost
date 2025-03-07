// Code generated by go-swagger; DO NOT EDIT.

package models

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"context"
	"strconv"

	"github.com/go-openapi/errors"
	"github.com/go-openapi/strfmt"
	"github.com/go-openapi/swag"
	"github.com/go-openapi/validate"
)

// Unmanaged unmanaged
//
// swagger:model unmanaged
type Unmanaged struct {

	// externally managed teams
	ExternallyManagedTeams []string `json:"externally_managed_teams"`

	// repos
	Repos []string `json:"repos"`

	// rulesets
	Rulesets []string `json:"rulesets"`

	// teams
	Teams []string `json:"teams"`

	// users
	Users []string `json:"users"`
}

// Validate validates this unmanaged
func (m *Unmanaged) Validate(formats strfmt.Registry) error {
	var res []error

	if err := m.validateExternallyManagedTeams(formats); err != nil {
		res = append(res, err)
	}

	if err := m.validateRepos(formats); err != nil {
		res = append(res, err)
	}

	if err := m.validateRulesets(formats); err != nil {
		res = append(res, err)
	}

	if err := m.validateTeams(formats); err != nil {
		res = append(res, err)
	}

	if err := m.validateUsers(formats); err != nil {
		res = append(res, err)
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}

func (m *Unmanaged) validateExternallyManagedTeams(formats strfmt.Registry) error {
	if swag.IsZero(m.ExternallyManagedTeams) { // not required
		return nil
	}

	for i := 0; i < len(m.ExternallyManagedTeams); i++ {

		if err := validate.MinLength("externally_managed_teams"+"."+strconv.Itoa(i), "body", m.ExternallyManagedTeams[i], 1); err != nil {
			return err
		}

	}

	return nil
}

func (m *Unmanaged) validateRepos(formats strfmt.Registry) error {
	if swag.IsZero(m.Repos) { // not required
		return nil
	}

	for i := 0; i < len(m.Repos); i++ {

		if err := validate.MinLength("repos"+"."+strconv.Itoa(i), "body", m.Repos[i], 1); err != nil {
			return err
		}

	}

	return nil
}

func (m *Unmanaged) validateRulesets(formats strfmt.Registry) error {
	if swag.IsZero(m.Rulesets) { // not required
		return nil
	}

	for i := 0; i < len(m.Rulesets); i++ {

		if err := validate.MinLength("rulesets"+"."+strconv.Itoa(i), "body", m.Rulesets[i], 1); err != nil {
			return err
		}

	}

	return nil
}

func (m *Unmanaged) validateTeams(formats strfmt.Registry) error {
	if swag.IsZero(m.Teams) { // not required
		return nil
	}

	for i := 0; i < len(m.Teams); i++ {

		if err := validate.MinLength("teams"+"."+strconv.Itoa(i), "body", m.Teams[i], 1); err != nil {
			return err
		}

	}

	return nil
}

func (m *Unmanaged) validateUsers(formats strfmt.Registry) error {
	if swag.IsZero(m.Users) { // not required
		return nil
	}

	for i := 0; i < len(m.Users); i++ {

		if err := validate.MinLength("users"+"."+strconv.Itoa(i), "body", m.Users[i], 1); err != nil {
			return err
		}

	}

	return nil
}

// ContextValidate validates this unmanaged based on context it is used
func (m *Unmanaged) ContextValidate(ctx context.Context, formats strfmt.Registry) error {
	return nil
}

// MarshalBinary interface implementation
func (m *Unmanaged) MarshalBinary() ([]byte, error) {
	if m == nil {
		return nil, nil
	}
	return swag.WriteJSON(m)
}

// UnmarshalBinary interface implementation
func (m *Unmanaged) UnmarshalBinary(b []byte) error {
	var res Unmanaged
	if err := swag.ReadJSON(b, &res); err != nil {
		return err
	}
	*m = res
	return nil
}
