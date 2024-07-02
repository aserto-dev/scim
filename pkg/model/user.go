// Do not edit. This file is auto-generated.
package model

// User Account
type User struct {
	Active            bool                  `json:"active,omitempty"`
	Addresses         []UserAddress         `json:"addresses,omitempty"`
	DisplayName       string                `json:"displayName,omitempty"`
	Emails            []UserEmail           `json:"emails,omitempty"`
	Entitlements      []UserEntitlement     `json:"entitlements,omitempty"`
	ExternalID        string                `json:"externalId,omitempty"`
	Groups            []UserGroup           `json:"groups,omitempty"`
	ID                string                `json:"id"`
	Ims               []UserIm              `json:"ims,omitempty"`
	Locale            string                `json:"locale,omitempty"`
	Name              UserName              `json:"name,omitempty"`
	NickName          string                `json:"nickName,omitempty"`
	Password          string                `json:"password,omitempty"`
	PhoneNumbers      []UserPhoneNumber     `json:"phoneNumbers,omitempty"`
	Photos            []UserPhoto           `json:"photos,omitempty"`
	PreferredLanguage string                `json:"preferredLanguage,omitempty"`
	ProfileUrl        string                `json:"profileUrl,omitempty"`
	Roles             []UserRole            `json:"roles,omitempty"`
	Timezone          string                `json:"timezone,omitempty"`
	Title             string                `json:"title,omitempty"`
	UserName          string                `json:"userName"`
	UserType          string                `json:"userType,omitempty"`
	X509Certificates  []UserX509Certificate `json:"x509Certificates,omitempty"`

	EnterpriseUser EnterpriseUserExtension `json:"urn:ietf:params:scim:schemas:extension:enterprise:2.0:User,omitempty"`
}

// A physical mailing address for this User. Canonical type values of 'work', 'home', and 'other'. This attribute is a
// type with the following sub-attributes.
type UserAddress struct {
	Formatted     string `json:"formatted,omitempty"`
	StreetAddress string `json:"streetAddress,omitempty"`
	Locality      string `json:"locality,omitempty"`
	Region        string `json:"region,omitempty"`
	PostalCode    string `json:"postalCode,omitempty"`
	Country       string `json:"country,omitempty"`
	Type          string `json:"type,omitempty"`
}

// Email addresses for the user. The value SHOULD be canonicalized by the service provider, e.g., 'bjensen@example.com'
// of 'bjensen@EXAMPLE.COM'. Canonical type values of 'work', 'home', and 'other'.
type UserEmail struct {
	Value   string `json:"value"`
	Display string `json:"display,omitempty"`
	Type    string `json:"type"`
	Primary bool   `json:"primary,omitempty"`
}

// A list of entitlements for the User that represent a thing the User has.
type UserEntitlement struct {
	Value   string `json:"value"`
	Display string `json:"display,omitempty"`
	Type    string `json:"type"`
	Primary bool   `json:"primary,omitempty"`
}

// A list of groups to which the user belongs, either through direct membership, through nested groups, or dynamically
type UserGroup struct {
	Value   string `json:"value"`
	Ref     string `json:"$ref"`
	Display string `json:"display,omitempty"`
	Type    string `json:"type"`
}

// Instant messaging addresses for the User.
type UserIm struct {
	Value   string `json:"value"`
	Display string `json:"display,omitempty"`
	Type    string `json:"type"`
	Primary bool   `json:"primary,omitempty"`
}

// The components of the user's real name. Providers MAY return just the full name as a single string in the formatted
// or they MAY return just the individual component attributes using the other sub-attributes, or they MAY return both.
// both variants are returned, they SHOULD be describing the same name, with the formatted name indicating how the
// attributes should be combined.
type UserName struct {
	Formatted       string `json:"formatted,omitempty"`
	FamilyName      string `json:"familyName,omitempty"`
	GivenName       string `json:"givenName,omitempty"`
	MiddleName      string `json:"middleName,omitempty"`
	HonorificPrefix string `json:"honorificPrefix,omitempty"`
	HonorificSuffix string `json:"honorificSuffix,omitempty"`
}

// Phone numbers for the User. The value SHOULD be canonicalized by the service provider according to the format
// in RFC 3966, e.g., 'tel:+1-201-555-0123'. Canonical type values of 'work', 'home', 'mobile', 'fax', 'pager', and
type UserPhoneNumber struct {
	Value   string `json:"value"`
	Display string `json:"display,omitempty"`
	Type    string `json:"type"`
	Primary bool   `json:"primary,omitempty"`
}

// URLs of photos of the User.
type UserPhoto struct {
	Value   string `json:"value"`
	Display string `json:"display,omitempty"`
	Type    string `json:"type"`
	Primary bool   `json:"primary,omitempty"`
}

// A list of roles for the User that collectively represent who the User is, e.g., 'Student', 'Faculty'.
type UserRole struct {
	Value   string `json:"value"`
	Display string `json:"display,omitempty"`
	Type    string `json:"type"`
	Primary bool   `json:"primary,omitempty"`
}

// A list of certificates issued to the User.
type UserX509Certificate struct {
	Value   string `json:"value"`
	Display string `json:"display,omitempty"`
	Type    string `json:"type"`
	Primary bool   `json:"primary,omitempty"`
}

// Enterprise User
type EnterpriseUserExtension struct {
	CostCenter     string                         `json:"costCenter,omitempty"`
	Department     string                         `json:"department,omitempty"`
	Division       string                         `json:"division,omitempty"`
	EmployeeNumber string                         `json:"employeeNumber,omitempty"`
	Manager        EnterpriseUserExtensionManager `json:"manager,omitempty"`
	Organization   string                         `json:"organization"`
}

// The User's manager. A complex type that optionally allows service providers to represent organizational hierarchy by
// the 'id' attribute of another User.
type EnterpriseUserExtensionManager struct {
	Value       string `json:"value"`
	Ref         string `json:"$ref"`
	DisplayName string `json:"displayName,,omitempty"`
}
