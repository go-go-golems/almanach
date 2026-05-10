package native

import "context"

// Client is the high-level native ESP-IDF provisioning client. It owns protocol
// sequencing and delegates endpoint IO to Transport.
type Client struct {
	Transport Transport
	Security  *Security1Session
}

func NewClient(t Transport) *Client {
	return &Client{Transport: t}
}

func (c *Client) VerifyVersion(ctx context.Context, want string) (*ProtoInfo, error) {
	return VerifyProtoVersion(ctx, c.Transport, want)
}

func (c *Client) EstablishSecurity1(ctx context.Context, pop string) (*Security1Session, error) {
	session := NewSecurity1Session(pop)
	if err := session.Establish(ctx, c.Transport); err != nil {
		return nil, err
	}
	c.Security = session
	return session, nil
}
