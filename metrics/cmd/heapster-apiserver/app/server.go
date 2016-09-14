/*
Copyright 2014 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Package app does all of the work necessary to create a Heapster
// APIServer by binding together the Master Metrics API.
// It can be configured and called directly or via the hyperkube framework.
package app

import (
	"strings"
	"time"

	"github.com/golang/glog"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"k8s.io/kubernetes/pkg/admission"
	"k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/api/unversioned"
	"k8s.io/kubernetes/pkg/apiserver/authenticator"
	"k8s.io/kubernetes/pkg/genericapiserver"
	genericauthorizer "k8s.io/kubernetes/pkg/genericapiserver/authorizer"
	genericoptions "k8s.io/kubernetes/pkg/genericapiserver/options"
	genericvalidation "k8s.io/kubernetes/pkg/genericapiserver/validation"
	"k8s.io/kubernetes/pkg/registry/cachesize"
	"k8s.io/kubernetes/pkg/registry/generic"
	"k8s.io/kubernetes/pkg/controller/framework/informers"
	"k8s.io/heapster/metrics/apis/metrics"
	"k8s.io/heapster/metrics/apis/metrics/v1alpha1"
)

// NewAPIServerCommand creates a *cobra.Command object with default parameters
func NewAPIServerCommand() *cobra.Command {
	s := genericoptions.NewServerRunOptions()
	s.AddUniversalFlags(pflag.CommandLine)
	cmd := &cobra.Command{
		Use: "heapster-apiserver",
		Long: `heapster apiserver`,
		Run: func(cmd *cobra.Command, args []string) {
		},
	}

	return cmd
}

// Run runs the specified APIServer.  This should never exit.
func Run(s *genericoptions.ServerRunOptions) error {
	genericvalidation.VerifyEtcdServersList(s)
	genericapiserver.DefaultAndValidateRunOptions(s)

	// TODO: register heapster resources here.
	resourceConfig := genericapiserver.NewResourceConfig()
	resourceConfig.EnableVersions(unversioned.GroupVersion{Group: metrics.GroupName, Version: "v1alpha1"})

	resourceConfig.EnableResources(v1alpha1.SchemeGroupVersion.WithResource("nodemetrics"))

	storageGroupsToEncodingVersion, err := s.StorageGroupsToEncodingVersion()
	if err != nil {
		glog.Fatalf("error generating storage version map: %s", err)
	}
	storageFactory, err := genericapiserver.BuildDefaultStorageFactory(
		s.StorageConfig, s.DefaultStorageMediaType, api.Codecs,
		genericapiserver.NewDefaultResourceEncodingConfig(), storageGroupsToEncodingVersion,
		[]unversioned.GroupVersionResource{}, resourceConfig, s.RuntimeConfig)
	if err != nil {
		glog.Fatalf("error in initializing storage factory: %s", err)
	}

	//for _, override := range s.EtcdServersOverrides {
	//	tokens := strings.Split(override, "#")
	//	if len(tokens) != 2 {
	//		glog.Errorf("invalid value of etcd server overrides: %s", override)
	//		continue
	//	}
	//
	//	apiresource := strings.Split(tokens[0], "/")
	//	if len(apiresource) != 2 {
	//		glog.Errorf("invalid resource definition: %s", tokens[0])
	//		continue
	//	}
	//	group := apiresource[0]
	//	resource := apiresource[1]
	//	groupResource := unversioned.GroupResource{Group: group, Resource: resource}
	//
	//	servers := strings.Split(tokens[1], ";")
	//	storageFactory.SetEtcdLocation(groupResource, servers)
	//}

	authz, err := authenticator.New(authenticator.AuthenticatorConfig{
		BasicAuthFile:     s.BasicAuthFile,
		ClientCAFile:      s.ClientCAFile,
		TokenAuthFile:     s.TokenAuthFile,
		OIDCIssuerURL:     s.OIDCIssuerURL,
		OIDCClientID:      s.OIDCClientID,
		OIDCCAFile:        s.OIDCCAFile,
		OIDCUsernameClaim: s.OIDCUsernameClaim,
		OIDCGroupsClaim:   s.OIDCGroupsClaim,
		KeystoneURL:       s.KeystoneURL,
	})
	if err != nil {
		glog.Fatalf("Invalid Authentication Config: %v", err)
	}

	authorizationModeNames := strings.Split(s.AuthorizationMode, ",")
	authorizationConfig := genericauthorizer.AuthorizationConfig{
		PolicyFile:                  s.AuthorizationPolicyFile,
		WebhookConfigFile:           s.AuthorizationWebhookConfigFile,
		WebhookCacheAuthorizedTTL:   s.AuthorizationWebhookCacheAuthorizedTTL,
		WebhookCacheUnauthorizedTTL: s.AuthorizationWebhookCacheUnauthorizedTTL,
		RBACSuperUser:               s.AuthorizationRBACSuperUser,
	}
	authorizer, err := genericauthorizer.NewAuthorizerFromAuthorizationConfig(authorizationModeNames, authorizationConfig)
	if err != nil {
		glog.Fatalf("Invalid Authorization Config: %v", err)
	}

	admissionControlPluginNames := strings.Split(s.AdmissionControl, ",")
	client, err := s.NewSelfClient()
	if err != nil {
		glog.Errorf("Failed to create clientset: %v", err)
	}

	sharedInformers := informers.NewSharedInformerFactory(client, 10*time.Minute)
	pluginInitializer := admission.NewPluginInitializer(sharedInformers)

	admissionController, err := admission.NewFromPlugins(client, admissionControlPluginNames, s.AdmissionControlConfigFile, pluginInitializer)

	genericConfig := genericapiserver.NewConfig(s)
	// TODO: Move the following to generic api server as well.
	genericConfig.StorageFactory = storageFactory
	genericConfig.Authenticator = authz
	genericConfig.SupportsBasicAuth = len(s.BasicAuthFile) > 0
	genericConfig.Authorizer = authorizer
	genericConfig.AdmissionControl = admissionController
	genericConfig.APIResourceConfigSource = storageFactory.APIResourceConfigSource
	genericConfig.MasterServiceNamespace = s.MasterServiceNamespace
	genericConfig.Serializer = api.Codecs

	// TODO: Move this to generic api server (Need to move the command line flag).
	if s.EnableWatchCache {
		cachesize.SetWatchCacheSizes(s.WatchCacheSizes)
	}

	m, err := genericapiserver.New(genericConfig)
	if err != nil {
		return err
	}

	installMetricsAPIs(s, m, storageFactory)

	m.Run(s)
	return nil
}

func createRESTOptionsOrDie(s *genericoptions.ServerRunOptions, g *genericapiserver.GenericAPIServer, f genericapiserver.StorageFactory, resource unversioned.GroupResource) generic.RESTOptions {
	storage, err := f.NewConfig(resource)
	if err != nil {
		glog.Fatalf("Unable to find storage destination for %v, due to %v", resource, err.Error())
	}
	return generic.RESTOptions{
		StorageConfig:                 storage,
		Decorator:               g.StorageDecorator(),
		DeleteCollectionWorkers: s.DeleteCollectionWorkers,
	}
}
