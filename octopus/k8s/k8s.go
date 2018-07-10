// Copyright 2015 Tigera Inc
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

package k8s

import (
	"strings"

	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

func NewK8sClient(kuber Kubernetes, policy Policy) (*kubernetes.Clientset, error) {
	// Some config can be passed in a kubeconfig file
	kubeconfig := kuber.Kubeconfig
	// Config can be overridden by config passed in explicitly in the network config.
	configOverrides := &clientcmd.ConfigOverrides{}

	// If an API root is given, make sure we're using using the name / port rather than
	// the full URL. Earlier versions of the config required the full `/api/v1/` extension,
	// so split that off to ensure compatibility.
	policy.K8sAPIRoot = strings.Split(policy.K8sAPIRoot, "/api/")[0]

	var overridesMap = []struct {
		variable *string
		value    string
	}{
		{&configOverrides.ClusterInfo.Server, policy.K8sAPIRoot},
		{&configOverrides.AuthInfo.ClientCertificate, policy.K8sClientCertificate},
		{&configOverrides.AuthInfo.ClientKey, policy.K8sClientKey},
		{&configOverrides.ClusterInfo.CertificateAuthority, policy.K8sCertificateAuthority},
		{&configOverrides.AuthInfo.Token, policy.K8sAuthToken},
	}

	// Using the override map above, populate any non-empty values.
	for _, override := range overridesMap {
		if override.value != "" {
			*override.variable = override.value
		}
	}

	// Also allow the K8sAPIRoot to appear under the "kubernetes" block in the network config.
	if kuber.K8sAPIRoot != "" {
		configOverrides.ClusterInfo.Server = kuber.K8sAPIRoot
	}

	// Use the kubernetes client code to load the kubeconfig file and combine it with the overrides.
	config, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		&clientcmd.ClientConfigLoadingRules{ExplicitPath: kubeconfig},
		configOverrides).ClientConfig()
	if err != nil {
		return nil, err
	}

	// Create the clientset
	return kubernetes.NewForConfig(config)
}

func GetK8sPodInfo(client *kubernetes.Clientset, podName, podNamespace string) (labels map[string]string, annotations map[string]string, err error) {
	pod, err := client.CoreV1().Pods(string(podNamespace)).Get(podName, v1.GetOptions{})
	if err != nil {
		return nil, nil, err
	}
	
	return pod.Labels, pod.Annotations, nil
}
