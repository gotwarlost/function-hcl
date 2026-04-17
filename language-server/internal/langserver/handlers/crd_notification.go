package handlers

import (
	"context"
	"sync"

	"github.com/crossplane-contrib/function-hcl/language-server/internal/eventbus"
	lsp "github.com/crossplane-contrib/function-hcl/language-server/internal/langserver/protocol"
)

const crdSetupURL = "https://github.com/crossplane-contrib/function-hcl/language-server/blob/main/README.md"

func (svc *service) startCRDNotificationHandler(ctx context.Context) {
	events := svc.eventBus.SubscribeToNoCRDSourcesEvents("handlers.crd_notification")
	var notifiedDirs sync.Map
	go func() {
		for {
			select {
			case event := <-events:
				svc.handleNoCRDSources(ctx, event, &notifiedDirs)
			case <-ctx.Done():
				return
			}
		}
	}()
}

func (svc *service) handleNoCRDSources(ctx context.Context, event eventbus.NoCRDSourcesEvent, notifiedDirs *sync.Map) {
	// Only notify once per directory per session
	if _, alreadyNotified := notifiedDirs.LoadOrStore(event.Dir, true); alreadyNotified {
		return
	}
	params := &lsp.ShowMessageRequestParams{
		Type:    lsp.Info,
		Message: "No CRD sources configured for this workspace. Configure CRDs for full completion and validation support.",
		Actions: []lsp.MessageActionItem{
			{Title: "Learn More"},
		},
	}
	resp, err := svc.server.Callback(ctx, "window/showMessageRequest", params)
	if err != nil {
		svc.logger.Printf("failed to send CRD notification request: %s", err)
		return
	}
	// check if user clicked the action button
	var action *lsp.MessageActionItem
	if err := resp.UnmarshalResult(&action); err != nil {
		svc.logger.Printf("failed to unmarshal showMessageRequest response: %s", err)
		return
	}
	// user dismissed the notification or clicked outside
	if action == nil {
		return
	}
	// user clicked "Learn More" - open the URL
	if action.Title == "Learn More" {
		resp, err := svc.server.Callback(ctx, "window/showDocument", &lsp.ShowDocumentParams{
			URI:      crdSetupURL,
			External: true,
		})
		if err != nil {
			svc.logger.Printf("failed to open CRD setup URL: %s", err)
			return
		}
		var result lsp.ShowDocumentResult
		if err := resp.UnmarshalResult(&result); err != nil {
			svc.logger.Printf("failed to unmarshal showDocument response: %s", err)
			return
		}
		if !result.Success {
			svc.logger.Printf("showDocument reported failure for URL: %s", crdSetupURL)
		}
	}
}
