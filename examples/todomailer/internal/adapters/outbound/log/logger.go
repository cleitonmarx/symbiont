package log

import (
	"context"
	"log"
	"os"

	"github.com/cleitonmarx/symbiont/depend"
)

type InitLogger struct{}

func (il *InitLogger) Initialize(ctx context.Context) (context.Context, error) {
	depend.Register(log.New(os.Stdout, "", log.Lmsgprefix))
	return ctx, nil
}
