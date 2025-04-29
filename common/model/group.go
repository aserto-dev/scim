// Do not edit. This file is auto-generated.
package model

// Group
type Group struct {
	DisplayName string        `json:"displayName,omitempty"`
	ExternalID  string        `json:"externalId,omitempty"`
	ID          string        `json:"id"`
	Members     []GroupMember `json:"members,omitempty"`
}

// A list of members of the Group.
type GroupMember struct {
	Value   string `json:"value"`
	Ref     string `json:"$ref,omitempty"`
	Type    string `json:"type,omitempty"`
	Display string `json:"display,omitempty"`
}
