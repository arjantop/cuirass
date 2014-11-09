package requestlog_test

import (
	"testing"

	"code.google.com/p/go.net/context"
	"github.com/arjantop/cuirass/requestlog"
	"github.com/stretchr/testify/assert"
)

func TestFromContext(t *testing.T) {
	ctx := requestlog.WithRequestLog(context.Background())
	requestlog.FromContext(ctx).AddRequest(nil)
	assert.Equal(t, 1, requestlog.FromContext(ctx).Size())
}
