package cmd

import "errors"

type ArtifactType string

const (
	ArtifactTypeOci      ArtifactType = "oci"
	ArtifactTypeImgpkg   ArtifactType = "imgpkg"
	ArtifactTypeEducates ArtifactType = "educates"
)

// String is used both by fmt.Print and by Cobra in help text
func (e *ArtifactType) String() string {
	return string(*e)
}

// Set must have pointer receiver so it doesn't change the value of a copy
func (e *ArtifactType) Set(v string) error {
	switch v {
	case "oci", "imgpkg", "educates":
		*e = ArtifactType(v)
		return nil
	default:
		return errors.New(`must be one of "oci", "imgpkg", or "educates"`)
	}
}

// Type is only used in help text
func (e *ArtifactType) Type() string {
	return "ArtifactType"
}
