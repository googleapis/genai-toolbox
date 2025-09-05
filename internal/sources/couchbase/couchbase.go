// Copyright 2024 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package couchbase

import (
	"context"
	"crypto/tls"
	"fmt"
	"os"

	"github.com/couchbase/gocb/v2"
	tlsutil "github.com/couchbase/tools-common/http/tls"
	"github.com/goccy/go-yaml"
	"github.com/googleapis/genai-toolbox/internal/sources"
	"go.opentelemetry.io/otel/trace"
)

const SourceType string = "couchbase"

// validate interface
var _ sources.SourceConfig = Config{}

func init() {
	if !sources.Register(SourceType, newConfig) {
		panic(fmt.Sprintf("source type %q already registered", SourceType))
	}
}

func newConfig(ctx context.Context, name string, decoder *yaml.Decoder) (sources.SourceConfig, error) {
	actual := Config{Name: name}
	if err := decoder.DecodeContext(ctx, &actual); err != nil {
		return nil, err
	}
	return actual, nil
}

type Config struct {
	Name                 string `yaml:"name" validate:"required"`
	Type                 string `yaml:"kind" validate:"required"`
	ConnectionString     string `yaml:"connectionString" validate:"required"`
	Bucket               string `yaml:"bucket" validate:"required"`
	Scope                string `yaml:"scope" validate:"required"`
	Username             string `yaml:"username"`
	Password             string `yaml:"password"`
	ClientCert           string `yaml:"clientCert"`
	ClientCertPassword   string `yaml:"clientCertPassword"`
	ClientKey            string `yaml:"clientKey"`
	ClientKeyPassword    string `yaml:"clientKeyPassword"`
	CACert               string `yaml:"caCert"`
	NoSSLVerify          bool   `yaml:"noSslVerify"`
	Profile              string `yaml:"profile"`
	QueryScanConsistency uint   `yaml:"queryScanConsistency"`
}

func (r Config) SourceConfigType() string {
	return SourceType
}

func (r Config) Initialize(ctx context.Context, tracer trace.Tracer) (sources.Source, error) {

	opts, err := r.createCouchbaseOptions()
	if err != nil {
		return nil, err
	}
	cluster, err := gocb.Connect(r.ConnectionString, opts)
	if err != nil {
		return nil, err
	}

	scope := cluster.Bucket(r.Bucket).Scope(r.Scope)
	s := &Source{
		Name:                 r.Name,
		Type:                 SourceType,
		QueryScanConsistency: r.QueryScanConsistency,
		Scope:                scope,
	}
	return s, nil
}

var _ sources.Source = &Source{}

type Source struct {
	Name                 string `yaml:"name"`
	Type                 string `yaml:"type"`
	QueryScanConsistency uint   `yaml:"queryScanConsistency"`
	Scope                *gocb.Scope
}

func (s *Source) SourceType() string {
	return SourceType
}

func (s *Source) CouchbaseScope() *gocb.Scope {
	return s.Scope
}

func (s *Source) CouchbaseQueryScanConsistency() uint {
	return s.QueryScanConsistency
}

func (r Config) createCouchbaseOptions() (gocb.ClusterOptions, error) {
	cbOpts := gocb.ClusterOptions{}

	if r.Username != "" {
		auth := gocb.PasswordAuthenticator{
			Username: r.Username,
			Password: r.Password,
		}
		cbOpts.Authenticator = auth
	}

	var clientCert, clientKey, caCert []byte
	var err error
	if r.ClientCert != "" {
		clientCert, err = os.ReadFile(r.ClientCert)
		if err != nil {
			return gocb.ClusterOptions{}, err
		}
	}

	if r.ClientKey != "" {
		clientKey, err = os.ReadFile(r.ClientKey)
		if err != nil {
			return gocb.ClusterOptions{}, err
		}
	}
	if r.CACert != "" {
		caCert, err = os.ReadFile(r.CACert)
		if err != nil {
			return gocb.ClusterOptions{}, err
		}
	}
	if clientCert != nil || caCert != nil {
		// tls parsing code is similar to the code used in the cbimport.
		tlsConfig, err := tlsutil.NewConfig(tlsutil.ConfigOptions{
			ClientCert:     clientCert,
			ClientKey:      clientKey,
			Password:       []byte(getCertKeyPassword(r.ClientCertPassword, r.ClientKeyPassword)),
			ClientAuthType: tls.VerifyClientCertIfGiven,
			RootCAs:        caCert,
			NoSSLVerify:    r.NoSSLVerify,
		})
		if err != nil {
			return gocb.ClusterOptions{}, err
		}

		if r.ClientCert != "" {
			auth := gocb.CertificateAuthenticator{
				ClientCertificate: &tlsConfig.Certificates[0],
			}
			cbOpts.Authenticator = auth
		}
		if r.CACert != "" {
			cbOpts.SecurityConfig = gocb.SecurityConfig{
				TLSSkipVerify: r.NoSSLVerify,
				TLSRootCAs:    tlsConfig.RootCAs,
			}
		}
		if r.NoSSLVerify {
			cbOpts.SecurityConfig = gocb.SecurityConfig{
				TLSSkipVerify: r.NoSSLVerify,
			}
		}
	}
	if r.Profile != "" {
		err = cbOpts.ApplyProfile(gocb.ClusterConfigProfile(r.Profile))
		if err != nil {
			return gocb.ClusterOptions{}, err
		}
	}
	return cbOpts, nil
}

// GetCertKeyPassword - Returns the password which should be used when creating a new TLS config.
func getCertKeyPassword(certPassword, keyPassword string) string {
	if keyPassword != "" {
		return keyPassword
	}

	return certPassword
}
