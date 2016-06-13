// Copyright 2014 Google Inc. All Rights Reserved.
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

package main

import (
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	restful "github.com/emicklei/go-restful"

	"k8s.io/heapster/metrics/util"
	"k8s.io/kubernetes/pkg/auth/authenticator"
	"k8s.io/kubernetes/pkg/auth/user"
	x509request "k8s.io/kubernetes/plugin/pkg/auth/authenticator/request/x509"
)

// authRequest attempts to authenticate and authorize a request against
// the given authenticator and authorizer.  If succesfull, it returns the
// user information from the authentication step.  A bool is returned indicating
// whether authentication and authorization were successful.
func authRequest(authn authenticator.Request, authz Authorizer, req *http.Request, resp http.ResponseWriter) (user.Info, bool) {
	// Check authn
	user, ok, err := authn.AuthenticateRequest(req)
	if err != nil {
		http.Error(resp, err.Error(), http.StatusInternalServerError)
		return nil, false
	}
	if !ok {
		http.Error(resp, "Unauthorized", http.StatusUnauthorized)
		return nil, false
	}

	// Check authz
	allowed, err := authz.AuthorizeRequest(req, user)
	if err != nil {
		http.Error(resp, err.Error(), http.StatusInternalServerError)
		return nil, false
	}
	if !allowed {
		http.Error(resp, "Forbidden", http.StatusForbidden)
		return nil, false
	}

	return user, true
}

func newAuthFilter(caFile string, allowedUsers string) (restful.FilterFunction, error) {
	// Authn/Authz setup
	authn, err := newAuthenticatorFromClientCAFile(caFile)
	if err != nil {
		return nil, err
	}

	authz, err := newAuthorizerFromUserList(strings.Split(allowedUsers, ",")...)
	if err != nil {
		return nil, err
	}

	return func(req *restful.Request, resp *restful.Response, filters *restful.FilterChain) {
		user, allowed := authRequest(authn, authz, req.Request, resp)
		if !allowed {
			return
		}

		// set the user info attribute for use later
		req.SetAttribute(util.UserAttributeName, user)
		filters.ProcessFilter(req, resp)
	}, nil
}

func newAuthHandler(handler http.Handler, caFile string, allowedUsers string) (http.Handler, error) {
	// Authn/Authz setup
	authn, err := newAuthenticatorFromClientCAFile(caFile)
	if err != nil {
		return nil, err
	}

	authz, err := newAuthorizerFromUserList(strings.Split(allowedUsers, ",")...)
	if err != nil {
		return nil, err
	}

	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		_, allowed := authRequest(authn, authz, req, w)
		if !allowed {
			return
		}
		handler.ServeHTTP(w, req)
	}), nil
}

// newAuthenticatorFromClientCAFile returns an authenticator.Request or an error
func newAuthenticatorFromClientCAFile(clientCAFile string) (authenticator.Request, error) {
	opts := x509request.DefaultVerifyOptions()

	// If at custom CA bundle is provided, load it (otherwise just use system roots)
	if len(clientCAFile) > 0 {
		if caData, err := ioutil.ReadFile(clientCAFile); err != nil {
			return nil, err
		} else if len(caData) > 0 {
			roots := x509.NewCertPool()
			if !roots.AppendCertsFromPEM(caData) {
				return nil, fmt.Errorf("no valid certs found in %s", clientCAFile)
			}
			opts.Roots = roots
		}
	}

	return x509request.New(opts, x509request.CommonNameUserConversion), nil
}

type Authorizer interface {
	AuthorizeRequest(req *http.Request, user user.Info) (bool, error)
}

func newAuthorizerFromUserList(allowedUsers ...string) (Authorizer, error) {
	if len(allowedUsers) == 1 && len(allowedUsers[0]) == 0 {
		return &allowAnyAuthorizer{}, nil
	}
	u := map[string]bool{}
	for _, allowedUser := range allowedUsers {
		u[allowedUser] = true
	}
	return &userAuthorizer{u}, nil
}

type allowAnyAuthorizer struct{}

func (a *allowAnyAuthorizer) AuthorizeRequest(req *http.Request, user user.Info) (bool, error) {
	return true, nil
}

type userAuthorizer struct {
	allowedUsers map[string]bool
}

func (a *userAuthorizer) AuthorizeRequest(req *http.Request, user user.Info) (bool, error) {
	return a.allowedUsers[user.GetName()], nil
}
