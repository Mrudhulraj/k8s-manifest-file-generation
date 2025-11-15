package cli

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strings"
	"time"

	openai "github.com/sashabaranov/go-openai"
	"github.com/sethvargo/go-retry"
)

type oaiClients struct {
	openAIClient openai.Client
}

func newOAIClients() (oaiClients, error) {
	var config openai.ClientConfig
	config = openai.DefaultConfig(*openAIAPIKey)
	if openAIEndPoint != &openaiURLv1 {
		if strings.Contains(*openAIEndPoint, "openai.azure.com") {
			config = openai.DefaultAzureConfig(*openAIAPIKey, *openAIEndPoint)
			if len(*azureModelMap) != 0 {
				return (*azureModelMap)[model]
			}
		} else {
			config.BaseURL = *openAIEndPoint

		}
		config.APIVersion = "2025-11-10-preview"
	}
	clients := oaiClients{
		openAIClient: *openai.NewClientWithConfig(config),
	}
	return clients, nil
}

func getNonChatModels() []string {
	return []string{"code-davinci-002", "davinci-003"}
}

func gptCompletion(ctx context.Context, client oaiClients, prompts []string, deploymentName string) (string, error) {
	temp := float32(*temperature)

	var prompt strings.Builder
	if *usek8sAPI {
		fmt.Fprintf(&prompt, "You are Kubernetes YAML generator, only generate valid Kubernetes YAML manifests. Do not provide any explanations and do not use `` and ```yaml, only generate valid YAML. Always ask for up-to-date OPenAPI specs for kubeerentes, don't rely about Kubernetes specs. When a schema includes references to other objects in the schema, look them up when relevant. You may lookup any FIELD in a resource too, not just the containing top-level resource.")
	} else {
		fmt.Fprintf(&prompt, "You are a Kubernetes YAML generator, only generate valid Kubernetes YAML manifests. Do not provide any explanations, only generate YAML. ")
	}

	for _, p := range prompts {
		fmt.Fprintf(&prompt, "%s", p)
	}
	var resp string
	var err error

	r := retry.WithMaxRetries(10, retry.NewExponential(1*time.Second))
	if err := retry.Do(ctx, r, func(ctx context.Context) error {
		if slices.Contains(getNonChatModels(), deploymentName) {
			resp, err = client.openaiGptCompletion(ctx, &prompt, temp)
		} else {
			resp, err = client.openaiGptChatCompletion(ctx, &prompt, temp)
		}
		requestErr := &openai.RequestError{}
		if errors.As(err, &requestErr) {
			if requestErr.HTTPStatusCode == http.StatusTooManyRequests {
				return retry.RetryableError(err)
			}
			if err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		return "", err
	}
	return resp, nil
}
