package papersappsdk

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

// UserData UserData Schema
//
// swagger:model UserData
type UserData struct {

	// Hex color (e.g. "#fe3018")
	Color string `json:"color,omitempty"`

	// Item creation timestamp
	Created string `json:"created,omitempty"`

	// User note
	Notes string `json:"notes,omitempty"`

	// Item flag/star
	Star bool `json:"star,omitempty"`

	// Item hashtags/keywords (e.g. Shark)
	Tags []string `json:"tags"`
}
