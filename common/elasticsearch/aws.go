package elasticsearch

import (
	"fmt"
	awsauth "github.com/smartystreets/go-aws-auth"
	"net/http"
	"os"
)

type AWSSigningTransport struct {
	HTTPClient  *http.Client
	Credentials awsauth.Credentials
}

// RoundTrip implementation
func (a AWSSigningTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return a.HTTPClient.Do(awsauth.Sign4(req, a.Credentials))
}

func createAWSClient() (*http.Client, error) {
	id := os.Getenv("AWS_ACCESS_KEY_ID")
	if id == "" {
		id = os.Getenv("AWS_ACCESS_KEY")
	}

	secret := os.Getenv("AWS_SECRET_ACCESS_KEY")
	if secret == "" {
		secret = os.Getenv("AWS_SECRET_KEY")
	}

	if id == "" || secret == "" {
		return nil, fmt.Errorf("Failed to configure AWS authentication. Both `AWS_ACCESS_KEY_ID` and " +
			"`AWS_SECRET_ACCESS_KEY` environment veriables required")
	}

	signingTransport := AWSSigningTransport{
		Credentials: awsauth.Credentials{
			AccessKeyID:     id,
			SecretAccessKey: secret,
		},
		HTTPClient: http.DefaultClient,
	}
	return &http.Client{Transport: http.RoundTripper(signingTransport)}, nil
}
