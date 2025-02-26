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
	"os"

	"github.com/couchbase/gocb/v2"
	tlsutil "github.com/couchbase/tools-common/http/tls"
	"github.com/googleapis/genai-toolbox/internal/sources"
	"go.opentelemetry.io/otel/trace"
)

const SourceKind string = "couchbase"

// validate interface
var _ sources.SourceConfig = Config{}

type Config struct {
	Name               string `yaml:"name" validate:"required"`
	Kind               string `yaml:"kind" validate:"required"`
	ConnectionString   string `yaml:"connection_string" validate:"required"`
	Bucket             string `yaml:"bucket" validate:"required"`
	Scope              string `yaml:"scope" validate:"required"`
	Username           string `yaml:"username"`
	Password           string `yaml:"password"`
	ClientCert         string `yaml:"client_cert"`
	ClientCertPassword string `yaml:"client_cert_password"`
	ClientKey          string `yaml:"client_key"`
	ClientKeyPassword  string `yaml:"client_key_password"`
	CACert             string `yaml:"ca_cert"`
	NoSSLVerify        bool   `yaml:"no_ssl_verify"`
	Profile            string `yaml:"profile"`
}

func (r Config) SourceConfigKind() string {
	return SourceKind
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
		Name:  r.Name,
		Kind:  SourceKind,
		Scope: scope,
	}
	return s, nil
}

var _ sources.Source = &Source{}

type Source struct {
	Name  string `yaml:"name"`
	Kind  string `yaml:"kind"`
	Scope *gocb.Scope
}

func (s *Source) SourceKind() string {
	return SourceKind
}

func (s *Source) CouchbaseScope() *gocb.Scope {
	return s.Scope
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
