// Copyright 2026 PolitePixels Limited
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

// This project stands against fascism, authoritarianism, and all forms of
// oppression. We built this to empower people, not to enable those who would
// strip others of their rights and dignity.

package config_resolver_kubernetes

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"piko.sh/piko/wdk/config"
)

var _ config.Resolver = (*Resolver)(nil)

// Resolver fetches secrets from the Kubernetes API server.
// It implements the config.Resolver interface.
//
// Circuit breaker protection is provided by the config Loader layer,
// not by this resolver directly.
//
// AUTHENTICATION:
// The resolver automatically handles both in-cluster and out-of-cluster
// authentication. When running inside a pod with a service account, it
// uses the service account token. When running locally, it uses the
// standard kubeconfig file (~/.kube/config or the path in KUBECONFIG
// env var).
//
// USAGE FORMAT:
// The value is expected in the format:
// "kubernetes-secret:[namespace/]secret-name#key"
// 1. With namespace: "kubernetes-secret:my-app/api-credentials#api-key"
// 2. Default namespace: "kubernetes-secret:db-credentials#password"
// uses "default" namespace.
type Resolver struct {
	// clientset is the Kubernetes API client used to fetch secrets.
	clientset *kubernetes.Clientset

	// namespace is the default Kubernetes namespace for secret lookups.
	namespace string
}

// NewResolver creates and sets up a new Kubernetes secret resolver.
//
// Returns *Resolver which is ready to access secrets in the detected
// namespace.
// Returns error when the Kubernetes configuration cannot be loaded or the
// client cannot be created.
func NewResolver() (*Resolver, error) {
	restConfig, err := rest.InClusterConfig()
	if err != nil {
		kubeconfig := os.Getenv("KUBECONFIG")
		if kubeconfig == "" {
			home, err := os.UserHomeDir()
			if err != nil {
				return nil, fmt.Errorf("cannot find user home directory for kubeconfig: %w", err)
			}
			kubeconfig = filepath.Join(home, ".kube", "config")
		}
		restConfig, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err != nil {
			return nil, fmt.Errorf("failed to load kubernetes config (in-cluster and kubeconfig failed): %w", err)
		}
	}

	namespace := "default"
	if restConfig.Host != "" {
		nsBytes, err := os.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/namespace")
		if err == nil {
			namespace = string(nsBytes)
		}
	}

	clientset, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create kubernetes clientset: %w", err)
	}

	return &Resolver{
		clientset: clientset,
		namespace: namespace,
	}, nil
}

// GetPrefix returns the prefix this resolver handles.
//
// Returns string which is the "kubernetes-secret:" prefix.
func (*Resolver) GetPrefix() string {
	return "kubernetes-secret:"
}

// Resolve fetches the secret value from the Kubernetes API.
//
// Takes value (string) which specifies the secret reference in the format
// "namespace/secret#key" or "secret#key" (uses the default namespace).
//
// Returns string which contains the decoded secret data.
// Returns error when the format is invalid, the secret is not found, or the
// key does not exist in the secret.
func (r *Resolver) Resolve(ctx context.Context, value string) (string, error) {
	secretRef, dataKey, ok := strings.Cut(value, "#")
	if !ok || dataKey == "" {
		return "", fmt.Errorf("invalid kubernetes secret format: %q; must include a data key after '#'", value)
	}

	namespace := r.namespace
	secretName := secretRef
	if parts := strings.SplitN(secretRef, "/", 2); len(parts) == 2 {
		namespace = parts[0]
		secretName = parts[1]
	}

	secret, err := r.clientset.CoreV1().Secrets(namespace).Get(ctx, secretName, metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			return "", fmt.Errorf("secret %q in namespace %q not found", secretName, namespace)
		}
		return "", fmt.Errorf("failed to get secret %q from kubernetes: %w", secretRef, err)
	}
	secretBytes, exists := secret.Data[dataKey]
	if !exists {
		return "", fmt.Errorf("key %q not found in kubernetes secret %q", dataKey, secretRef)
	}
	secretValue := string(secretBytes)

	return secretValue, nil
}

// Register creates a new Kubernetes secret resolver and registers it in the
// global resolver registry. This is a convenience function equivalent to
// [NewResolver] followed by [config.RegisterResolver].
//
// Returns error when resolver creation or registration fails.
//
// Example:
//
//	func init() {
//	    if err := config_resolver_kubernetes.Register(); err != nil {
//	        log.Fatal(err)
//	    }
//	}
func Register() error {
	resolver, err := NewResolver()
	if err != nil {
		return fmt.Errorf("creating Kubernetes resolver: %w", err)
	}
	return config.RegisterResolver(resolver)
}
