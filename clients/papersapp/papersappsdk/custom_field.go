package papersappsdk

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

// CustomField custom field
//
// swagger:model CustomField
type CustomField struct {

	// display
	Display string `json:"display,omitempty"`

	// field
	Field string `json:"field,omitempty"`

	// show in details
	ShowInDetails bool `json:"show_in_details,omitempty"`

	// show in table
	ShowInTable bool `json:"show_in_table,omitempty"`

	// type
	Type string `json:"type,omitempty"`
}
