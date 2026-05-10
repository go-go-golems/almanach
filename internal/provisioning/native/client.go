package native

import "context"

// Client is the high-level native ESP-IDF provisioning client. It owns protocol
// sequencing and delegates endpoint IO to Transport.
type Client struct {
	Transport Transport
}

func NewClient(t Transport) *Client {
	return &Client{Transport: t}
}

func (c *Client) VerifyVersion(ctx context.Context, want string) (*ProtoInfo, error) {
	return VerifyProtoVersion(ctx, c.Transport, want)
}
