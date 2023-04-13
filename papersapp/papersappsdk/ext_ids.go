package papersappsdk

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

// ExtIds ExtIds schema
//
// swagger:model ExtIds
type ExtIds struct {

	// arxiv
	Arxiv string `json:"arxiv,omitempty"`

	// Item DOI (e.g. 10.1002/jbmr.178)
	Doi string `json:"doi,omitempty"`

	// gsid
	Gsid string `json:"gsid,omitempty"`

	// Item patent ID (e.g. CH-708619-A1)
	PatentID string `json:"patent_id,omitempty"`

	// pmcid
	Pmcid string `json:"pmcid,omitempty"`

	// Item PMID (e.g. 20614475)
	Pmid string `json:"pmid,omitempty"`
}
