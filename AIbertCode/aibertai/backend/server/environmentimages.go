package server

import (
	"context"
	"io"

	"aibertai.local/backend/internal/envimage"
)

func EnvironmentImages(ctx context.Context, args []string, output io.Writer) error {
	return envimage.Command(ctx, args, output)
}
