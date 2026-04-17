package crds

import (
	"context"

	lsp "github.com/crossplane-contrib/function-hcl/language-server/internal/langserver/protocol"
)

func (c *CRDs) start(ctx context.Context) {
	c.store.Start(ctx)
	open := c.bus.SubscribeToOpenEvents("feature.crds")
	changeWatch := c.bus.SubscribeToChangeWatchEvents("feature.crds")
	go func() {
		for {
			var err error
			select {
			case event := <-open:
				err = c.onOpen(event.Doc.Dir.Path())
			case event := <-changeWatch:
				err = c.onChangeWatch(event.ChangeType, event.RawPath, event.IsDir)
			case <-ctx.Done():
				c.logger.Print("stopped crds feature")
				return
			}
			if err != nil {
				c.logger.Printf("crds: process event: %q", err)
			}
		}
	}()
}

func (c *CRDs) onOpen(path string) error {
	c.store.RegisterOpenDir(path)
	return nil
}

func (c *CRDs) onChangeWatch(changeType lsp.FileChangeType, path string, isDir bool) error {
	c.logger.Printf("change watch event: type=%d path=%s isDir=%t", changeType, path, isDir)
	switch {
	case changeType == lsp.Deleted:
		c.store.ProcessPathDeletion(path)
	case isDir && changeType == lsp.Created: // no need to handle dir modification events, since some file would have changed
		c.store.ProcessNewDir(path)
	default:
		c.store.ProcessFile(path)
	}
	return nil
}
