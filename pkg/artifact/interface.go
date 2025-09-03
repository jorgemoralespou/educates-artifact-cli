package artifact

type Artifact interface {
	Push() error
	Pull() error
}
