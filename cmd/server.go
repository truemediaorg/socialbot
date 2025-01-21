package cmd

import (
	"context"
	"encoding/json"
	"net/http"
	"os/signal"
	"syscall"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/truemediaorg/socialbot/config"
	"github.com/truemediaorg/socialbot/database"
	"github.com/truemediaorg/socialbot/responder"
	"github.com/truemediaorg/socialbot/service"
	"github.com/truemediaorg/socialbot/watcher"
	"golang.org/x/sync/errgroup"
)

func init() {
	rootCmd.AddCommand(serverCmd)
}

var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "Runs the socialbot server",
	Long:  `Runs the socialbot server`,
	Run: func(cmd *cobra.Command, args []string) {

		cfg := config.FromEnvfile()

		log.SetLevel(cfg.LogLevel)

		switch cfg.LogFormat {
		case config.LogFormatJSON:
			log.SetFormatter(&log.JSONFormatter{})
		default:
			log.SetFormatter(&log.TextFormatter{})
		}

		if cfg.TestModeEnabled {
			log.Info("TEST MODE ENABLED")
		}

		awsConfig, err := awsconfig.LoadDefaultConfig(context.Background())
		if err != nil {
			log.Fatal(err)
		}
		secretsManagerClient := secretsmanager.NewFromConfig(awsConfig)

		databaseURL := cfg.PostgresURL
		if databaseURL == "" {
			// Get the DB secrets from AWS Secrets Manager
			result, err := secretsManagerClient.GetSecretValue(context.Background(), &secretsmanager.GetSecretValueInput{SecretId: aws.String(cfg.PostgresSecretPath)})
			if err != nil {
				log.Fatal(err.Error())
			}
			var pgSecrets config.PostgresSecretData
			err = json.Unmarshal([]byte(*result.SecretString), &pgSecrets)
			if err != nil {
				log.Fatalf("postgres secrets read error: %v", err)
			}
			databaseURL = pgSecrets.ConnectionString
		}

		/*
			Graceful shutdown is possible with errgroup + signal.NotifyContext
			NotifyContext returns a context that will close on OS signals to terminate the process
			errgroup uses that context, and also closes it in case a goroutine errors out
		*/
		ctx, done := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM, syscall.SIGKILL)
		defer done()
		g, gCtx := errgroup.WithContext(ctx)

		twitterService := service.NewTwitterService(gCtx, cfg, secretsManagerClient)
		truemediaService := service.NewTruemediaService(cfg, secretsManagerClient)

		database := database.NewDatabase(databaseURL)
		if err = database.Connect(gCtx); err != nil {
			log.Fatalf("error connecting to database: %v", err)
		}
		defer database.Disconnect()

		watcher := watcher.NewWatcher(twitterService, truemediaService, database)

		responder := responder.NewResponder(twitterService, truemediaService, database, cfg.TestModeEnabled)

		healthchecker := service.NewHealthchecker(8080)

		g.Go(func() error {
			defer log.Info("exiting watcher")
			return watcher.Watch(gCtx)
		})

		g.Go(func() error {
			defer log.Info("exiting responder")
			return responder.Respond(gCtx)
		})

		// For deployed instances, provide a basic healthcheck endpoint to show it's online
		g.Go(func() error {
			if err := healthchecker.Server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				return err
			}
			return nil
		})
		// ...and shut down the server if the bot needs to terminate
		g.Go(func() error {
			<-gCtx.Done()
			defer log.Info("exiting healthchecker")
			return healthchecker.Server.Shutdown(context.Background())
		})

		err = g.Wait()
		if err != nil {
			log.Errorf("caught error: %v", err)
		}
	},
}
