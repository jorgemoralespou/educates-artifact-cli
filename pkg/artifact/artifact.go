package artifact

import "context"

type Artifact interface {
	Push(ctx context.Context) error
	Pull(ctx context.Context) error
}
