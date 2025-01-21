package service

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/truemediaorg/socialbot/config"
	"github.com/truemediaorg/socialbot/truemedia"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	log "github.com/sirupsen/logrus"
)

type TruemediaService struct {
	config config.TruemediaConfig
	client *truemedia.Client
}

func NewTruemediaService(cfg config.Config, secretsManagerClient *secretsmanager.Client) *TruemediaService {
	// Get the TrueMedia secrets from AWS Secrets Manager
	result, err := secretsManagerClient.GetSecretValue(
		context.Background(),
		&secretsmanager.GetSecretValueInput{
			SecretId: aws.String(cfg.Truemedia.SecretPath),
		},
	)
	if err != nil {
		log.Fatal(err.Error())
	}
	var trueMediaSecrets config.TrueMediaSecretData
	err = json.Unmarshal([]byte(*result.SecretString), &trueMediaSecrets)
	if err != nil {
		log.Panicf("truemedia secrets read error: %v", err)
	}

	client := truemedia.NewClient(trueMediaSecrets.ApiKey, cfg.Truemedia.ApiURL)
	log.Infof("TrueMedia client initialized. Host: %s", cfg.Truemedia.ApiURL.String())

	return &TruemediaService{
		config: cfg.Truemedia,
		client: client,
	}
}

func (s *TruemediaService) ResolvePostMedia(postURL string) (string, error) {
	resolve, err := s.client.ResolveMedia(postURL)
	if err != nil {
		return "", err
	}
	if len(resolve.Media) == 0 {
		// TODO: Consider adding pkg/errors
		return "", errors.New("no media resolved")
	}
	return resolve.Media[0].ID, nil
}

func (s *TruemediaService) GetAnalysis(mediaID string) (*truemedia.GetResultResponse, error) {
	return s.client.GetResults(mediaID)
}

func (s *TruemediaService) ResolveInterval() time.Duration {
	return s.config.ResolveInterval
}
