package recipes

import (
	"context"
	"time"

	hexcore "github.com/2chevskii/go-hexcore"
)

// #region list-servers
func listServers(parent context.Context, token string) ([]hexcore.VDS, error) {
	client, err := hexcore.NewClient(hexcore.Config{Token: token})
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(parent, 30*time.Second)
	defer cancel()
	return client.VDS.List(ctx)
}

// #endregion list-servers
