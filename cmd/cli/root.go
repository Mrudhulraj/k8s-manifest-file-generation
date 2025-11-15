package cli

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strconv"

	"github.com/janeczku/go-spinner"
	"github.com/manifoldco/promptui"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/walles/env"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

const (
	apply     = "Apply"
	dontApply = "Don't Apply"
	reprompt  = "Reprompt"
)

var (
	openaiURLv1           = "https://api.openai.com/v1"
	kubernetesConfigFlags = genericclioptions.NewConfigFlags(false)
	openAIDeploymentName  = flag.String("open-ai-deployment", env.GetOr("OPEN_AI_DEPLOYMENT", env.String, "gpt-3.5-turbo-0301"), "Open AI Deployment")
	openAIEndPoint        = flag.String("open-ai-endpoint", env.GetOr("OPEN_AI_ENDPOINT", env.String, openaiURLv1), "Default is "+openaiURLv1+" Mention the endpoint in the env file")
	version               = "dev"
	azureModelMap         = flag.StringToString("azure-model-map", env.GetOr("AZURE_MODEL_MAP", env.Map(env.String, "=", env.String, ""), map[string]string{}), "Azure Model Map being employed")
	openAIAPIKey          = flag.String("open-api-key", env.GetOr("OPENAI_API_KEY", env.String, ""), "Set the OPEN_API_KEY in the env file")
	debug                 = flag.Bool("debug", env.GetOr("DEBUG", strconv.ParseBool, false), "Whether to debug or not default to Loss")
	raw                   = flag.Bool("raw", false, "Print the raw YAML Output Immediately returned from GPT. Default value false")
	requireConfirmation   = flag.Bool("require-confirmation", env.GetOr("REQUIRE_CONFIRMATION", strconv.ParseBool, true), "Do we need user confirmation again. Default set to True")
	temperature           = flag.Float64("temperature", env.GetOr("TEMPERATURE", env.WithBitSize(strconv.ParseFloat, 64), 0.0), "Tempeature set to 0 (More deterministic) -> 1 (More random)")
	usek8sAPI             = flag.Bool("use-k8s-api", env.GetOr("USE_K8S_API", strconv.ParseBool, true), "Bool to determine if we have to use the K8S or not")
	k8sOpenAPIURL         = flag.String("k8s-openapi-url", env.GetOr("K8S_OPENAPI_URL", env.String, ""), "The URL of k8s api is got only if the usek8sAPI flag is set to true")
)

func InitAndExecute() {
	if *openAIAPIKey == "" {
		fmt.Println("Please provide OpenAI key")
		os.Exit(1)
	}
	if err := RootCmd().Execute(); err != nil {
		os.Exit(1)
	}

}

func RootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:          "k8-ai-assistant",
		Short:        "k8-ai-assistant",
		Long:         "k8-ai-assistant is a plugin for kubectl (command line tool for kubernetest) gives you the power of OpenAI API",
		Version:      version,
		SilenceUsage: true,
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			if *debug {
				log.SetLevel(log.DebugLevel)
				printDebugFlags()
			}
		},
		RunE: func(_ *cobra.Command, args []string) error {
			if len(args) == 0 {
				return fmt.Errorf("Prompt arguments must be provided")
			}

			err := run(args)
			if err != nil {
				return err
			}
			return nil
		},
	}
	kubernetesConfigFlags.AddFlags(cmd.PersistentFlags())
	return cmd
}

func printDebugFlags() {
	log.Debugf("openai-endpoint: %s", *openAIEndPoint)
	log.Debugf("openai-deployment-name: %s", *openAIDeploymentName)
	log.Debugf("azure-openai-map: %s", *azureModelMap)
	log.Debugf("temperature: %f", *temperature)
	log.Debugf("use-k8s-api: %t", *usek8sAPI)
	log.Debugf("k8s-openai-url: %s", *k8sOpenAPIURL)
}

func run(args []string) error {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()
	oaiClients, err := newOAIClients()
	if err != nil {
		return err
	}

	var action, completion string

	for action != apply {
		args = append(args, action)
		s := spinner.NewSpinner("Processing....")
		if !*debug && !*raw {
			s.SetCharset([]string{"⣾", "⣽", "⣻", "⢿", "⡿", "⣟", "⣯", "⣷"})
			s.Start()
		}

		completion, err = gptCompletion(ctx, oaiClients, args, *openAIDeploymentName)
		if err != nil {
			return err
		}

		s.Stop()
		if *raw {
			fmt.Println(completion)
			return nil
		}
		text := fmt.Sprintf("Attempting to apply the following manifest:\n %s", completion)
		fmt.Println(text)
		action, err = userActionPrompt()

		if action == dontApply {
			return nil
		}

	}
	return applyManifest(completion)
}

func userActionPrompt() (string, error) {
	if !*requireConfirmation {
		return apply, nil
	}
	var result string
	var err error
	items := []string{apply, dontApply}
	currentContext, err := getCurrentContextName()
	label := fmt.Sprintf("Would you like to apply this: [%[1]s/%[2]s/%[3]s]", reprompt, apply, dontApply)
	if err == nil {
		label = fmt.Sprintf("Context: %[1]s %[2]s", &currentContext, label)
	}
	prompt := promptui.SelectWithAdd{
		Label:    label,
		Items:    items,
		AddLabel: reprompt,
	}

	_, result, err = prompt.Run()
	if err != nil {
		return dontApply, err
	}
	return result, nil
}
