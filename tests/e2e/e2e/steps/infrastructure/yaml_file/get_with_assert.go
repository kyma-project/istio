package yaml_file

import "github.com/stretchr/testify/require"

type GetWithAssert struct {
	FilePathDesired string
	ResourceActual  string
}

func (g GetWithAssert) Description() string {
	//TODO implement me
	panic("implement me")
}

func (g GetWithAssert) Execute(ctx context.Context, client client.Client, logger *log.Logger) error {
	require.Equal(g.FilePathDesired, g.ResourceActual)
}

func (g GetWithAssert) Cleanup(ctx context.Context, client client.Client) error {
	//TODO implement me
	panic("implement me")
}
