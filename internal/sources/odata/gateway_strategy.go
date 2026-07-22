// Copyright 2026 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package odata

import (
	"container/list"
	"context"
	"fmt"
	"net/http"
	"net/http/cookiejar"
	"sync"
	"time"
)

type UserSession struct {
	CsrfToken string
	Jar       http.CookieJar
	ExpiresAt time.Time
}

type lruItem struct {
	key     string
	session *UserSession
}

type sessionLRUCache struct {
	maxItems int
	ttl      time.Duration
	ll       *list.List
	cache    map[string]*list.Element
	mu       sync.RWMutex
}

func newSessionCache(maxItems int, ttl time.Duration) *sessionLRUCache {
	return &sessionLRUCache{
		maxItems: maxItems,
		ttl:      ttl,
		ll:       list.New(),
		cache:    make(map[string]*list.Element),
	}
}

func (c *sessionLRUCache) Get(key string) *UserSession {
	c.mu.Lock()
	defer c.mu.Unlock()

	if elem, ok := c.cache[key]; ok {
		item := elem.Value.(*lruItem)
		if time.Now().After(item.session.ExpiresAt) {
			c.removeElement(elem)
			return nil
		}
		c.ll.MoveToFront(elem)
		return item.session
	}
	return nil
}

func (c *sessionLRUCache) Set(key string, session *UserSession) {
	c.mu.Lock()
	defer c.mu.Unlock()

	session.ExpiresAt = time.Now().Add(c.ttl)

	if elem, ok := c.cache[key]; ok {
		c.ll.MoveToFront(elem)
		elem.Value.(*lruItem).session = session
		return
	}

	item := &lruItem{key: key, session: session}
	elem := c.ll.PushFront(item)
	c.cache[key] = elem

	if c.ll.Len() > c.maxItems {
		c.removeOldest()
	}
}

func (c *sessionLRUCache) Remove(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if elem, ok := c.cache[key]; ok {
		c.removeElement(elem)
	}
}

func (c *sessionLRUCache) removeElement(elem *list.Element) {
	c.ll.Remove(elem)
	item := elem.Value.(*lruItem)
	delete(c.cache, item.key)
}

func (c *sessionLRUCache) removeOldest() {
	elem := c.ll.Back()
	if elem != nil {
		c.removeElement(elem)
	}
}

// GatewayStrategy is the decorator strategy that handles CSRF token retrieval and cookie session management.
type GatewayStrategy struct {
	BaseURL             string
	DefaultHeaders      map[string]string
	Primary             AuthStrategy // Underlying credential strategy (Basic, mTLS, Bearer)
	DynamicOauth        AuthStrategy // Fallback strategy for dynamic pass-through OAuth
	AuthTokenHeaderName string       // Header name for the dynamic pass-through OAuth token
	sessionCache        *sessionLRUCache
}

func NewGatewayStrategy(baseURL string, defaultHeaders map[string]string, primary AuthStrategy, dynamicOauth AuthStrategy, authTokenHeaderName string) *GatewayStrategy {
	return &GatewayStrategy{
		BaseURL:             baseURL,
		DefaultHeaders:      defaultHeaders,
		Primary:             primary,
		DynamicOauth:        dynamicOauth,
		AuthTokenHeaderName: authTokenHeaderName,
		sessionCache:        newSessionCache(1000, 30*time.Minute),
	}
}

func (s *GatewayStrategy) Authorize(ctx context.Context, req *http.Request, client *http.Client) error {
	// 1. Apply Credentials using Dynamic OAuth or Fallback Primary
	if s.DynamicOauth != nil && req.Header.Get(s.AuthTokenHeaderName) != "" {
		if err := s.DynamicOauth.Authorize(ctx, req, client); err != nil {
			return err
		}
	} else {
		if err := s.Primary.Authorize(ctx, req, client); err != nil {
			return err
		}
	}

	sessionKey := HashCredentialsWithHeader(req, s.AuthTokenHeaderName)
	session := s.sessionCache.Get(sessionKey)

	// 2. Fetch CSRF Token and Cookies Pre-flight if modifying data and session doesn't exist
	if req.Method != "GET" && req.Method != "HEAD" {
		if session == nil {
			var err error
			session, err = s.fetchUserCsrf(ctx, req, client, sessionKey)
			if err != nil {
				return fmt.Errorf("OData gateway csrf pre-flight failed: %w", err)
			}
		}
		req.Header.Set("X-CSRF-Token", session.CsrfToken)
	}

	// 3. Inject Cached Cookies
	if session != nil && session.Jar != nil {
		cookies := session.Jar.Cookies(req.URL)
		for _, c := range cookies {
			req.AddCookie(c)
		}
	}

	return nil
}

func (s *GatewayStrategy) Evict(ctx context.Context, req *http.Request) {
	sessionKey := HashCredentialsWithHeader(req, s.AuthTokenHeaderName)
	s.sessionCache.Remove(sessionKey)
}

func (s *GatewayStrategy) fetchUserCsrf(ctx context.Context, req *http.Request, client *http.Client, sessionKey string) (*UserSession, error) {
	fetchReq, err := http.NewRequestWithContext(ctx, "HEAD", s.BaseURL, nil)
	if err != nil {
		return nil, err
	}

	// Apply credentials and headers to pre-flight request
	fetchReq.Header.Set(s.AuthTokenHeaderName, req.Header.Get(s.AuthTokenHeaderName))
	for k, v := range s.DefaultHeaders {
		fetchReq.Header.Set(k, v)
	}
	fetchReq.Header.Set("X-CSRF-Token", "Fetch")

	jar, _ := cookiejar.New(nil)
	fetchClient := &http.Client{
		Timeout:   client.Timeout,
		Transport: client.Transport,
		Jar:       jar,
	}

	resp, err := fetchClient.Do(fetchReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	token := resp.Header.Get("X-CSRF-Token")
	if token == "" || token == "Required" {
		return nil, fmt.Errorf("failed to fetch valid CSRF token. status: %d", resp.StatusCode)
	}

	session := &UserSession{
		CsrfToken: token,
		Jar:       jar,
	}

	s.sessionCache.Set(sessionKey, session)
	return session, nil
}
