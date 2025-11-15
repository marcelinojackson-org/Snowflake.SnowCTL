package runtime

import (
	"context"
	"fmt"
	"strings"

	"github.com/Snowflake-Labs/Snowflake.SnowCTL/pkg/config"
)

// Runtime captures process-wide configuration shared across commands.
type Runtime struct {
	Config            *config.Config
	ActiveContext     *config.Context
	ActiveContextName string
	OutputFormat      string
}

type runtimeKey struct{}

// NewRuntime loads CLI configuration, selects the active context, and validates output flags.
func NewRuntime(contextOverride, output string) (*Runtime, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, err
	}

    normalizedOutput := strings.ToLower(output)
    if normalizedOutput == "" {
        normalizedOutput = "json"
    }
    if normalizedOutput != "json" {
        return nil, fmt.Errorf("output format %q not supported; use json", output)
    }

	ctxName := contextOverride
	if ctxName == "" {
		ctxName = cfg.CurrentContext
	}
	if ctxName == "" {
		ctxName = cfg.DefaultContext
	}

	var active *config.Context
	if ctxName != "" {
		if c, ok := cfg.Contexts[ctxName]; ok {
			c.Name = ctxName
			active = c
		}
	}

	return &Runtime{
		Config:            cfg,
		ActiveContext:     active,
		ActiveContextName: ctxName,
		OutputFormat:      normalizedOutput,
	}, nil
}

// WithRuntime attaches runtime metadata to a context.
func WithRuntime(ctx context.Context, rt *Runtime) context.Context {
	return context.WithValue(ctx, runtimeKey{}, rt)
}

// FromContext unwraps the runtime from a context.
func FromContext(ctx context.Context) (*Runtime, bool) {
	if ctx == nil {
		return nil, false
	}
	rt, ok := ctx.Value(runtimeKey{}).(*Runtime)
	return rt, ok
}

// RequireRuntime fetches the runtime or returns an error if it has not been initialized.
func RequireRuntime(ctx context.Context) (*Runtime, error) {
	rt, ok := FromContext(ctx)
	if !ok {
		return nil, fmt.Errorf("runtime not initialized")
	}
	return rt, nil
}

// RequireActiveContext ensures a context is available for commands that need one.
func RequireActiveContext(ctx context.Context) (*config.Context, error) {
	rt, err := RequireRuntime(ctx)
	if err != nil {
		return nil, err
	}
	if rt.ActiveContext == nil {
		return nil, fmt.Errorf("no active connection configured. Configure one via 'snowctl connection set' and 'snowctl connection use'")
	}
	return rt.ActiveContext, nil
}
