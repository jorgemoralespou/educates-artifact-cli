package educates

import "fmt"

type EducatesImageArtifact struct {
	repoRef       string
	pushPlatforms []string
	pullPlatform  string
	path          string
}

func NewEducatesImageArtifact(repoRef string, pushPlatforms []string, pullPlatform string, path string) *EducatesImageArtifact {
	return &EducatesImageArtifact{repoRef: repoRef, pushPlatforms: pushPlatforms, pullPlatform: pullPlatform, path: path}
}

func (a *EducatesImageArtifact) Push() error {
	fmt.Println("Pushing educates artifact...")
	fmt.Println("Opps, not implemented yet")
	return nil
}

func (a *EducatesImageArtifact) Pull() error {
	fmt.Println("Pulling educates artifact...")
	fmt.Println("Opps, not implemented yet")
	return nil
}
