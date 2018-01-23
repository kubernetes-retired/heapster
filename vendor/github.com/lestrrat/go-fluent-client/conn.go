package fluent

import (
	"context"
	"net"
	"time"

	"github.com/pkg/errors"
)

func dial(ctx context.Context, network, address string, timeout time.Duration) (net.Conn, error) {
	connCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	var dialer net.Dialer
	conn, err := dialer.DialContext(connCtx, network, address)
	if err != nil {
		return nil, errors.Wrap(err, `failed to connect to server`)
	}

	return conn, nil
}
