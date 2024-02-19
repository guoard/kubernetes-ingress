//
// Copyright 2019 HAProxy Technologies LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Code generated by lister-gen. DO NOT EDIT.

package v1

import (
	v1 "github.com/haproxytech/kubernetes-ingress/crs/api/ingress/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/cache"
)

// DefaultsLister helps list Defaults.
// All objects returned here must be treated as read-only.
type DefaultsLister interface {
	// List lists all Defaults in the indexer.
	// Objects returned here must be treated as read-only.
	List(selector labels.Selector) (ret []*v1.Defaults, err error)
	// Defaults returns an object that can list and get Defaults.
	Defaults(namespace string) DefaultsNamespaceLister
	DefaultsListerExpansion
}

// defaultsLister implements the DefaultsLister interface.
type defaultsLister struct {
	indexer cache.Indexer
}

// NewDefaultsLister returns a new DefaultsLister.
func NewDefaultsLister(indexer cache.Indexer) DefaultsLister {
	return &defaultsLister{indexer: indexer}
}

// List lists all Defaults in the indexer.
func (s *defaultsLister) List(selector labels.Selector) (ret []*v1.Defaults, err error) {
	err = cache.ListAll(s.indexer, selector, func(m interface{}) {
		ret = append(ret, m.(*v1.Defaults))
	})
	return ret, err
}

// Defaults returns an object that can list and get Defaults.
func (s *defaultsLister) Defaults(namespace string) DefaultsNamespaceLister {
	return defaultsNamespaceLister{indexer: s.indexer, namespace: namespace}
}

// DefaultsNamespaceLister helps list and get Defaults.
// All objects returned here must be treated as read-only.
type DefaultsNamespaceLister interface {
	// List lists all Defaults in the indexer for a given namespace.
	// Objects returned here must be treated as read-only.
	List(selector labels.Selector) (ret []*v1.Defaults, err error)
	// Get retrieves the Defaults from the indexer for a given namespace and name.
	// Objects returned here must be treated as read-only.
	Get(name string) (*v1.Defaults, error)
	DefaultsNamespaceListerExpansion
}

// defaultsNamespaceLister implements the DefaultsNamespaceLister
// interface.
type defaultsNamespaceLister struct {
	indexer   cache.Indexer
	namespace string
}

// List lists all Defaults in the indexer for a given namespace.
func (s defaultsNamespaceLister) List(selector labels.Selector) (ret []*v1.Defaults, err error) {
	err = cache.ListAllByNamespace(s.indexer, s.namespace, selector, func(m interface{}) {
		ret = append(ret, m.(*v1.Defaults))
	})
	return ret, err
}

// Get retrieves the Defaults from the indexer for a given namespace and name.
func (s defaultsNamespaceLister) Get(name string) (*v1.Defaults, error) {
	obj, exists, err := s.indexer.GetByKey(s.namespace + "/" + name)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.NewNotFound(v1.Resource("defaults"), name)
	}
	return obj.(*v1.Defaults), nil
}