// Code generated by go-swagger; DO NOT EDIT.

package papersappsdk

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"context"

	"github.com/go-openapi/strfmt"
	"github.com/go-openapi/swag"
)

// List List Schema
//
// swagger:model List
type List struct {

	// Indicates whether the list was deleted or not
	Deleted bool `json:"deleted,omitempty"`

	// List id (e.g. 61a19540-77b8-46b2-9b82-5ca8a3af88d7)
	ID string `json:"id,omitempty"`

	// List modification timestamp (e.g. 2020-04-29T16:12:17Z)
	Modified string `json:"modified,omitempty"`

	// List name (e.g. Shark Folder)
	Name string `json:"name,omitempty"`

	// List parent id
	ParentID string `json:"parent_id,omitempty"`
}

// Validate validates this list
func (m *List) Validate(formats strfmt.Registry) error {
	return nil
}

// ContextValidate validates this list based on context it is used
func (m *List) ContextValidate(ctx context.Context, formats strfmt.Registry) error {
	return nil
}

// MarshalBinary interface implementation
func (m *List) MarshalBinary() ([]byte, error) {
	if m == nil {
		return nil, nil
	}
	return swag.WriteJSON(m)
}

// UnmarshalBinary interface implementation
func (m *List) UnmarshalBinary(b []byte) error {
	var res List
	if err := swag.ReadJSON(b, &res); err != nil {
		return err
	}
	*m = res
	return nil
}
